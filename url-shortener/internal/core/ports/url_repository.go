package ports

import (
	"github.com/google/uuid"
	"github.com/url-shortener/internal/model"
)

type URLRepository interface {
	Create(url *model.URL) error
	FindByID(id uuid.UUID) (*model.URL, error)
	FindByIDWithUser(id uuid.UUID) (*model.URL, error)
	FindByShortCode(shortCode string) (*model.URL, error)
	FindByUserID(userID uuid.UUID) ([]model.URL, error)
	FindAll(page, pageSize int) ([]model.URL, int64, error)
	FindAllWithUser() ([]model.URL, error)
	Update(url *model.URL) error
	Delete(id uuid.UUID) error
	IncrementClickCount(id uuid.UUID) error
	IncrementClickCountByShortCode(shortCode string) error
	Deactivate(id uuid.UUID) error
	Activate(id uuid.UUID) error
	CountByUserID(userID uuid.UUID) (int64, error)
	GetTotalClicks(userID uuid.UUID) (int64, error)
}
