package main

import (
	"errors"
	"fmt"
	"github.com/hayohtee/books/internal/cache"
	"net/http"
	"strings"
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

func (app *application) requireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Value(BearerAuthScopes).([]string); ok {
			authHeader := r.Header.Get("Authorization")
			// Check for the Authorization header.
			if authHeader == "" {
				app.missingAuthorizationHeaderResponse(w, r)
				return
			}
			if !strings.HasPrefix(authHeader, "Bearer ") {
				app.invalidAuthenticationTokenResponse(w, r)
				return
			}
			bearerToken := strings.TrimPrefix(authHeader, "Bearer ")
			tokenData, err := app.cache.GetToken(cache.AccessTokenScope, bearerToken)
			if err != nil {
				switch {
				case errors.Is(err, cache.ErrRecordNotFound):
					app.errorResponse(w, r, http.StatusUnauthorized, Error{Message: "invalid or expired bearer token"})
				default:
					app.serverError(w, r, err)
				}
				return
			}
			r = app.contextWithUserID(r, tokenData.UserID)
		}
		next.ServeHTTP(w, r)
	})
}

func (app *application) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Add("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
