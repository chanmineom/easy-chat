package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

func DirectConversationID(userA string, userB string) (string, string, error) {
	userA = strings.TrimSpace(userA)
	userB = strings.TrimSpace(userB)

	if userA == "" || userB == "" {
		return "", "", fmt.Errorf("both user ids are required")
	}
	if userA == userB {
		return "", "", fmt.Errorf(
			"a direct conversation requires two different users",
		)
	}

	users := []string{userA, userB}
	sort.Strings(users)

	sum := sha256.Sum256([]byte(users[0] + "\x00" + users[1]))
	directKey := hex.EncodeToString(sum[:16])

	return "dc_" + directKey, directKey, nil
}