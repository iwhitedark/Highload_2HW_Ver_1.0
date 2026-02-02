package models

import (
	"errors"
	"regexp"
	"strings"
)

// User represents a user entity in the system
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// emailRegex is a compiled regular expression for email validation
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Validate validates the user data
func (u *User) Validate() error {
	if strings.TrimSpace(u.Name) == "" {
		return errors.New("name is required")
	}
	if len(u.Name) < 2 {
		return errors.New("name must be at least 2 characters")
	}
	if len(u.Name) > 100 {
		return errors.New("name must not exceed 100 characters")
	}
	if strings.TrimSpace(u.Email) == "" {
		return errors.New("email is required")
	}
	if !emailRegex.MatchString(u.Email) {
		return errors.New("invalid email format")
	}
	return nil
}

// Sanitize cleans and normalizes user data
func (u *User) Sanitize() {
	u.Name = strings.TrimSpace(u.Name)
	u.Email = strings.TrimSpace(strings.ToLower(u.Email))
}
