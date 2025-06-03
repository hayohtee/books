package cache

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

const (
	EmailOTPScope    = "email_otp"
	PasswordOTPScope = "password_otp"
)

type OTP struct {
	Code      string    `redis:"code"`
	Email     string    `redis:"email"`
	UserID    string    `redis:"user_id"`
	Scope     string    `redis:"scope"`
	ExpiresAt time.Time `redis:"expires_at"`
}

func (c *Cache) NewOTP(userID, email, scope string, ttl time.Duration) (OTP, error) {
	otpCode, err := generateOTP()
	if err != nil {
		return OTP{}, err
	}

	otp := OTP{
		Code:      otpCode,
		Scope:     scope,
		Email:     email,
		ExpiresAt: time.Now().Add(ttl),
		UserID:    userID,
	}

	if err = c.InsertOTP(otp); err != nil {
		return OTP{}, err
	}

	return otp, nil
}

func (c *Cache) InsertOTP(otp OTP) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := c.client.HSet(ctx, fmt.Sprintf("%s:%s", otp.Email, otp.Scope), otp).Err()
	if err != nil {
		return err
	}

	return c.client.ExpireAt(ctx, fmt.Sprintf("%s:%s", otp.Email, otp.Scope), otp.ExpiresAt).Err()
}

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
