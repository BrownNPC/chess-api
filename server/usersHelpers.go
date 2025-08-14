package server

import (
	"errors"
	"fmt"
	"regexp"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]*$`)

func ValidatePassword(password string) error {
	if len([]rune(password)) < 3 {
		return fmt.Errorf("password must be at least 3 characters")
	}
	return nil
}

const INVALID_USERNAME_ERROR = "username can only contain letters, numbers, and underscores"

func ValidateUsername(username string) error {
	length := len([]rune(username))
	if length < 3 {
		return errors.New("username must be at least 3 characters long")
	}
	if length > 20 {
		return errors.New("username cannot be longer than 20 characters")
	}
	if !usernameRegex.MatchString(username) {
		return errors.New(INVALID_USERNAME_ERROR)
	}
	return nil
}
