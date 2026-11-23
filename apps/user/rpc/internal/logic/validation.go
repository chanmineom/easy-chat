package logic

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

var phonePattern = regexp.MustCompile(`^\+?[0-9]{6,20}$`)

func normalizePhone(phone string) (string, error) {
	phone = strings.TrimSpace(phone)
	if !phonePattern.MatchString(phone) {
		return "", fmt.Errorf("phone must contain 6-20 digits and may start with +")
	}
	return phone, nil
}

func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must contain at least 8 bytes")
	}
	if len(password) > 72 {
		return fmt.Errorf("password must not exceed 72 bytes")
	}
	return nil
}

func normalizeNickname(nickname string) (string, error) {
	nickname = strings.TrimSpace(nickname)
	length := utf8.RuneCountInString(nickname)
	if length < 1 || length > 32 {
		return "", fmt.Errorf(
			"nickname must contain 1-32 characters",
		)
	}
	return nickname, nil
}

func validateAvatarURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if len(value) > 512 {
		return "", fmt.Errorf("avatar url is too long")
	}

	parsed, err := url.ParseRequestURI(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf(
			"avatar url must be an http or https URL",
		)
	}

	return value, nil
}
