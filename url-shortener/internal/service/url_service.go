package service

import (
	"github.com/google/uuid"
	"github.com/url-shortener/internal/adapter/shortcode"
	"github.com/url-shortener/internal/core/usecase"
	"github.com/url-shortener/internal/model"
	"github.com/url-shortener/internal/repository"
)

var (
	ErrURLNotFound     = usecase.ErrURLNotFound
	ErrInvalidURL      = usecase.ErrInvalidURL
	ErrShortCodeExists = usecase.ErrShortCodeExists
	ErrUnauthorized    = usecase.ErrUnauthorized
)

type UserURLStats = usecase.UserURLStats

// URLService is a compatibility wrapper over the hexagonal URL use case.
type URLService struct {
	uc *usecase.URLUseCase
}

func NewURLService(urlRepo *repository.URLRepository, baseURL string) *URLService {
	generator := shortcode.NewRandomGenerator()
	uc := usecase.NewURLUseCase(urlRepo, generator, baseURL)
	return &URLService{uc: uc}
}

func (s *URLService) CreateShortURL(userID uuid.UUID, originalURL string) (*model.URL, error) {
	return s.uc.CreateShortURL(userID, originalURL)
}

func (s *URLService) GetURLByShortCode(shortCode string) (*model.URL, error) {
	return s.uc.GetURLByShortCode(shortCode)
}

func (s *URLService) GetURLByID(id uuid.UUID) (*model.URL, error) {
	return s.uc.GetURLByID(id)
}

func (s *URLService) GetUserURLs(userID uuid.UUID) ([]model.URL, error) {
	return s.uc.GetUserURLs(userID)
}

func (s *URLService) GetAllURLs(page, pageSize int) ([]model.URL, int64, error) {
	return s.uc.GetAllURLs(page, pageSize)
}

func (s *URLService) GetAllURLsWithUser() ([]model.URL, error) {
	return s.uc.GetAllURLsWithUser()
}

func (s *URLService) DeleteURL(id uuid.UUID, userID uuid.UUID, isAdmin bool) error {
	return s.uc.DeleteURL(id, userID, isAdmin)
}

func (s *URLService) DeleteURLByAdmin(id uuid.UUID) error {
	return s.uc.DeleteURLByAdmin(id)
}

func (s *URLService) IncrementClickCount(shortCode string) error {
	return s.uc.IncrementClickCount(shortCode)
}

func (s *URLService) IncrementClickCountAsync(shortCode string) {
	go func() {
		_ = s.uc.IncrementClickCount(shortCode)
	}()
}

func (s *URLService) DeactivateURL(id uuid.UUID, userID uuid.UUID, isAdmin bool) error {
	return s.uc.DeactivateURL(id, userID, isAdmin)
}

func (s *URLService) ActivateURL(id uuid.UUID, userID uuid.UUID, isAdmin bool) error {
	return s.uc.ActivateURL(id, userID, isAdmin)
}

func (s *URLService) GetUserStats(userID uuid.UUID) (*UserURLStats, error) {
	return s.uc.GetUserStats(userID)
}

func (s *URLService) GetBaseURL() string {
	return s.uc.GetBaseURL()
}

func (s *URLService) ToResponse(url *model.URL) model.URLResponse {
	return s.uc.ToResponse(url)
}

func (s *URLService) ToResponses(urls []model.URL) []model.URLResponse {
	return s.uc.ToResponses(urls)
}
