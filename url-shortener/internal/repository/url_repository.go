package repository

import (
	"github.com/google/uuid"
	"github.com/url-shortener/internal/model"
	"gorm.io/gorm"
)

// URLRepository handles database operations for URLs
type URLRepository struct {
	db *gorm.DB
}

// NewURLRepository creates a new URL repository instance
func NewURLRepository(db *gorm.DB) *URLRepository {
	return &URLRepository{db: db}
}

// Create creates a new URL record in the database
func (r *URLRepository) Create(url *model.URL) error {
	return r.db.Create(url).Error
}

// FindByID finds a URL by its ID
func (r *URLRepository) FindByID(id uuid.UUID) (*model.URL, error) {
	var url model.URL
	err := r.db.Where("id = ?", id).First(&url).Error
	if err != nil {
		return nil, err
	}
	return &url, nil
}

// FindByIDWithUser finds a URL by its ID and preloads the user
func (r *URLRepository) FindByIDWithUser(id uuid.UUID) (*model.URL, error) {
	var url model.URL
	err := r.db.Preload("User").Where("id = ?", id).First(&url).Error
	if err != nil {
		return nil, err
	}
	return &url, nil
}

// FindByShortCode finds a URL by its short code
func (r *URLRepository) FindByShortCode(shortCode string) (*model.URL, error) {
	var url model.URL
	err := r.db.Where("short_code = ? AND is_active = ?", shortCode, true).First(&url).Error
	if err != nil {
		return nil, err
	}
	return &url, nil
}

// FindByUserID finds all URLs belonging to a user
func (r *URLRepository) FindByUserID(userID uuid.UUID) ([]model.URL, error) {
	var urls []model.URL
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&urls).Error
	if err != nil {
		return nil, err
	}
	return urls, nil
}

// FindAll returns all URLs with pagination
func (r *URLRepository) FindAll(page, pageSize int) ([]model.URL, int64, error) {
	var urls []model.URL
	var total int64

	// Get total count
	if err := r.db.Model(&model.URL{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get paginated results with user preloaded
	err := r.db.Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&urls).Error
	if err != nil {
		return nil, 0, err
	}

	return urls, total, nil
}

// FindAllWithUser returns all URLs with user information
func (r *URLRepository) FindAllWithUser() ([]model.URL, error) {
	var urls []model.URL
	err := r.db.Preload("User").Order("created_at DESC").Find(&urls).Error
	if err != nil {
		return nil, err
	}
	return urls, nil
}

// Update updates a URL record
func (r *URLRepository) Update(url *model.URL) error {
	return r.db.Save(url).Error
}

// Delete deletes a URL record by ID
func (r *URLRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&model.URL{}, "id = ?", id).Error
}

// IncrementClickCount increments the click count for a URL
func (r *URLRepository) IncrementClickCount(id uuid.UUID) error {
	return r.db.Model(&model.URL{}).
		Where("id = ?", id).
		UpdateColumn("click_count", gorm.Expr("click_count + ?", 1)).
		Error
}

// IncrementClickCountByShortCode increments click count using short code
func (r *URLRepository) IncrementClickCountByShortCode(shortCode string) error {
	return r.db.Model(&model.URL{}).
		Where("short_code = ?", shortCode).
		UpdateColumn("click_count", gorm.Expr("click_count + ?", 1)).
		Error
}

// ShortCodeExists checks if a short code already exists
func (r *URLRepository) ShortCodeExists(shortCode string) (bool, error) {
	var count int64
	err := r.db.Model(&model.URL{}).Where("short_code = ?", shortCode).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Deactivate deactivates a URL (soft disable without deleting)
func (r *URLRepository) Deactivate(id uuid.UUID) error {
	return r.db.Model(&model.URL{}).
		Where("id = ?", id).
		Update("is_active", false).
		Error
}

// Activate activates a URL
func (r *URLRepository) Activate(id uuid.UUID) error {
	return r.db.Model(&model.URL{}).
		Where("id = ?", id).
		Update("is_active", true).
		Error
}

// CountByUserID returns the total number of URLs created by a user
func (r *URLRepository) CountByUserID(userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&model.URL{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

// GetTotalClicks returns the total number of clicks for all URLs of a user
func (r *URLRepository) GetTotalClicks(userID uuid.UUID) (int64, error) {
	var total int64
	err := r.db.Model(&model.URL{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(click_count), 0)").
		Scan(&total).Error
	return total, err
}
