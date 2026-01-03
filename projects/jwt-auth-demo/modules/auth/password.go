package auth

import (
	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultBcryptCost is the default cost for bcrypt hashing.
	// A cost of 12 provides good security while keeping hashing time reasonable.
	DefaultBcryptCost = 12
)

// PasswordHasher provides password hashing and verification functionality.
type PasswordHasher struct {
	cost int
}

// NewPasswordHasher creates a new PasswordHasher with default cost.
func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{
		cost: DefaultBcryptCost,
	}
}

// Hash generates a bcrypt hash of the given password.
func (h *PasswordHasher) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Verify checks if the provided password matches the hash.
func (h *PasswordHasher) Verify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
