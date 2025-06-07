package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"github.com/hayohtee/books/internal/cache"
	"github.com/hayohtee/books/internal/data"
	"github.com/hayohtee/books/internal/mailer"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"os"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Read the value of the config fields from the command-line flags
	var cfg config
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("DATABASE_DSN"), "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conn", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conn", 25, "PostgreSQL max idle connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max idle time")

	flag.StringVar(&cfg.redisDSN, "redis-dsn", os.Getenv("REDIS_DSN"), "Redis DSN")

	flag.StringVar(&cfg.smtp.host, "smtp-host", os.Getenv("SMTP_HOST"), "SMTP host")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", os.Getenv("SMTP_SENDER"), "SMTP Sender")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 465, "SMTP port")
	flag.StringVar(&cfg.smtp.password, "smtp-password", os.Getenv("SMTP_PASSWORD"), "SMTP password")
	flag.StringVar(&cfg.smtp.username, "smtp-username", os.Getenv("SMTP_USERNAME"), "SMTP username")
	flag.Parse()

	mailClient, err := mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.sender, cfg.smtp.username, cfg.smtp.password)
	if err != nil {
		logger.Error(fmt.Sprintf("error creating mail client: %v", err))
		os.Exit(1)
	}
	logger.Info("mail client created")

	db, err := openDB(cfg)
	if err != nil {
		logger.Error(fmt.Sprintf("error opening database: %v", err))
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connection pool established")

	redisClient, err := openRedis(cfg.redisDSN)
	if err != nil {
		logger.Error(fmt.Sprintf("error creating redis client: %v", err))
		os.Exit(1)
	}
	defer redisClient.Close()
	logger.Info("redis connection established")

	// Declare an instance of the application struct
	app := &application{
		cfg:     cfg,
		logger:  logger,
		queries: data.New(db),
		mailer:  mailClient,
		cache:   cache.New(redisClient),
	}

	if err := app.serve(); err != nil {
		logger.Error(fmt.Sprintf("error starting server: %v", err))
		os.Exit(1)
	}
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)
	db.SetConnMaxIdleTime(cfg.db.maxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func openRedis(dsn string) (*redis.Client, error) {
	opt, err := redis.ParseURL(dsn)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.Ping(ctx).Result(); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}
