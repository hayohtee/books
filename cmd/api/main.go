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
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"os"
	"strconv"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := godotenv.Load(); err != nil {
		logger.Error(fmt.Sprintf("error loading .env file: %v", err))
		os.Exit(1)
	}

	// Read the value of the config fields from the command-line flags
	var cfg config
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("DATABASE_DSN"), "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conn", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conn", 25, "PostgreSQL max idle connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max idle time")
	flag.StringVar(&cfg.redis.addr, "redis-addr", "localhost:6379", "Redis server address")
	flag.StringVar(&cfg.redis.password, "redis-password", os.Getenv("REDIS_PASSWORD"), "Redis server password")
	flag.IntVar(&cfg.redis.db, "redis-db", 0, "Redis server database")
	flag.Parse()

	// Read the configurations for smtp from environment variables
	cfg.smtp.host = os.Getenv("SMTP_HOST")
	cfg.smtp.sender = os.Getenv("SMTP_SENDER")
	cfg.smtp.password = os.Getenv("SMTP_PASSWORD")
	cfg.smtp.username = os.Getenv("SMTP_USERNAME")
	port, err := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil {
		panic(err)
	}
	cfg.smtp.port = port
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

	redisClient, err := openRedis(cfg.redis.addr, cfg.redis.password, cfg.redis.db)
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

func openRedis(addr, password string, db int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.Ping(ctx).Result(); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}
