package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// URL represents a shortened URL in the system
type URL struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	ShortCode   string    `gorm:"type:varchar(10);uniqueIndex;not null" json:"short_code"`
	OriginalURL string    `gorm:"type:text;not null" json:"original_url"`
	UserID      uuid.UUID `gorm:"type:uuid;index;not null" json:"user_id"`
	User        User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ClickCount  int64     `gorm:"default:0" json:"click_count"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null" json:"updated_at"`
}

// BeforeCreate hook to set UUID before creating a new URL
func (u *URL) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// TableName returns the table name for the URL model
func (URL) TableName() string {
	return "urls"
}

// CreateURLRequest represents the request body for creating a URL
type CreateURLRequest struct {
	OriginalURL string `json:"original_url" validate:"required,url"`
}

// URLResponse represents the response for a URL
type URLResponse struct {
	ID          uuid.UUID     `json:"id"`
	ShortCode   string        `json:"short_code"`
	ShortURL    string        `json:"short_url"`
	OriginalURL string        `json:"original_url"`
	ClickCount  int64         `json:"click_count"`
	IsActive    bool          `json:"is_active"`
	CreatedAt   time.Time     `json:"created_at"`
	User        *UserResponse `json:"user,omitempty"`
}

// ToResponse converts URL model to URLResponse
func (u *URL) ToResponse(baseURL string) URLResponse {
	response := URLResponse{
		ID:          u.ID,
		ShortCode:   u.ShortCode,
		ShortURL:    baseURL + "/" + u.ShortCode,
		OriginalURL: u.OriginalURL,
		ClickCount:  u.ClickCount,
		IsActive:    u.IsActive,
		CreatedAt:   u.CreatedAt,
	}

	// Include user info if loaded
	if u.User.ID != uuid.Nil {
		userResponse := u.User.ToResponse()
		response.User = &userResponse
	}

	return response
}
