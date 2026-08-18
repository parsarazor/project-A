package ports

import (
	"github.com/google/uuid"
	"github.com/url-shortener/internal/model"
)

type UserRepository interface {
	Create(user *model.User) error
	FindByID(id uuid.UUID) (*model.User, error)
	FindByEmail(email string) (*model.User, error)
	Update(user *model.User) error
	Delete(id uuid.UUID) error
	FindAll(page, pageSize int) ([]model.User, int64, error)
	ExistsByEmail(email string) (bool, error)
	UpdatePassword(id uuid.UUID, passwordHash string) error
	UpdateRole(id uuid.UUID, role model.Role) error
}
