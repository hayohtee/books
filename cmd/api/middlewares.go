package main

import (
	"fmt"
	"net/http"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a deferred function which always run in the event
		// of panic as Go unwinds the stack.
		defer func() {
			// Use the built-in recover function to check if
			// there has been a panic or not.
			if err := recover(); err != nil {
				// If there was a panic, set "Connection: close" header
				// on the response. This acts as a trigger to make
				// Go's HTTP server automatically close the current connection
				// after a response has been sent
				w.Header().Set("Connection", "close")

				// The value returned by recover is a type of any, so we use
				// fmt.Errorf to normalize it into an error and call
				// app.serverErrorResponse method.
				app.serverError(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}
