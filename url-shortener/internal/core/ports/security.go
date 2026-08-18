package ports

import (
	"github.com/google/uuid"
	"github.com/url-shortener/internal/model"
)

type TokenClaims struct {
	UserID uuid.UUID
	Email  string
	Role   model.Role
}

type TokenManager interface {
	Generate(user *model.User) (token string, expiresIn int64, err error)
	Validate(token string) (*TokenClaims, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) bool
}
