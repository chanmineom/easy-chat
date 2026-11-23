package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func main() {
	userID := flag.String("uid", "", "user id")
	secret := flag.String("secret", "", "jwt secret")
	ttl := flag.Duration("ttl", time.Hour, "token lifetime")
	flag.Parse()

	now := time.Now()
	claims := jwt.MapClaims{
		"easy-chat-uid": *userID,
		"iat":           now.Unix(),
		"exp":           now.Add(*ttl).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(*secret))
	if err != nil {
		panic(err)
	}

	fmt.Print(signed)
}