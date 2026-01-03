package shortener

import (
	"crypto/rand"
	"math/big"
)

// Base62 characters for short code generation.
const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// DefaultCodeLength is the default length for generated short codes.
const DefaultCodeLength = 7

// GenerateShortCode generates a random short code of the specified length.
func GenerateShortCode(length int) (string, error) {
	if length <= 0 {
		length = DefaultCodeLength
	}

	code := make([]byte, length)
	max := big.NewInt(int64(len(base62Chars)))

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		code[i] = base62Chars[n.Int64()]
	}

	return string(code), nil
}

// IsValidShortCode checks if a short code is valid (alphanumeric only).
func IsValidShortCode(code string) bool {
	if code == "" || len(code) > 20 {
		return false
	}

	for _, c := range code {
		if !isAlphanumeric(c) {
			return false
		}
	}
	return true
}

func isAlphanumeric(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}
