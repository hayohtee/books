package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	// Create a shutdownError channel for receiving any errors returned
	// by the graceful Shutdown() function.
	shutdownError := make(chan error)

	// Start a background goroutine.
	go func() {
		// Create a quit channel which carries os.Signal values
		quit := make(chan os.Signal, 1)

		// Use signal.Notify() to listen for incoming SIGINT and SIGTERM signals
		// and relay them to the quit channel.
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		// Read the signal from the quit channel. This will block until a signal is received.
		s := <-quit

		// Log a message to say that the server is shutting down.
		app.logger.Info("shutting down server", slog.String("signal", s.String()))

		// Create a context with a 30-seconds timeout.
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Call Shutdown() on the server, passing in the context. The Shutdown() will return
		// nil if the graceful shutdown was successful, or an error (which may happen because of
		// a problem closing the listeners, or because the shutdown did not complete before the
		// 30-second context deadline is hit). Then relay the returned value to the shutdownError channel.
		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		app.logger.Info("completing background tasks", slog.String("addr", srv.Addr))

		app.wg.Wait()
		shutdownError <- nil
	}()

	app.logger.Info("starting server", slog.String("addr", srv.Addr))

	// Calling Shutdown() on the server will cause ListenAndServe() to immediately return a http.ErrServerClosed
	// error. So if we see this error, it is actually an indication that the graceful shutdown has started.
	// So we check specifically for this and only return the error if it is not http.ErrServerClosed.
	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	// Otherwise, we wait to receive the return value from Shutdown() on the shutdownErr channel.
	// If the returned value is an error, we know that there is a problem with the graceful shutdown,
	// and we return the error.
	err = <-shutdownError
	if err != nil {
		return err
	}

	// At this point we know the graceful shutdown completed successfully, and we
	// log a "stopped server" message
	app.logger.Info("server stopped", "addr", srv.Addr)
	return nil
}
