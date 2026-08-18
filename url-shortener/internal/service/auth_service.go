package service

import (
	"github.com/google/uuid"
	"github.com/url-shortener/internal/adapter/security"
	"github.com/url-shortener/internal/config"
	"github.com/url-shortener/internal/core/usecase"
	"github.com/url-shortener/internal/model"
	"github.com/url-shortener/internal/repository"
)

var (
	ErrInvalidCredentials = usecase.ErrInvalidCredentials
	ErrUserNotFound       = usecase.ErrUserNotFound
	ErrUserAlreadyExists  = usecase.ErrUserAlreadyExists
	ErrInvalidToken       = usecase.ErrInvalidToken
)

// JWTClaims is the transport-level user context stored in middleware.
type JWTClaims struct {
	UserID uuid.UUID  `json:"user_id"`
	Email  string     `json:"email"`
	Role   model.Role `json:"role"`
}

// AuthService is a compatibility wrapper over the hexagonal auth use case.
type AuthService struct {
	uc *usecase.AuthUseCase
}

func NewAuthService(userRepo *repository.UserRepository, cfg *config.Config) *AuthService {
	hasher := security.NewBcryptHasher()
	tokenManager := security.NewJWTManager(cfg.JWT.Secret, cfg.JWT.Expiry)
	uc := usecase.NewAuthUseCase(userRepo, hasher, tokenManager)
	return &AuthService{uc: uc}
}

func (s *AuthService) Register(req *model.RegisterRequest) (*model.User, error) {
	return s.uc.Register(req)
}

func (s *AuthService) Login(req *model.LoginRequest) (*model.LoginResponse, error) {
	return s.uc.Login(req)
}

func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	claims, err := s.uc.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	return &JWTClaims{UserID: claims.UserID, Email: claims.Email, Role: claims.Role}, nil
}

func (s *AuthService) GetUserByID(id uuid.UUID) (*model.User, error) {
	return s.uc.GetUserByID(id)
}

func (s *AuthService) ChangePassword(userID uuid.UUID, currentPassword, newPassword string) error {
	return s.uc.ChangePassword(userID, currentPassword, newPassword)
}

func (s *AuthService) CreateAdmin(email, password, name string) (*model.User, error) {
	return s.uc.CreateAdmin(email, password, name)
}
