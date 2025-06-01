package main

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
)

// generatePasswordHash calculates and return the bycrypt hash of plaintext password.
func generatePasswordHash(plaintext string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(plaintext), 12)
}

// Matches checks whether the provided plaintext password matches the
// hashed password, returning true if it matches and false otherwise.
func passwordMatches(plaintext string, hash []byte) (bool, error) {
	err := bcrypt.CompareHashAndPassword(hash, []byte(plaintext))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err

		}
	}
	return true, nil
}
