package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/url-shortener/internal/config"
	"github.com/url-shortener/internal/handler"
	"github.com/url-shortener/internal/middleware"
	"github.com/url-shortener/internal/model"
	"github.com/url-shortener/internal/repository"
	"github.com/url-shortener/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// AuthTestSuite defines the test suite for authentication endpoints
type AuthTestSuite struct {
	suite.Suite
	app            *fiber.App
	db             *gorm.DB
	authService    *service.AuthService
	authHandler    *handler.AuthHandler
	authMiddleware *middleware.AuthMiddleware
}

// SetupSuite runs once before all tests in the suite
func (s *AuthTestSuite) SetupSuite() {
	// Create test config
	cfg := &config.Config{
		App: config.AppConfig{
			Env:     "test",
			Port:    "8080",
			BaseURL: "http://localhost:8080",
		},
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-integration-testing",
			Expiry: 1 * time.Hour,
		},
	}

	// Setup in-memory SQLite database for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	s.Require().NoError(err)

	// Run migrations
	err = db.AutoMigrate(&model.User{}, &model.URL{})
	s.Require().NoError(err)

	s.db = db

	// Initialize repositories and services
	userRepo := repository.NewUserRepository(db)
	s.authService = service.NewAuthService(userRepo, cfg)
	s.authHandler = handler.NewAuthHandler(s.authService)
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
func (s *AuthTestSuite) setupRoutes() {
	api := s.app.Group("/api/v1")

	// Auth routes
	auth := api.Group("/auth")
	auth.Post("/register", s.authHandler.Register)
	auth.Post("/login", s.authHandler.Login)

	// Protected routes
	authProtected := auth.Group("", s.authMiddleware.Authenticate())
	authProtected.Get("/me", s.authHandler.GetMe)
	authProtected.Post("/change-password", s.authHandler.ChangePassword)
}

// TearDownSuite runs once after all tests in the suite
func (s *AuthTestSuite) TearDownSuite() {
	sqlDB, err := s.db.DB()
	if err == nil {
		sqlDB.Close()
	}
}

// SetupTest runs before each test
func (s *AuthTestSuite) SetupTest() {
	// Clean up users table before each test
	s.db.Exec("DELETE FROM users")
}

// TestRegister_Success tests successful user registration
func (s *AuthTestSuite) TestRegister_Success() {
	reqBody := model.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.app.Test(req, -1)
	s.Require().NoError(err)

	assert.Equal(s.T(), fiber.StatusCreated, resp.StatusCode)

	// Parse response
	var response map[string]interface{}
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &response)

	assert.True(s.T(), response["success"].(bool))
	assert.Equal(s.T(), "User registered successfully", response["message"])

	// Verify user data in response
	data := response["data"].(map[string]interface{})
	assert.Equal(s.T(), "test@example.com", data["email"])
	assert.Equal(s.T(), "Test User", data["name"])
	assert.Equal(s.T(), "user", data["role"])
	assert.NotEmpty(s.T(), data["id"])
}

// TestRegister_DuplicateEmail tests registration with existing email
func (s *AuthTestSuite) TestRegister_DuplicateEmail() {
	// Register first user
	reqBody := model.RegisterRequest{
		Email:    "duplicate@example.com",
		Password: "password123",
		Name:     "First User",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusCreated, resp.StatusCode)

	// Try to register with same email
	reqBody.Name = "Second User"
	body, _ = json.Marshal(reqBody)
	req = httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ = s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusConflict, resp.StatusCode)

	var response map[string]interface{}
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &response)

	assert.False(s.T(), response["success"].(bool))
}

// TestRegister_InvalidEmail tests registration with invalid email
func (s *AuthTestSuite) TestRegister_InvalidEmail() {
	reqBody := model.RegisterRequest{
		Email:    "invalid-email",
		Password: "password123",
		Name:     "Test User",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusUnprocessableEntity, resp.StatusCode)
}

// TestRegister_ShortPassword tests registration with short password
func (s *AuthTestSuite) TestRegister_ShortPassword() {
	reqBody := model.RegisterRequest{
		Email:    "test@example.com",
		Password: "short",
		Name:     "Test User",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusUnprocessableEntity, resp.StatusCode)
}

// TestRegister_EmptyName tests registration with empty name
func (s *AuthTestSuite) TestRegister_EmptyName() {
	reqBody := model.RegisterRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusUnprocessableEntity, resp.StatusCode)
}

// TestRegister_InvalidJSON tests registration with invalid JSON body
func (s *AuthTestSuite) TestRegister_InvalidJSON() {
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusBadRequest, resp.StatusCode)
}

// TestLogin_Success tests successful login
func (s *AuthTestSuite) TestLogin_Success() {
	// First register a user
	registerReq := model.RegisterRequest{
		Email:    "login@example.com",
		Password: "password123",
		Name:     "Login User",
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.app.Test(req, -1)

	// Now login
	loginReq := model.LoginRequest{
		Email:    "login@example.com",
		Password: "password123",
	}
	body, _ = json.Marshal(loginReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.app.Test(req, -1)
	s.Require().NoError(err)

	assert.Equal(s.T(), fiber.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &response)

	assert.True(s.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.NotEmpty(s.T(), data["access_token"])
	assert.Equal(s.T(), "Bearer", data["token_type"])
	assert.Greater(s.T(), data["expires_in"].(float64), float64(0))
}

// TestLogin_WrongPassword tests login with wrong password
func (s *AuthTestSuite) TestLogin_WrongPassword() {
	// First register a user
	registerReq := model.RegisterRequest{
		Email:    "wrongpass@example.com",
		Password: "password123",
		Name:     "Wrong Pass User",
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.app.Test(req, -1)

	// Try to login with wrong password
	loginReq := model.LoginRequest{
		Email:    "wrongpass@example.com",
		Password: "wrongpassword",
	}
	body, _ = json.Marshal(loginReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusUnauthorized, resp.StatusCode)
}

// TestLogin_NonExistentUser tests login with non-existent email
func (s *AuthTestSuite) TestLogin_NonExistentUser() {
	loginReq := model.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusUnauthorized, resp.StatusCode)
}

// TestLogin_InvalidEmail tests login with invalid email format
func (s *AuthTestSuite) TestLogin_InvalidEmail() {
	loginReq := model.LoginRequest{
		Email:    "invalid-email",
		Password: "password123",
	}
	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusUnprocessableEntity, resp.StatusCode)
}

// TestLogin_EmptyPassword tests login with empty password
func (s *AuthTestSuite) TestLogin_EmptyPassword() {
	loginReq := model.LoginRequest{
		Email:    "test@example.com",
		Password: "",
	}
	body, _ := json.Marshal(loginReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusUnprocessableEntity, resp.StatusCode)
}

// TestGetMe_Success tests getting current user profile
func (s *AuthTestSuite) TestGetMe_Success() {
	// Register and login to get token
	token := s.registerAndLogin("me@example.com", "password123", "Me User")

	// Get profile
	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.app.Test(req, -1)
	s.Require().NoError(err)

	assert.Equal(s.T(), fiber.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &response)

	assert.True(s.T(), response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(s.T(), "me@example.com", data["email"])
	assert.Equal(s.T(), "Me User", data["name"])
}

// TestGetMe_NoToken tests accessing protected route without token
func (s *AuthTestSuite) TestGetMe_NoToken() {
	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusUnauthorized, resp.StatusCode)
}

// TestGetMe_InvalidToken tests accessing protected route with invalid token
func (s *AuthTestSuite) TestGetMe_InvalidToken() {
	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusUnauthorized, resp.StatusCode)
}

// TestGetMe_MalformedAuthHeader tests with malformed authorization header
func (s *AuthTestSuite) TestGetMe_MalformedAuthHeader() {
	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "NotBearer token")

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusUnauthorized, resp.StatusCode)
}

// TestChangePassword_Success tests successful password change
func (s *AuthTestSuite) TestChangePassword_Success() {
	// Register and login to get token
	token := s.registerAndLogin("changepass@example.com", "oldpassword123", "Change Pass User")

	// Change password
	changeReq := handler.ChangePasswordRequest{
		CurrentPassword: "oldpassword123",
		NewPassword:     "newpassword123",
	}
	body, _ := json.Marshal(changeReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.app.Test(req, -1)
	s.Require().NoError(err)

	assert.Equal(s.T(), fiber.StatusOK, resp.StatusCode)

	// Verify old password no longer works
	loginReq := model.LoginRequest{
		Email:    "changepass@example.com",
		Password: "oldpassword123",
	}
	body, _ = json.Marshal(loginReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ = s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusUnauthorized, resp.StatusCode)

	// Verify new password works
	loginReq.Password = "newpassword123"
	body, _ = json.Marshal(loginReq)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ = s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusOK, resp.StatusCode)
}

// TestChangePassword_WrongCurrentPassword tests password change with wrong current password
func (s *AuthTestSuite) TestChangePassword_WrongCurrentPassword() {
	// Register and login to get token
	token := s.registerAndLogin("wrongcurrent@example.com", "correctpassword", "Wrong Current User")

	// Try to change password with wrong current password
	changeReq := handler.ChangePasswordRequest{
		CurrentPassword: "wrongcurrentpassword",
		NewPassword:     "newpassword123",
	}
	body, _ := json.Marshal(changeReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusUnauthorized, resp.StatusCode)
}

// TestChangePassword_WeakNewPassword tests password change with weak new password
func (s *AuthTestSuite) TestChangePassword_WeakNewPassword() {
	// Register and login to get token
	token := s.registerAndLogin("weaknew@example.com", "password123", "Weak New User")

	// Try to change to weak password
	changeReq := handler.ChangePasswordRequest{
		CurrentPassword: "password123",
		NewPassword:     "weak",
	}
	body, _ := json.Marshal(changeReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusUnprocessableEntity, resp.StatusCode)
}

// TestChangePassword_NoToken tests password change without token
func (s *AuthTestSuite) TestChangePassword_NoToken() {
	changeReq := handler.ChangePasswordRequest{
		CurrentPassword: "password123",
		NewPassword:     "newpassword123",
	}
	body, _ := json.Marshal(changeReq)
	req := httptest.NewRequest("POST", "/api/v1/auth/change-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := s.app.Test(req, -1)
	assert.Equal(s.T(), fiber.StatusUnauthorized, resp.StatusCode)
}

// Helper function to register and login, returns the access token
func (s *AuthTestSuite) registerAndLogin(email, password, name string) string {
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

// TestAuthTestSuite runs the test suite
func TestAuthTestSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}

// TestRegistrationConcurrentSameEmail fires two concurrent registrations for
// the same email and asserts exactly one 201, one 409 (never a 500) and a
// single DB row.
func TestRegistrationConcurrentSameEmail(t *testing.T) {
	cfg := &config.Config{App: config.AppConfig{Env: "test"}}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	// SQLite :memory: is per-connection; pin the pool to one connection so
	// both requests share the same in-memory database and the second insert
	// deterministically hits the unique index.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	authService := service.NewAuthService(
		repository.NewUserRepository(db),
		cfg,
	)
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		},
	})
	app.Post("/api/v1/auth/register", handler.NewAuthHandler(authService).Register)

	email := fmt.Sprintf("race-%d@example.com", time.Now().UnixNano())
	const attempts = 2

	type result struct {
		status int
	}
	results := make([]result, attempts)
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			body, _ := json.Marshal(model.RegisterRequest{
				Email:    email,
				Password: "sup3r-Secret!",
				Name:     "Race User",
			})
			req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Errorf("request %d failed: %v", idx, err)
				return
			}
			results[idx] = result{status: resp.StatusCode}
		}(i)
	}

	close(start)
	wg.Wait()

	statuses := []int{results[0].status, results[1].status}
	sort.Ints(statuses)
	assert.Equal(t, []int{fiber.StatusCreated, fiber.StatusConflict}, statuses,
		"expected exactly one 201 and one 409, got %v", statuses)

	var count int64
	db.Model(&model.User{}).Where("email = ?", email).Count(&count)
	assert.EqualValues(t, 1, count, "expected exactly one user row for %s", email)
}

// TestRegisterCaseInsensitiveDuplicateEmail asserts that registering with the
// same email in a different case is rejected as a duplicate.
func TestRegisterCaseInsensitiveDuplicateEmail(t *testing.T) {
	cfg := &config.Config{App: config.AppConfig{Env: "test"}}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	authService := service.NewAuthService(repository.NewUserRepository(db), cfg)
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		},
	})
	app.Post("/api/v1/auth/register", handler.NewAuthHandler(authService).Register)

	register := func(email string) int {
		body, _ := json.Marshal(model.RegisterRequest{
			Email:    email,
			Password: "sup3r-Secret!",
			Name:     "Case User",
		})
		req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		return resp.StatusCode
	}

	assert.Equal(t, fiber.StatusCreated, register("mixed@example.com"))
	assert.Equal(t, fiber.StatusConflict, register("MIXED@example.com"),
		"same email in another case must be rejected as duplicate, not create a second account")
}

// TestReadinessEndpoint asserts /health stays a static liveness endpoint while
// /ready reports 200 with the DB up and 503 without it.
func TestReadinessEndpoint(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	app := fiber.New()
	handler.RegisterHealthHandlers(app, db, "")

	resp, err := app.Test(httptest.NewRequest("GET", "/health", nil), -1)
	if err != nil {
		t.Fatalf("/health request failed: %v", err)
	}
	assert.Equal(t, fiber.StatusOK, resp.StatusCode, "/health must stay live while DB is up")

	resp, err = app.Test(httptest.NewRequest("GET", "/ready", nil), -1)
	if err != nil {
		t.Fatalf("/ready request failed: %v", err)
	}
	assert.Equal(t, fiber.StatusOK, resp.StatusCode, "/ready must return 200 with a reachable DB")

	// Close the DB to simulate an unreachable database.
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.Close()
	}

	resp, err = app.Test(httptest.NewRequest("GET", "/health", nil), -1)
	if err != nil {
		t.Fatalf("/health request failed: %v", err)
	}
	assert.Equal(t, fiber.StatusOK, resp.StatusCode, "/health must stay a static liveness endpoint when the DB is down")

	resp, err = app.Test(httptest.NewRequest("GET", "/ready", nil), -1)
	if err != nil {
		t.Fatalf("/ready request failed: %v", err)
	}
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode, "/ready must return 503 when the DB is unreachable")
}
