package usecase

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/url-shortener/internal/core/ports"
	"github.com/url-shortener/internal/model"
	"github.com/url-shortener/internal/utils"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user with this email already exists")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type AuthUseCase struct {
	userRepo ports.UserRepository
	hasher   ports.PasswordHasher
	tokens   ports.TokenManager
}

func NewAuthUseCase(userRepo ports.UserRepository, hasher ports.PasswordHasher, tokens ports.TokenManager) *AuthUseCase {
	return &AuthUseCase{userRepo: userRepo, hasher: hasher, tokens: tokens}
}

func (uc *AuthUseCase) Register(req *model.RegisterRequest) (*model.User, error) {
	validator := utils.NewValidator()
	validator.ValidateEmail("email", req.Email)
	validator.ValidatePassword("password", req.Password)
	validator.ValidateRequired("name", req.Name)
	if validator.HasErrors() {
		return nil, validator.Errors()
	}

	req.Email = utils.SanitizeString(strings.ToLower(req.Email))
	req.Name = utils.SanitizeString(req.Name)

	exists, err := uc.userRepo.ExistsByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserAlreadyExists
	}

	hashedPassword, err := uc.hasher.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Name:         req.Name,
		Role:         model.RoleUser,
	}

	if err := uc.userRepo.Create(user); err != nil {
		// Concurrent duplicate insert: the unique index on email rejected
		// the write. Translate the port error to the domain error the
		// handler maps to 409 Conflict.
		if errors.Is(err, ports.ErrAlreadyExist) {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}
	return user, nil
}

func (uc *AuthUseCase) Login(req *model.LoginRequest) (*model.LoginResponse, error) {
	validator := utils.NewValidator()
	validator.ValidateEmail("email", req.Email)
	validator.ValidateRequired("password", req.Password)
	if validator.HasErrors() {
		return nil, validator.Errors()
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	user, err := uc.userRepo.FindByEmail(req.Email)
	if err != nil {
		// Only a genuine "not found" is a credentials error; any other DB
		// failure (connection loss, timeout) must surface as a 500, not a
		// misleading 401.
		if errors.Is(err, ports.ErrNotFound) || errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !uc.hasher.Compare(user.PasswordHash, req.Password) {
		return nil, ErrInvalidCredentials
	}

	token, expiresIn, err := uc.tokens.Generate(user)
	if err != nil {
		return nil, err
	}

	return &model.LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}, nil
}

func (uc *AuthUseCase) ValidateToken(tokenString string) (*ports.TokenClaims, error) {
	claims, err := uc.tokens.Validate(tokenString)
	if err != nil {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func (uc *AuthUseCase) GetUserByID(id uuid.UUID) (*model.User, error) {
	user, err := uc.userRepo.FindByID(id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (uc *AuthUseCase) ChangePassword(userID uuid.UUID, currentPassword, newPassword string) error {
	validator := utils.NewValidator()
	validator.ValidatePassword("new_password", newPassword)
	if validator.HasErrors() {
		return validator.Errors()
	}

	user, err := uc.userRepo.FindByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	if !uc.hasher.Compare(user.PasswordHash, currentPassword) {
		return ErrInvalidCredentials
	}

	hashed, err := uc.hasher.Hash(newPassword)
	if err != nil {
		return err
	}

	if err := uc.userRepo.UpdatePassword(userID, hashed); err != nil {
		return ErrUserNotFound
	}

	return nil
}

func (uc *AuthUseCase) CreateAdmin(email, password, name string) (*model.User, error) {
	validator := utils.NewValidator()
	validator.ValidateEmail("email", email)
	validator.ValidatePassword("password", password)
	validator.ValidateRequired("name", name)
	if validator.HasErrors() {
		return nil, validator.Errors()
	}

	email = utils.SanitizeString(strings.ToLower(email))
	name = utils.SanitizeString(name)

	exists, err := uc.userRepo.ExistsByEmail(email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUserAlreadyExists
	}

	hashedPassword, err := uc.hasher.Hash(password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Email:        email,
		PasswordHash: hashedPassword,
		Name:         name,
		Role:         model.RoleAdmin,
	}

	if err := uc.userRepo.Create(user); err != nil {
		if errors.Is(err, ports.ErrAlreadyExist) {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}

	return user, nil
}
