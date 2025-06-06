package cache

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/google/uuid"
	"math/big"
	"time"
)

type VerificationData struct {
	UserID   string    `redis:"user_id"`
	Code     string    `redis:"code"`
	Email    string    `redis:"email"`
	ExpireAt time.Time `redis:"expire_at"`
}

func (c *Cache) NewVerificationData(userID uuid.UUID, email string, ttl time.Duration) (VerificationData, error) {
	otpCode, err := generateCode()
	if err != nil {
		return VerificationData{}, err
	}

	v := VerificationData{
		UserID:   userID.String(),
		Code:     otpCode,
		Email:    email,
		ExpireAt: time.Now().Add(ttl),
	}

	if err = c.InsertVerificationData(v); err != nil {
		return VerificationData{}, err
	}

	return v, nil
}

func (c *Cache) InsertVerificationData(v VerificationData) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := c.client.HSet(ctx, fmt.Sprintf("%s:verification_code", v.Email), v).Err()
	if err != nil {
		return err
	}

	return c.client.ExpireAt(ctx, fmt.Sprintf("%s:verification_code", v.Email), v.ExpireAt).Err()
}

func (c *Cache) GetVerificationData(email string) (VerificationData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var verificationCode VerificationData

	value := c.client.HGetAll(ctx, fmt.Sprintf("%s:verification_code", email))
	if err := value.Err(); err != nil {
		return VerificationData{}, err
	}

	res, err := value.Result()
	if err != nil {
		return VerificationData{}, err
	}

	if len(res) == 0 {
		return VerificationData{}, ErrRecordNotFound
	}

	if err = value.Scan(&verificationCode); err != nil {
		return VerificationData{}, err
	}

	return verificationCode, nil
}

func (c *Cache) DeleteVerificationData(email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.client.Del(ctx, fmt.Sprintf("%s:verification_code", email)).Err()
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
