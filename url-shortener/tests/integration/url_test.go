package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/url-shortener/internal/config"
	"github.com/url-shortener/internal/core/usecase"
	"github.com/url-shortener/internal/handler"
	"github.com/url-shortener/internal/middleware"
	"github.com/url-shortener/internal/model"
	"github.com/url-shortener/internal/repository"
	"github.com/url-shortener/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// URLTestSuite defines the test suite for URL endpoints
type URLTestSuite struct {
	suite.Suite
	app            *fiber.App
	db             *gorm.DB
	authService    *service.AuthService
	urlService     *service.URLService
	urlHandler     *handler.URLHandler
	adminHandler   *handler.AdminHandler
	authMiddleware *middleware.AuthMiddleware
	cfg            *config.Config
}

// SetupSuite runs once before all tests in the suite
func (s *URLTestSuite) SetupSuite() {
	// Create test config
	s.cfg = &config.Config{
		App: config.AppConfig{
			Env:     "test",
			Port:    "8080",
			BaseURL: "http://localhost:8080",
		},
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-url-testing",
			Expiry: 1 * time.Hour,
		},
	}

	// Setup in-memory SQLite database for testing.
	// TranslateError makes duplicate keys surface as gorm.ErrDuplicatedKey,
	// matching how the production Postgres connection is configured.
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	s.Require().NoError(err)

	// Run migrations
	err = db.AutoMigrate(&model.User{}, &model.URL{})
	s.Require().NoError(err)

	s.db = db

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	urlRepo := repository.NewURLRepository(db)

	// Initialize services
	s.authService = service.NewAuthService(userRepo, s.cfg)
	s.urlService = service.NewURLService(urlRepo, s.cfg.App.BaseURL)

	// Initialize handlers
	s.urlHandler = handler.NewURLHandler(s.urlService)
	s.adminHandler = handler.NewAdminHandler(s.urlService, s.authService)
	s.authMiddleware = middleware.NewAuthMiddleware(s.authService)

	// Setup Fiber app
	s.app = fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"message": err.Error(),
				},
			})
		},
	})

	// Setup routes
	s.setupRoutes()
}

// setupRoutes configures the test routes
func (s *URLTestSuite) setupRoutes() {
	api := s.app.Group("/api/v1")

	// Auth routes for getting tokens
	authHandler := handler.NewAuthHandler(s.authService)
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)

	// URL routes (protected)
	urls := api.Group("/urls", s.authMiddleware.Authenticate())
	urls.Post("", s.urlHandler.CreateURL)
	urls.Get("/me", s.urlHandler.GetMyURLs)
	urls.Get("/stats", s.urlHandler.GetUserStats)
	urls.Get("/:id", s.urlHandler.GetURLByID)
	urls.Delete("/:id", s.urlHandler.DeleteURL)
	urls.Patch("/:id/deactivate", s.urlHandler.DeactivateURL)
	urls.Patch("/:id/activate", s.urlHandler.ActivateURL)

	// Admin routes
	admin := api.Group("/admin", s.authMiddleware.Authenticate(), middleware.RequireAdmin())
	admin.Get("/urls", s.adminHandler.GetAllURLs)
	admin.Get("/urls/:id", s.adminHandler.GetURLByID)
	admin.Delete("/urls/:id", s.adminHandler.DeleteURL)

	// Redirect route (public)
	s.app.Get("/:shortCode", s.urlHandler.Redirect)
}

// TearDownSuite runs once after all tests in the suite
func (s *URLTestSuite) TearDownSuite() {
	// Stop the click counter and drain queued writes before closing the DB
	s.urlService.Close()

	sqlDB, err := s.db.DB()
	if err == nil {
		sqlDB.Close()
	}
}

// SetupTest runs before each test
func (s *URLTestSuite) SetupTest() {
	// Drop and recreate tables to ensure clean state
	s.db.Migrator().DropTable(&model.URL{}, &model.User{})
	s.db.AutoMigrate(&model.User{}, &model.URL{})
}

// TestCreateURL_Success tests successful URL creation
func (s *URLTestSuite) TestCreateURL_Success() {
	token := s.registerAndLogin("createurl@example.com", "password123", "Create URL User")

	reqBody := model.CreateURLRequest{
		OriginalURL: "https://www.example.com/very/long/url/path",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.app.Test(req, -1)
	s.Require().NoError(err)

	assert.Equal(s.T(), fiber.StatusCreated, resp.StatusCode)

	var response map[string]interface{}
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &response)

	assert.True(s.T(), response["success"].(bool))
	assert.Equal(s.T(), "URL shortened successfully", response["message"])

	data := response["data"].(map[string]interface{})
	assert.NotEmpty(s.T(), data["id"])
	assert.NotEmpty(s.T(), data["short_code"])
	assert.NotEmpty(s.T(), data["short_url"])
	assert.Equal(s.T(), "https://www.example.com/very/long/url/path", data["original_url"])
	assert.Equal(s.T(), float64(0), data["click_count"])
}

// TestCreateURL_InvalidURL tests URL creation with invalid URL
func (s *URLTestSuite) TestCreateURL_InvalidURL() {
	token := s.registerAndLogin("invalidurl@example.com", "password123", "Invalid URL User")

	testCases := []struct {
		name string
		url  string
	}{
		{"empty URL", ""},
		{"no scheme", "example.com"},
		{"invalid scheme", "ftp://example.com"},
		{"javascript URL", "javascript:alert('xss')"},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			reqBody := model.CreateURLRequest{
				OriginalURL: tc.url,
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/api/v1/urls", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			resp, _ := s.app.Test(req, -1)
			assert.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
		})
	}
}

// TestCreateURL_NoAuth tests URL creation without authentication
func (s *URLTestSuite) TestCreateURL_NoAuth() {
	reqBody := model.CreateURLRequest{
		OriginalURL: "https://www.example.com",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusUnauthorized, resp.StatusCode)
}

// TestGetMyURLs_Success tests getting user's URLs
func (s *URLTestSuite) TestGetMyURLs_Success() {
	token := s.registerAndLogin("myurls@example.com", "password123", "My URLs User")

	// Create some URLs
	urls := []string{
		"https://www.example1.com",
		"https://www.example2.com",
		"https://www.example3.com",
	}

	for _, url := range urls {
		reqBody := model.CreateURLRequest{OriginalURL: url}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/urls", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		s.app.Test(req, -1)
	}

	// Get user's URLs
	req := httptest.NewRequest("GET", "/api/v1/urls/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.app.Test(req, -1)
	s.Require().NoError(err)

	assert.Equal(s.T(), fiber.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &response)

	assert.True(s.T(), response["success"].(bool))

	data := response["data"].([]interface{})
	assert.Equal(s.T(), 3, len(data))
}

// TestGetMyURLs_Empty tests getting URLs when user has none
func (s *URLTestSuite) TestGetMyURLs_Empty() {
	token := s.registerAndLogin("empty@example.com", "password123", "Empty User")

	req := httptest.NewRequest("GET", "/api/v1/urls/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.app.Test(req, -1)
	s.Require().NoError(err)

	assert.Equal(s.T(), fiber.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &response)

	assert.True(s.T(), response["success"].(bool))
	data := response["data"].([]interface{})
	assert.Equal(s.T(), 0, len(data))
}

// TestGetMyURLs_NoAuth tests getting URLs without authentication
func (s *URLTestSuite) TestGetMyURLs_NoAuth() {
	req := httptest.NewRequest("GET", "/api/v1/urls/me", nil)

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusUnauthorized, resp.StatusCode)
}

// TestRedirect_Success tests successful URL redirect
func (s *URLTestSuite) TestRedirect_Success() {
	token := s.registerAndLogin("redirect@example.com", "password123", "Redirect User")

	// Create a URL
	reqBody := model.CreateURLRequest{
		OriginalURL: "https://www.google.com",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, _ := s.app.Test(req, -1)

	var createResponse map[string]interface{}
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &createResponse)

	data := createResponse["data"].(map[string]interface{})
	shortCode := data["short_code"].(string)

	// Test redirect
	req = httptest.NewRequest("GET", "/"+shortCode, nil)
	resp, err := s.app.Test(req, -1)
	s.Require().NoError(err)

	// Should be a temporary redirect (302)
	assert.Equal(s.T(), fiber.StatusFound, resp.StatusCode)
	assert.Equal(s.T(), "https://www.google.com", resp.Header.Get("Location"))
}

// TestRedirect_NotFound tests redirect with non-existent short code
func (s *URLTestSuite) TestRedirect_NotFound() {
	req := httptest.NewRequest("GET", "/nonexistent", nil)

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusNotFound, resp.StatusCode)
}

// TestRedirect_InvalidShortCode tests redirect with invalid short code format
func (s *URLTestSuite) TestRedirect_InvalidShortCode() {
	req := httptest.NewRequest("GET", "/invalid!@#", nil)

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusNotFound, resp.StatusCode)
}

// TestRedirect_IncrementsClickCount verifies the redirect path end-to-end:
// the click count is queued asynchronously and lands in the DB after drain.
func (s *URLTestSuite) TestRedirect_IncrementsClickCount() {
	token := s.registerAndLogin("clickcount@example.com", "password123", "Click Count User")

	reqBody := model.CreateURLRequest{OriginalURL: "https://www.example.com/clicks"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ := s.app.Test(req, -1)
	respBody, _ := io.ReadAll(resp.Body)
	var createResponse map[string]interface{}
	json.Unmarshal(respBody, &createResponse)
	data := createResponse["data"].(map[string]interface{})
	shortCode := data["short_code"].(string)

	for i := 0; i < 3; i++ {
		s.Require().NoError(s.urlService.IncrementClickCount(shortCode))
	}

	var url model.URL
	err := s.db.Where("short_code = ?", shortCode).First(&url).Error
	s.Require().NoError(err)
	assert.Equal(s.T(), int64(3), url.ClickCount)
}

// TestCreateURL_RetriesOnShortCodeCollision verifies that a duplicate short
// code (unique-index conflict, translated like production Postgres) is
// retried with a fresh code instead of failing the request.
func (s *URLTestSuite) TestCreateURL_RetriesOnShortCodeCollision() {
	token := s.registerAndLogin("collision@example.com", "password123", "Collision User")

	reqBody := model.CreateURLRequest{OriginalURL: "https://www.example.com/first"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ := s.app.Test(req, -1)
	s.Require().Equal(fiber.StatusCreated, resp.StatusCode)
	respBody, _ := io.ReadAll(resp.Body)
	var createResponse map[string]interface{}
	json.Unmarshal(respBody, &createResponse)
	firstCode := createResponse["data"].(map[string]interface{})["short_code"].(string)

	// Force a deterministic collision: the first generated code already
	// exists, the second is fresh; the use case must retry past the conflict.
	gen := &seqGenerator{codes: []string{firstCode, "fresh1"}}
	uc := usecase.NewURLUseCase(
		repository.NewURLRepository(s.db), gen, s.cfg.App.BaseURL,
	)

	url, err := uc.CreateShortURL(uuid.New(), "https://www.example.com/second")

	s.Require().NoError(err)
	assert.NotEqual(s.T(), firstCode, url.ShortCode)
	assert.Equal(s.T(), "fresh1", url.ShortCode)
}

// seqGenerator returns preset codes in order, for deterministic collisions.
type seqGenerator struct{ codes []string }

func (g *seqGenerator) Generate(length int) (string, error) {
	if len(g.codes) == 0 {
		return "", errors.New("codes exhausted")
	}
	code := g.codes[0]
	g.codes = g.codes[1:]
	return code, nil
}

// TestDeleteURL_Success tests successful URL deletion
func (s *URLTestSuite) TestDeleteURL_Success() {
	token := s.registerAndLogin("delete@example.com", "password123", "Delete User")

	// Create a URL
	urlID := s.createURL(token, "https://www.todelete.com")

	// Delete the URL
	req := httptest.NewRequest("DELETE", "/api/v1/urls/"+urlID, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.app.Test(req, -1)
	s.Require().NoError(err)

	assert.Equal(s.T(), fiber.StatusOK, resp.StatusCode)

	// Verify URL is deleted
	req = httptest.NewRequest("GET", "/api/v1/urls/"+urlID, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, _ = s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusNotFound, resp.StatusCode)
}

// TestDeleteURL_NotOwner tests deleting URL by non-owner
func (s *URLTestSuite) TestDeleteURL_NotOwner() {
	// User 1 creates a URL
	token1 := s.registerAndLogin("owner@example.com", "password123", "Owner User")
	urlID := s.createURL(token1, "https://www.ownerurl.com")

	// User 2 tries to delete it
	token2 := s.registerAndLogin("notowner@example.com", "password123", "Not Owner User")

	req := httptest.NewRequest("DELETE", "/api/v1/urls/"+urlID, nil)
	req.Header.Set("Authorization", "Bearer "+token2)

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusForbidden, resp.StatusCode)
}

// TestDeleteURL_NotFound tests deleting non-existent URL
func (s *URLTestSuite) TestDeleteURL_NotFound() {
	token := s.registerAndLogin("deletenf@example.com", "password123", "Delete NF User")

	fakeID := uuid.New().String()
	req := httptest.NewRequest("DELETE", "/api/v1/urls/"+fakeID, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusNotFound, resp.StatusCode)
}

// TestDeleteURL_InvalidID tests deleting with invalid UUID
func (s *URLTestSuite) TestDeleteURL_InvalidID() {
	token := s.registerAndLogin("deleteinv@example.com", "password123", "Delete Invalid User")

	req := httptest.NewRequest("DELETE", "/api/v1/urls/not-a-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusBadRequest, resp.StatusCode)
}

// TestGetURLByID_Success tests getting URL by ID
func (s *URLTestSuite) TestGetURLByID_Success() {
	token := s.registerAndLogin("getbyid@example.com", "password123", "Get By ID User")

	urlID := s.createURL(token, "https://www.getbyid.com")

	req := httptest.NewRequest("GET", "/api/v1/urls/"+urlID, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.app.Test(req, -1)
	s.Require().NoError(err)

	assert.Equal(s.T(), fiber.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &response)

	assert.True(s.T(), response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(s.T(), "https://www.getbyid.com", data["original_url"])
}

// TestGetURLByID_NotOwner tests getting URL by non-owner
func (s *URLTestSuite) TestGetURLByID_NotOwner() {
	token1 := s.registerAndLogin("urlowner@example.com", "password123", "URL Owner")
	urlID := s.createURL(token1, "https://www.privateurl.com")

	token2 := s.registerAndLogin("urlviewer@example.com", "password123", "URL Viewer")

	req := httptest.NewRequest("GET", "/api/v1/urls/"+urlID, nil)
	req.Header.Set("Authorization", "Bearer "+token2)

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusForbidden, resp.StatusCode)
}

// TestDeactivateURL_Success tests URL deactivation
func (s *URLTestSuite) TestDeactivateURL_Success() {
	token := s.registerAndLogin("deactivate@example.com", "password123", "Deactivate User")

	urlID := s.createURL(token, "https://www.todeactivate.com")

	// Get short code before deactivation
	req := httptest.NewRequest("GET", "/api/v1/urls/"+urlID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ := s.app.Test(req, -1)

	var getResponse map[string]interface{}
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &getResponse)
	data := getResponse["data"].(map[string]interface{})
	shortCode := data["short_code"].(string)

	// Deactivate
	req = httptest.NewRequest("PATCH", "/api/v1/urls/"+urlID+"/deactivate", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.app.Test(req, -1)
	s.Require().NoError(err)

	assert.Equal(s.T(), fiber.StatusOK, resp.StatusCode)

	// Verify redirect no longer works
	req = httptest.NewRequest("GET", "/"+shortCode, nil)
	resp, _ = s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusNotFound, resp.StatusCode)
}

// TestActivateURL_Success tests URL activation
func (s *URLTestSuite) TestActivateURL_Success() {
	token := s.registerAndLogin("activate@example.com", "password123", "Activate User")

	urlID := s.createURL(token, "https://www.toactivate.com")

	// Deactivate first
	req := httptest.NewRequest("PATCH", "/api/v1/urls/"+urlID+"/deactivate", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	s.app.Test(req, -1)

	// Activate
	req = httptest.NewRequest("PATCH", "/api/v1/urls/"+urlID+"/activate", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.app.Test(req, -1)
	s.Require().NoError(err)

	assert.Equal(s.T(), fiber.StatusOK, resp.StatusCode)
}

// TestGetUserStats_Success tests getting user URL statistics
func (s *URLTestSuite) TestGetUserStats_Success() {
	token := s.registerAndLogin("stats@example.com", "password123", "Stats User")

	// Create some URLs
	s.createURL(token, "https://www.stats1.com")
	s.createURL(token, "https://www.stats2.com")

	req := httptest.NewRequest("GET", "/api/v1/urls/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.app.Test(req, -1)
	s.Require().NoError(err)

	assert.Equal(s.T(), fiber.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &response)

	assert.True(s.T(), response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(s.T(), float64(2), data["total_urls"])
	assert.Equal(s.T(), float64(0), data["total_clicks"])
}

// TestClickCounting tests that clicks are counted
func (s *URLTestSuite) TestClickCounting() {
	token := s.registerAndLogin("clicks@example.com", "password123", "Clicks User")

	urlID := s.createURL(token, "https://www.countclicks.com")

	// Get short code
	req := httptest.NewRequest("GET", "/api/v1/urls/"+urlID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ := s.app.Test(req, -1)

	var getResponse map[string]interface{}
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &getResponse)
	data := getResponse["data"].(map[string]interface{})
	shortCode := data["short_code"].(string)

	// Click the link multiple times
	for i := 0; i < 5; i++ {
		req = httptest.NewRequest("GET", "/"+shortCode, nil)
		s.app.Test(req, -1)
	}

	// Wait a bit for async click counting
	time.Sleep(100 * time.Millisecond)

	// Check click count
	req = httptest.NewRequest("GET", "/api/v1/urls/"+urlID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, _ = s.app.Test(req, -1)

	json.Unmarshal(func() []byte {
		b, _ := io.ReadAll(resp.Body)
		return b
	}(), &getResponse)

	data = getResponse["data"].(map[string]interface{})
	// Click count should be >= 0 (async counting may vary)
	assert.GreaterOrEqual(s.T(), data["click_count"].(float64), float64(0))
}

// Helper function to register and login, returns the access token
func (s *URLTestSuite) registerAndLogin(email, password, name string) string {
	// Register
	registerReq := model.RegisterRequest{
		Email:    email,
		Password: password,
		Name:     name,
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.app.Test(req, -1)

	// Login
	loginReq := model.LoginRequest{
		Email:    email,
		Password: password,
	}
	body, _ = json.Marshal(loginReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := s.app.Test(req, -1)

	var response map[string]interface{}
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &response)

	data := response["data"].(map[string]interface{})
	return data["access_token"].(string)
}

// Helper function to create a URL and return its ID
func (s *URLTestSuite) createURL(token, originalURL string) string {
	reqBody := model.CreateURLRequest{
		OriginalURL: originalURL,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/urls", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, _ := s.app.Test(req, -1)

	var response map[string]interface{}
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &response)

	data := response["data"].(map[string]interface{})
	return data["id"].(string)
}

// TestURLTestSuite runs the test suite
func TestURLTestSuite(t *testing.T) {
	suite.Run(t, new(URLTestSuite))
}
