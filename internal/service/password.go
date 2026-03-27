package service

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// PasswordService handles password hashing and verification
type PasswordService struct {
	cost int
}

// NewPasswordService creates a new password service
func NewPasswordService(cost int) *PasswordService {
	if cost < 4 || cost > 31 {
		cost = 12 // Default cost
	}
	return &PasswordService{
		cost: cost,
	}
}

// Hash generates a bcrypt hash from a plain password
func (s *PasswordService) Hash(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// Verify compares a plain password with a hashed password
func (s *PasswordService) Verify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidatePassword checks if a password meets minimum requirements
func (s *PasswordService) ValidatePassword(password string) error {
	if password == "" {
		return errors.New("password cannot be empty")
	}
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	return nil
}

// MustUpdateHash checks if the hash needs to be updated (e.g., bcrypt cost changed)
func (s *PasswordService) MustUpdateHash(hash string) bool {
	// Extract the cost from the hash
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return true
	}
	return cost != s.cost
}
