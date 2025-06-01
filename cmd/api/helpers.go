package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// generateOTP generates a one-time password (OTP) consisting of numeric digits.
//
// It generates a 6-digit OTP using a cryptographically secure random number generator.
// The OTP is composed of digits from 0 to 9.
//
// Returns the generated OTP as a string, or an error if there is a problem generating the OTP.
func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
