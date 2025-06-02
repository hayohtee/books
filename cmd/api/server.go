package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

func (app *application) serve() error {
	mux := http.NewServeMux()
	srv := http.Server{
		Addr:         fmt.Sprintf(":%d", app.cfg.port),
		Handler:      HandlerFromMux(app, mux),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
	}

	return srv.ListenAndServe()
}
