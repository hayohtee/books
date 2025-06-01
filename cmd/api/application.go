package main

// config struct holds the configuration settings for the application.
type config struct {
	// the port to listen on
	port int
	// the configuration settings for database connection pool.
	db struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
	// the configurations settings for smtp.
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	// the configuration settings for redis.
	redis struct {
		addr     string
		password string
		db       int
	}
}
