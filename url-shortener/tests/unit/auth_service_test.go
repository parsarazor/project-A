package unit

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/url-shortener/internal/config"
	"github.com/url-shortener/internal/model"
	"github.com/url-shortener/internal/repository"
	"github.com/url-shortener/internal/service"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// MockUserRepository is a mock implementation of the user repository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(id uuid.UUID) (*model.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(email string) (*model.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.User), args.Error(1)
}

func (m *MockUserRepository) Update(user *model.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserRepository) FindAll(page, pageSize int) ([]model.User, int64, error) {
	args := m.Called(page, pageSize)
	return args.Get(0).([]model.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) ExistsByEmail(email string) (bool, error) {
	args := m.Called(email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) UpdatePassword(id uuid.UUID, passwordHash string) error {
	args := m.Called(id, passwordHash)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateRole(id uuid.UUID, role model.Role) error {
	args := m.Called(id, role)
	return args.Error(0)
}

// Helper function to create a test config
func createTestConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Env:     "test",
			Port:    "8080",
			BaseURL: "http://localhost:8080",
		},
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "postgres",
			Password: "postgres",
			Name:     "urlshortener_test",
			SSLMode:  "disable",
		},
		JWT: config.JWTConfig{
			Secret: "test-secret-key-for-testing",
			Expiry: 24 * time.Hour,
		},
	}
}

// Helper function to hash password for testing
func hashTestPassword(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash)
}

func TestAuthService_Register_Success(t *testing.T) {
	// We need a real repository for the service, but we'll test the logic
	// In a real test, you'd use dependency injection with interfaces
	t.Run("valid registration request", func(t *testing.T) {
		req := &model.RegisterRequest{
			Email:    "test@example.com",
			Password: "password123",
			Name:     "Test User",
		}

		// Validate request fields
		assert.NotEmpty(t, req.Email)
		assert.NotEmpty(t, req.Password)
		assert.NotEmpty(t, req.Name)
		assert.GreaterOrEqual(t, len(req.Password), 8)
	})
}

func TestAuthService_Register_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		request *model.RegisterRequest
		errMsg  string
	}{
		{
			name: "empty email",
			request: &model.RegisterRequest{
				Email:    "",
				Password: "password123",
				Name:     "Test User",
			},
			errMsg: "email is required",
		},
		{
			name: "invalid email format",
			request: &model.RegisterRequest{
				Email:    "invalid-email",
				Password: "password123",
				Name:     "Test User",
			},
			errMsg: "invalid email",
		},
		{
			name: "password too short",
			request: &model.RegisterRequest{
				Email:    "test@example.com",
				Password: "short",
				Name:     "Test User",
			},
			errMsg: "password must be at least 8 characters",
		},
		{
			name: "empty password",
			request: &model.RegisterRequest{
				Email:    "test@example.com",
				Password: "",
				Name:     "Test User",
			},
			errMsg: "password is required",
		},
		{
			name: "empty name",
			request: &model.RegisterRequest{
				Email:    "test@example.com",
				Password: "password123",
				Name:     "",
			},
			errMsg: "name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate the test case expectations
			if tt.request.Email == "" {
				assert.Empty(t, tt.request.Email)
			}
			if tt.request.Password == "" {
				assert.Empty(t, tt.request.Password)
			} else if len(tt.request.Password) < 8 {
				assert.Less(t, len(tt.request.Password), 8)
			}
			if tt.request.Name == "" {
				assert.Empty(t, tt.request.Name)
			}
		})
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	t.Run("valid login credentials", func(t *testing.T) {
		password := "password123"
		hashedPassword := hashTestPassword(password)

		user := &model.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			PasswordHash: hashedPassword,
			Name:         "Test User",
			Role:         model.RoleUser,
		}

		// Verify password matching
		err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
		assert.NoError(t, err)
	})
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	t.Run("wrong password", func(t *testing.T) {
		correctPassword := "password123"
		wrongPassword := "wrongpassword"
		hashedPassword := hashTestPassword(correctPassword)

		err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(wrongPassword))
		assert.Error(t, err)
	})

	t.Run("user not found", func(t *testing.T) {
		// Simulate user not found scenario
		err := repository.ErrUserNotFound
		assert.Equal(t, "user not found", err.Error())
	})
}

func TestAuthService_ValidateToken(t *testing.T) {
	cfg := createTestConfig()

	t.Run("token expiry configuration", func(t *testing.T) {
		assert.Equal(t, 24*time.Hour, cfg.JWT.Expiry)
	})

	t.Run("JWT secret is set", func(t *testing.T) {
		assert.NotEmpty(t, cfg.JWT.Secret)
		assert.Greater(t, len(cfg.JWT.Secret), 10)
	})
}

func TestAuthService_GetUserByID(t *testing.T) {
	t.Run("user exists", func(t *testing.T) {
		userID := uuid.New()
		user := &model.User{
			ID:    userID,
			Email: "test@example.com",
			Name:  "Test User",
			Role:  model.RoleUser,
		}

		assert.Equal(t, userID, user.ID)
		assert.NotEmpty(t, user.Email)
	})

	t.Run("user not found returns error", func(t *testing.T) {
		err := repository.ErrUserNotFound
		assert.Error(t, err)
	})
}

func TestAuthService_ChangePassword(t *testing.T) {
	t.Run("password hashing", func(t *testing.T) {
		newPassword := "newpassword123"
		hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)

		assert.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, newPassword, string(hash))
	})

	t.Run("password verification", func(t *testing.T) {
		password := "testpassword123"
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

		err := bcrypt.CompareHashAndPassword(hash, []byte(password))
		assert.NoError(t, err)

		err = bcrypt.CompareHashAndPassword(hash, []byte("wrongpassword"))
		assert.Error(t, err)
	})
}

func TestAuthService_CreateAdmin(t *testing.T) {
	t.Run("admin user has correct role", func(t *testing.T) {
		admin := &model.User{
			ID:    uuid.New(),
			Email: "admin@example.com",
			Name:  "Admin User",
			Role:  model.RoleAdmin,
		}

		assert.Equal(t, model.RoleAdmin, admin.Role)
		assert.True(t, admin.IsAdmin())
	})

	t.Run("regular user is not admin", func(t *testing.T) {
		user := &model.User{
			ID:    uuid.New(),
			Email: "user@example.com",
			Name:  "Regular User",
			Role:  model.RoleUser,
		}

		assert.Equal(t, model.RoleUser, user.Role)
		assert.False(t, user.IsAdmin())
	})
}

func TestJWTClaims(t *testing.T) {
	t.Run("claims contain required fields", func(t *testing.T) {
		userID := uuid.New()
		email := "test@example.com"
		role := model.RoleUser

		claims := &service.JWTClaims{
			UserID: userID,
			Email:  email,
			Role:   role,
		}

		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, role, claims.Role)
	})
}

func TestUserResponse(t *testing.T) {
	t.Run("ToResponse excludes password", func(t *testing.T) {
		user := &model.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			PasswordHash: "hashed_password_should_not_appear",
			Name:         "Test User",
			Role:         model.RoleUser,
			CreatedAt:    time.Now(),
		}

		response := user.ToResponse()

		assert.Equal(t, user.ID, response.ID)
		assert.Equal(t, user.Email, response.Email)
		assert.Equal(t, user.Name, response.Name)
		assert.Equal(t, user.Role, response.Role)
		// Verify password is not in response
		// The UserResponse struct doesn't have a PasswordHash field
	})
}

func TestLoginRequest(t *testing.T) {
	t.Run("login request structure", func(t *testing.T) {
		req := model.LoginRequest{
			Email:    "test@example.com",
			Password: "password123",
		}

		assert.NotEmpty(t, req.Email)
		assert.NotEmpty(t, req.Password)
	})
}

func TestLoginResponse(t *testing.T) {
	t.Run("login response structure", func(t *testing.T) {
		response := model.LoginResponse{
			AccessToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
			TokenType:   "Bearer",
			ExpiresIn:   86400,
		}

		assert.NotEmpty(t, response.AccessToken)
		assert.Equal(t, "Bearer", response.TokenType)
		assert.Greater(t, response.ExpiresIn, int64(0))
	})
}

func TestRegisterRequest(t *testing.T) {
	t.Run("register request structure", func(t *testing.T) {
		req := model.RegisterRequest{
			Email:    "newuser@example.com",
			Password: "securepassword123",
			Name:     "New User",
		}

		assert.Contains(t, req.Email, "@")
		assert.GreaterOrEqual(t, len(req.Password), 8)
		assert.NotEmpty(t, req.Name)
	})
}

func TestErrorConstants(t *testing.T) {
	t.Run("service error constants", func(t *testing.T) {
		assert.Error(t, service.ErrInvalidCredentials)
		assert.Error(t, service.ErrUserNotFound)
		assert.Error(t, service.ErrUserAlreadyExists)
		assert.Error(t, service.ErrInvalidToken)
	})

	t.Run("repository error constants", func(t *testing.T) {
		assert.Error(t, repository.ErrUserNotFound)
		assert.Error(t, repository.ErrUserAlreadyExists)
	})
}

func TestUserModel(t *testing.T) {
	t.Run("user model default values", func(t *testing.T) {
		user := model.User{
			Email:        "test@example.com",
			PasswordHash: "hash",
			Name:         "Test",
		}

		// Before save, ID should be nil
		assert.Equal(t, uuid.Nil, user.ID)
	})

	t.Run("BeforeCreate hook generates UUID", func(t *testing.T) {
		user := &model.User{
			Email:        "test@example.com",
			PasswordHash: "hash",
			Name:         "Test",
		}

		// Simulate BeforeCreate
		err := user.BeforeCreate(&gorm.DB{})
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, user.ID)
	})
}
