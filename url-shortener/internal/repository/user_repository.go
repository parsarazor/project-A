package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/url-shortener/internal/model"
	"gorm.io/gorm"
)

// Common errors
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user with this email already exists")
)

// UserRepository handles database operations for users
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new UserRepository instance
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user in the database
func (r *UserRepository) Create(user *model.User) error {
	// Check if user with email already exists
	existing, err := r.FindByEmail(user.Email)
	if err == nil && existing != nil {
		return ErrUserAlreadyExists
	}

	result := r.db.Create(user)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// FindByID finds a user by their ID
func (r *UserRepository) FindByID(id uuid.UUID) (*model.User, error) {
	var user model.User
	result := r.db.Where("id = ?", id).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, result.Error
	}
	return &user, nil
}

// FindByEmail finds a user by their email address
func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	result := r.db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, result.Error
	}
	return &user, nil
}

// Update updates a user in the database
func (r *UserRepository) Update(user *model.User) error {
	result := r.db.Save(user)
	return result.Error
}

// Delete soft deletes a user from the database
func (r *UserRepository) Delete(id uuid.UUID) error {
	result := r.db.Delete(&model.User{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// FindAll returns all users with pagination
func (r *UserRepository) FindAll(page, pageSize int) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	// Count total records
	r.db.Model(&model.User{}).Count(&total)

	// Calculate offset
	offset := (page - 1) * pageSize

	// Fetch users with pagination
	result := r.db.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users)
	if result.Error != nil {
		return nil, 0, result.Error
	}

	return users, total, nil
}

// ExistsByEmail checks if a user with the given email exists
func (r *UserRepository) ExistsByEmail(email string) (bool, error) {
	var count int64
	result := r.db.Model(&model.User{}).Where("email = ?", email).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// UpdatePassword updates only the password hash for a user
func (r *UserRepository) UpdatePassword(id uuid.UUID, passwordHash string) error {
	result := r.db.Model(&model.User{}).Where("id = ?", id).Update("password_hash", passwordHash)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

// UpdateRole updates the role of a user
func (r *UserRepository) UpdateRole(id uuid.UUID, role model.Role) error {
	result := r.db.Model(&model.User{}).Where("id = ?", id).Update("role", role)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}
