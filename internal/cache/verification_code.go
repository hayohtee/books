package cache

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/google/uuid"
	"math/big"
	"time"
)

type VerificationCode struct {
	UserID   string    `redis:"user_id"`
	Code     string    `redis:"code"`
	Email    string    `redis:"email"`
	ExpireAt time.Time `redis:"expire_at"`
}

func (c *Cache) NewVerificationCode(userID uuid.UUID, email string, ttl time.Duration) (VerificationCode, error) {
	otpCode, err := generateCode()
	if err != nil {
		return VerificationCode{}, err
	}

	v := VerificationCode{
		UserID:   userID.String(),
		Code:     otpCode,
		Email:    email,
		ExpireAt: time.Now().Add(ttl),
	}

	if err = c.InsertVerificationCode(v); err != nil {
		return VerificationCode{}, err
	}

	return v, nil
}

func (c *Cache) InsertVerificationCode(v VerificationCode) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := c.client.HSet(ctx, fmt.Sprintf("%s:verification_code", v.Email), v).Err()
	if err != nil {
		return err
	}

	return c.client.ExpireAt(ctx, fmt.Sprintf("%s:verification_code", v.Email), v.ExpireAt).Err()
}

func (c *Cache) GetVerificationCode(email string) (VerificationCode, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	value := c.client.Get(ctx, fmt.Sprintf("%s:verification_code", email))
	if err := value.Err(); err != nil {
		return VerificationCode{}, err
	}

	res, err := value.Result()
	if err != nil {
		return VerificationCode{}, err
	}

	if len(res) == 0 {
		return VerificationCode{}, ErrRecordNotFound
	}

	var verificationCode VerificationCode
	if err = value.Scan(&verificationCode); err != nil {
		return VerificationCode{}, err
	}

	return verificationCode, nil
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
