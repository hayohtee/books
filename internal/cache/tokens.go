package cache

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"github.com/google/uuid"
	"time"
)

const (
	AccessTokenScope  = "access_token"
	RefreshTokenScope = "refresh_token"
)

type Token struct {
	UserID    string    `redis:"user_id"`
	ExpiresAt time.Time `redis:"expires_at"`
	Scope     string    `redis:"scope"`
	PlainText string    `redis:"token"`
}

func generateOpaqueToken() (string, error) {
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes), nil
}

func (c *Cache) NewToken(userID uuid.UUID, ttl time.Duration, scope string) (Token, error) {
	opaqueToken, err := generateOpaqueToken()
	if err != nil {
		return Token{}, err
	}

	token := Token{
		UserID:    userID.String(),
		ExpiresAt: time.Now().Add(ttl),
		Scope:     scope,
		PlainText: opaqueToken,
	}

	if err = c.InsertToken(token); err != nil {
		return Token{}, err
	}

	return token, nil
}

func (c *Cache) InsertToken(token Token) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := c.client.HSet(ctx, fmt.Sprintf("%s:%s", token.Scope, token.PlainText), token).Err()
	if err != nil {
		return err
	}

	err = c.client.ExpireAt(ctx, fmt.Sprintf("%s:%s", token.Scope, token.PlainText), token.ExpiresAt).Err()
	if err != nil {
		return err
	}

	return nil
}

func (c *Cache) GetToken(scope, plainText string) (Token, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var token Token

	value := c.client.HGetAll(ctx, fmt.Sprintf("%s:%s", scope, plainText))
	if err := value.Err(); err != nil {
		return Token{}, err
	}

	res, err := value.Result()
	if err != nil {
		return Token{}, err
	}

	if len(res) == 0 {
		return Token{}, ErrRecordNotFound
	}

	if err := value.Scan(&token); err != nil {
		return Token{}, err
	}

	return token, nil
}

func (c *Cache) DeleteToken(scope, plainText string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.client.HDel(ctx, fmt.Sprintf("%s:%s", scope, plainText)).Err()
}
