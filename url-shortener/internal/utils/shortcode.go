package utils

import (
	"crypto/rand"
	"math/big"
	"strings"
)

const (
	// DefaultCodeLength is the default length of generated short codes
	DefaultCodeLength = 6

	// Charset contains characters used for generating short codes
	// Excludes ambiguous characters like 0, O, I, l for better readability
	Charset = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789"
)

// GenerateShortCode generates a random short code of the specified length
func GenerateShortCode(length int) (string, error) {
	if length <= 0 {
		length = DefaultCodeLength
	}

	var sb strings.Builder
	sb.Grow(length)

	charsetLength := big.NewInt(int64(len(Charset)))

	for i := 0; i < length; i++ {
		randomIndex, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			return "", err
		}
		sb.WriteByte(Charset[randomIndex.Int64()])
	}

	return sb.String(), nil
}

// GenerateDefaultShortCode generates a short code with the default length
func GenerateDefaultShortCode() (string, error) {
	return GenerateShortCode(DefaultCodeLength)
}

// IsValidShortCode validates if a string is a valid short code
func IsValidShortCode(code string) bool {
	if len(code) < 1 || len(code) > 10 {
		return false
	}

	for _, char := range code {
		if !strings.ContainsRune(Charset, char) {
			return false
		}
	}

	return true
}
