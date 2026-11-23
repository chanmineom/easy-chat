package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const IdentityClaim = "easy-chat-uid"

type TokenResult struct {
	Token  string
	Expire int64
}

func GenerateToken(secret string, userID string, expireSeconds int64, now time.Time) (TokenResult, error) {
	if secret == "" {
		return TokenResult{}, fmt.Errorf("jwt secret is required")
	}
	if userID == "" {
		return TokenResult{}, fmt.Errorf("user id is required")
	}
	if expireSeconds <= 0 {
		return TokenResult{}, fmt.Errorf("expire must be positive")
	}

	expire := now.Add(time.Duration(expireSeconds) * time.Second)

	claims := jwt.MapClaims{
		IdentityClaim: userID,
		"iat": now.Unix(),
		"exp": expire.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return TokenResult{}, fmt.Errorf("sign jwt: %w", err)
	}

	return TokenResult{
		Token: signed,
		Expire: expire.Unix(),
	}, nil
}