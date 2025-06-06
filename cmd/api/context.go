package main

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"net/http"
)

// Define a custom contextKey type, with the underlying type string.
type contextKey string

// Represent a key for storing and retrieving userID in the context.
const userIDContextKey = contextKey("user-id")

func (app *application) contextWithUserID(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), userIDContextKey, userID)
	return r.WithContext(ctx)
}

func (app *application) contextGetUserID(r *http.Request) (uuid.UUID, error) {
	userIDStr, ok := r.Context().Value(userIDContextKey).(string)
	if !ok {
		return uuid.Nil, errors.New("no user id found in request context")
	}
	return uuid.Parse(userIDStr)
}
