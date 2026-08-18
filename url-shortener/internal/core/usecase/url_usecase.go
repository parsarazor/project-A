package usecase

import (
	"errors"

	"github.com/google/uuid"
	"github.com/url-shortener/internal/core/ports"
	"github.com/url-shortener/internal/model"
	"github.com/url-shortener/internal/utils"
)

var (
	ErrURLNotFound     = errors.New("URL not found")
	ErrInvalidURL      = errors.New("invalid URL")
	ErrShortCodeExists = errors.New("short code already exists")
	ErrUnauthorized    = errors.New("unauthorized to access this URL")
)

type UserURLStats struct {
	TotalURLs   int64 `json:"total_urls"`
	TotalClicks int64 `json:"total_clicks"`
}

type URLUseCase struct {
	urlRepo      ports.URLRepository
	shortCodeGen ports.ShortCodeGenerator
	baseURL      string
}

func NewURLUseCase(urlRepo ports.URLRepository, shortCodeGen ports.ShortCodeGenerator, baseURL string) *URLUseCase {
	return &URLUseCase{urlRepo: urlRepo, shortCodeGen: shortCodeGen, baseURL: baseURL}
}

func (uc *URLUseCase) CreateShortURL(userID uuid.UUID, originalURL string) (*model.URL, error) {
	if !utils.IsValidURL(originalURL) {
		return nil, ErrInvalidURL
	}

	var shortCode string
	var err error
	for i := 0; i < 10; i++ {
		shortCode, err = uc.shortCodeGen.Generate(utils.DefaultCodeLength)
		if err != nil {
			return nil, err
		}

		exists, err := uc.urlRepo.ShortCodeExists(shortCode)
		if err != nil {
			return nil, err
		}

		if !exists {
			break
		}

		if i == 9 {
			return nil, ErrShortCodeExists
		}
	}

	url := &model.URL{ShortCode: shortCode, OriginalURL: originalURL, UserID: userID, ClickCount: 0, IsActive: true}
	if err := uc.urlRepo.Create(url); err != nil {
		return nil, err
	}
	return url, nil
}

func (uc *URLUseCase) GetURLByShortCode(shortCode string) (*model.URL, error) {
	url, err := uc.urlRepo.FindByShortCode(shortCode)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrURLNotFound
		}
		return nil, err
	}
	if !url.IsActive {
		return nil, ErrURLNotFound
	}
	return url, nil
}

func (uc *URLUseCase) GetURLByID(id uuid.UUID) (*model.URL, error) {
	url, err := uc.urlRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrURLNotFound
		}
		return nil, err
	}
	return url, nil
}

func (uc *URLUseCase) GetUserURLs(userID uuid.UUID) ([]model.URL, error) {
	return uc.urlRepo.FindByUserID(userID)
}

func (uc *URLUseCase) GetAllURLs(page, pageSize int) ([]model.URL, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return uc.urlRepo.FindAll(page, pageSize)
}

func (uc *URLUseCase) GetAllURLsWithUser() ([]model.URL, error) {
	return uc.urlRepo.FindAllWithUser()
}

func (uc *URLUseCase) DeleteURL(id, userID uuid.UUID, isAdmin bool) error {
	url, err := uc.urlRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return ErrURLNotFound
		}
		return err
	}
	if !isAdmin && url.UserID != userID {
		return ErrUnauthorized
	}
	return uc.urlRepo.Delete(id)
}

func (uc *URLUseCase) DeleteURLByAdmin(id uuid.UUID) error {
	_, err := uc.urlRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return ErrURLNotFound
		}
		return err
	}
	return uc.urlRepo.Delete(id)
}

func (uc *URLUseCase) IncrementClickCount(shortCode string) error {
	return uc.urlRepo.IncrementClickCountByShortCode(shortCode)
}

func (uc *URLUseCase) DeactivateURL(id, userID uuid.UUID, isAdmin bool) error {
	url, err := uc.urlRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return ErrURLNotFound
		}
		return err
	}
	if !isAdmin && url.UserID != userID {
		return ErrUnauthorized
	}
	return uc.urlRepo.Deactivate(id)
}

func (uc *URLUseCase) ActivateURL(id, userID uuid.UUID, isAdmin bool) error {
	url, err := uc.urlRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return ErrURLNotFound
		}
		return err
	}
	if !isAdmin && url.UserID != userID {
		return ErrUnauthorized
	}
	return uc.urlRepo.Activate(id)
}

func (uc *URLUseCase) GetUserStats(userID uuid.UUID) (*UserURLStats, error) {
	urlCount, err := uc.urlRepo.CountByUserID(userID)
	if err != nil {
		return nil, err
	}
	totalClicks, err := uc.urlRepo.GetTotalClicks(userID)
	if err != nil {
		return nil, err
	}
	return &UserURLStats{TotalURLs: urlCount, TotalClicks: totalClicks}, nil
}

func (uc *URLUseCase) GetBaseURL() string {
	return uc.baseURL
}

func (uc *URLUseCase) ToResponse(url *model.URL) model.URLResponse {
	return url.ToResponse(uc.baseURL)
}

func (uc *URLUseCase) ToResponses(urls []model.URL) []model.URLResponse {
	out := make([]model.URLResponse, len(urls))
	for i := range urls {
		out[i] = urls[i].ToResponse(uc.baseURL)
	}
	return out
}
