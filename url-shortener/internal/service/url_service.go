package service

import (
	"errors"

	"github.com/google/uuid"
	"github.com/url-shortener/internal/model"
	"github.com/url-shortener/internal/repository"
	"github.com/url-shortener/internal/utils"
	"gorm.io/gorm"
)

// URLService errors
var (
	ErrURLNotFound     = errors.New("URL not found")
	ErrInvalidURL      = errors.New("invalid URL")
	ErrShortCodeExists = errors.New("short code already exists")
	ErrUnauthorized    = errors.New("unauthorized to access this URL")
)

// URLService handles URL shortening business logic
type URLService struct {
	urlRepo *repository.URLRepository
	baseURL string
}

// NewURLService creates a new URLService instance
func NewURLService(urlRepo *repository.URLRepository, baseURL string) *URLService {
	return &URLService{
		urlRepo: urlRepo,
		baseURL: baseURL,
	}
}

// CreateShortURL creates a new shortened URL
func (s *URLService) CreateShortURL(userID uuid.UUID, originalURL string) (*model.URL, error) {
	// Validate URL
	if !utils.IsValidURL(originalURL) {
		return nil, ErrInvalidURL
	}

	// Generate a unique short code
	var shortCode string
	var err error
	maxAttempts := 10

	for i := 0; i < maxAttempts; i++ {
		shortCode, err = utils.GenerateDefaultShortCode()
		if err != nil {
			return nil, err
		}

		exists, err := s.urlRepo.ShortCodeExists(shortCode)
		if err != nil {
			return nil, err
		}

		if !exists {
			break
		}

		if i == maxAttempts-1 {
			return nil, ErrShortCodeExists
		}
	}

	// Create the URL record
	url := &model.URL{
		ShortCode:   shortCode,
		OriginalURL: originalURL,
		UserID:      userID,
		ClickCount:  0,
		IsActive:    true,
	}

	if err := s.urlRepo.Create(url); err != nil {
		return nil, err
	}

	return url, nil
}

// GetURLByShortCode retrieves a URL by its short code
func (s *URLService) GetURLByShortCode(shortCode string) (*model.URL, error) {
	url, err := s.urlRepo.FindByShortCode(shortCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrURLNotFound
		}
		return nil, err
	}

	if !url.IsActive {
		return nil, ErrURLNotFound
	}

	return url, nil
}

// GetURLByID retrieves a URL by its ID
func (s *URLService) GetURLByID(id uuid.UUID) (*model.URL, error) {
	url, err := s.urlRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrURLNotFound
		}
		return nil, err
	}
	return url, nil
}

// GetUserURLs retrieves all URLs for a specific user
func (s *URLService) GetUserURLs(userID uuid.UUID) ([]model.URL, error) {
	return s.urlRepo.FindByUserID(userID)
}

// GetAllURLs retrieves all URLs with pagination (admin only)
func (s *URLService) GetAllURLs(page, pageSize int) ([]model.URL, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.urlRepo.FindAll(page, pageSize)
}

// GetAllURLsWithUser retrieves all URLs with user information (admin only)
func (s *URLService) GetAllURLsWithUser() ([]model.URL, error) {
	return s.urlRepo.FindAllWithUser()
}

// DeleteURL deletes a URL by ID
func (s *URLService) DeleteURL(id uuid.UUID, userID uuid.UUID, isAdmin bool) error {
	url, err := s.urlRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrURLNotFound
		}
		return err
	}

	// Check authorization: only owner or admin can delete
	if !isAdmin && url.UserID != userID {
		return ErrUnauthorized
	}

	return s.urlRepo.Delete(id)
}

// DeleteURLByAdmin deletes a URL by ID (admin only, no ownership check)
func (s *URLService) DeleteURLByAdmin(id uuid.UUID) error {
	url, err := s.urlRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrURLNotFound
		}
		return err
	}

	return s.urlRepo.Delete(url.ID)
}

// IncrementClickCount increments the click count for a URL
func (s *URLService) IncrementClickCount(shortCode string) error {
	return s.urlRepo.IncrementClickCountByShortCode(shortCode)
}

// IncrementClickCountAsync increments click count asynchronously
// This is the "Zero Network Overhead" approach - increment happens in a goroutine
func (s *URLService) IncrementClickCountAsync(shortCode string) {
	go func() {
		_ = s.urlRepo.IncrementClickCountByShortCode(shortCode)
	}()
}

// DeactivateURL deactivates a URL without deleting it
func (s *URLService) DeactivateURL(id uuid.UUID, userID uuid.UUID, isAdmin bool) error {
	url, err := s.urlRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrURLNotFound
		}
		return err
	}

	// Check authorization
	if !isAdmin && url.UserID != userID {
		return ErrUnauthorized
	}

	return s.urlRepo.Deactivate(id)
}

// ActivateURL activates a previously deactivated URL
func (s *URLService) ActivateURL(id uuid.UUID, userID uuid.UUID, isAdmin bool) error {
	url, err := s.urlRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrURLNotFound
		}
		return err
	}

	// Check authorization
	if !isAdmin && url.UserID != userID {
		return ErrUnauthorized
	}

	return s.urlRepo.Activate(id)
}

// GetUserStats returns statistics for a user's URLs
func (s *URLService) GetUserStats(userID uuid.UUID) (*UserURLStats, error) {
	urlCount, err := s.urlRepo.CountByUserID(userID)
	if err != nil {
		return nil, err
	}

	totalClicks, err := s.urlRepo.GetTotalClicks(userID)
	if err != nil {
		return nil, err
	}

	return &UserURLStats{
		TotalURLs:   urlCount,
		TotalClicks: totalClicks,
	}, nil
}

// UserURLStats represents statistics for a user's URLs
type UserURLStats struct {
	TotalURLs   int64 `json:"total_urls"`
	TotalClicks int64 `json:"total_clicks"`
}

// GetBaseURL returns the base URL for constructing short URLs
func (s *URLService) GetBaseURL() string {
	return s.baseURL
}

// ToResponse converts a URL model to its response representation
func (s *URLService) ToResponse(url *model.URL) model.URLResponse {
	return url.ToResponse(s.baseURL)
}

// ToResponses converts multiple URL models to their response representations
func (s *URLService) ToResponses(urls []model.URL) []model.URLResponse {
	responses := make([]model.URLResponse, len(urls))
	for i, url := range urls {
		responses[i] = url.ToResponse(s.baseURL)
	}
	return responses
}
