package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/url-shortener/internal/config"
	"github.com/url-shortener/internal/model"
	"github.com/url-shortener/internal/repository"
	"github.com/url-shortener/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

// Common auth errors
var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user with this email already exists")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

// JWTClaims represents the claims stored in JWT tokens
type JWTClaims struct {
	UserID uuid.UUID  `json:"user_id"`
	Email  string     `json:"email"`
	Role   model.Role `json:"role"`
	jwt.RegisteredClaims
}

// AuthService handles authentication business logic
type AuthService struct {
	userRepo  *repository.UserRepository
	jwtSecret []byte
	jwtExpiry time.Duration
}

// NewAuthService creates a new AuthService instance
func NewAuthService(userRepo *repository.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: []byte(cfg.JWT.Secret),
		jwtExpiry: cfg.JWT.Expiry,
	}
}

// Register registers a new user
func (s *AuthService) Register(req *model.RegisterRequest) (*model.User, error) {
	// Validate input
	validator := utils.NewValidator()
	validator.ValidateEmail("email", req.Email)
	validator.ValidatePassword("password", req.Password)
	validator.ValidateRequired("name", req.Name)

	if validator.HasErrors() {
		return nil, validator.Errors()
	}

	// Sanitize input
	req.Email = utils.SanitizeString(req.Email)
	req.Name = utils.SanitizeString(req.Name)

	// Check if user already exists
	exists, err := s.userRepo.ExistsByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserAlreadyExists
	}

	// Hash password
	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &model.User{
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Name:         req.Name,
		Role:         model.RoleUser,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

// Login authenticates a user and returns a JWT token
func (s *AuthService) Login(req *model.LoginRequest) (*model.LoginResponse, error) {
	// Validate input
	validator := utils.NewValidator()
	validator.ValidateEmail("email", req.Email)
	validator.ValidateRequired("password", req.Password)

	if validator.HasErrors() {
		return nil, validator.Errors()
	}

	// Find user by email
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Verify password
	if !s.checkPassword(req.Password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// Generate JWT token
	token, expiresIn, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &model.LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetUserByID retrieves a user by their ID
func (s *AuthService) GetUserByID(id uuid.UUID) (*model.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// hashPassword hashes a password using bcrypt
func (s *AuthService) hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// checkPassword compares a password with a hash
func (s *AuthService) checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// generateToken generates a JWT token for a user
func (s *AuthService) generateToken(user *model.User) (string, int64, error) {
	expiresAt := time.Now().Add(s.jwtExpiry)
	expiresIn := int64(s.jwtExpiry.Seconds())

	claims := &JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "url-shortener",
			Subject:   user.ID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", 0, err
	}

	return tokenString, expiresIn, nil
}

// ChangePassword changes a user's password
func (s *AuthService) ChangePassword(userID uuid.UUID, currentPassword, newPassword string) error {
	// Validate new password
	validator := utils.NewValidator()
	validator.ValidatePassword("new_password", newPassword)

	if validator.HasErrors() {
		return validator.Errors()
	}

	// Get user
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	// Verify current password
	if !s.checkPassword(currentPassword, user.PasswordHash) {
		return ErrInvalidCredentials
	}

	// Hash new password
	hashedPassword, err := s.hashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update password
	return s.userRepo.UpdatePassword(userID, hashedPassword)
}

// CreateAdmin creates an admin user (for initial setup)
func (s *AuthService) CreateAdmin(email, password, name string) (*model.User, error) {
	// Validate input
	validator := utils.NewValidator()
	validator.ValidateEmail("email", email)
	validator.ValidatePassword("password", password)
	validator.ValidateRequired("name", name)

	if validator.HasErrors() {
		return nil, validator.Errors()
	}

	// Hash password
	hashedPassword, err := s.hashPassword(password)
	if err != nil {
		return nil, err
	}

	// Create admin user
	user := &model.User{
		Email:        email,
		PasswordHash: hashedPassword,
		Name:         name,
		Role:         model.RoleAdmin,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}
