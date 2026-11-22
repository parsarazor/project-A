package service

import (
	"log"
	"sync"

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

// clickQueueSize bounds the pending click-count writes. When the queue is
// full, new counts are dropped instead of slowing down redirects.
const clickQueueSize = 1024

// URLService is a compatibility wrapper over the hexagonal URL use case.
type URLService struct {
	uc        *usecase.URLUseCase
	clicks    chan string
	quit      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
}

func NewURLService(urlRepo *repository.URLRepository, baseURL string) *URLService {
	generator := shortcode.NewRandomGenerator()
	uc := usecase.NewURLUseCase(urlRepo, generator, baseURL)
	s := &URLService{
		uc:     uc,
		clicks: make(chan string, clickQueueSize),
		quit:   make(chan struct{}),
		done:   make(chan struct{}),
	}
	go s.processClicks()
	return s
}

// processClicks is the single writer for click counts: one worker goroutine
// drains the queue, instead of an unbounded goroutine per redirect racing the
// app shutdown. After Close it drains what is already queued.
func (s *URLService) processClicks() {
	defer close(s.done)
	for {
		select {
		case shortCode := <-s.clicks:
			s.recordClick(shortCode)
		case <-s.quit:
			for {
				select {
				case shortCode := <-s.clicks:
					s.recordClick(shortCode)
				default:
					return
				}
			}
		}
	}
}

func (s *URLService) recordClick(shortCode string) {
	if err := s.uc.IncrementClickCount(shortCode); err != nil {
		log.Printf("click count update failed for %q: %v", shortCode, err)
	}
}

// Close stops the click counter and waits for queued writes to drain. It is
// safe to call more than once.
func (s *URLService) Close() {
	s.closeOnce.Do(func() {
		close(s.quit)
		<-s.done
	})
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
	select {
	case s.clicks <- shortCode:
	default:
		// Queue full: drop the count rather than block the redirect.
		log.Printf("click count queue full, dropping count for %q", shortCode)
	}
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
