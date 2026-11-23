package auth

import (
	"context"
	"errors"
)

const IdentityClaim = "easy-chat-uid"

func UserID(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(IdentityClaim).(string)
	if !ok || userID == "" {
		return "", errors.New("authenticated user id is missing")
	}

	return userID, nil
}
