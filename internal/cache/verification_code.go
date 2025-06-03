package cache

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

func (c *Cache) NewEmailVerificationCode(email string, ttl time.Duration) (string, error) {
	otpCode, err := generateCode()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = c.client.Set(ctx, fmt.Sprintf("%s:email_otp", email), otpCode, ttl).Err()
	if err != nil {
		return "", err
	}

	return otpCode, nil
}

func (c *Cache) GetEmailVerificationCode(email string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	value := c.client.Get(ctx, fmt.Sprintf("%s:email_otp", email))
	if err := value.Err(); err != nil {
		return "", err
	}

	res, err := value.Result()
	if err != nil {
		return "", err
	}

	if len(res) == 0 {
		return "", ErrRecordNotFound
	}

	return res, nil
}

// generateCode generates a 6-digits verification code.
//
// It generates a 6-digit OTP using a cryptographically secure random number generator.
// The OTP is composed of digits from 0 to 9.
//
// Returns the generated OTP as a string, or an error if there is a problem generating the OTP.
func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
