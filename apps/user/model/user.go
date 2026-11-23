package model

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type User struct {
	ID           string    `db:"id"`
	Phone        string    `db:"phone"`
	PasswordHash string    `db:"password_hash"`
	Nickname     string    `db:"nickname"`
	AvatarURL    string    `db:"avatar_url"`
	Status       int       `db:"status"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

func NewUserID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return "u_" + hex.EncodeToString(value[:]), nil
}