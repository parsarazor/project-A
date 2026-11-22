package repository

import (
	"errors"

	"github.com/google/uuid"
	"github.com/url-shortener/internal/core/ports"
	"github.com/url-shortener/internal/model"
	"gorm.io/gorm"
)

var ErrURLNotFound = ports.ErrNotFound

// URLRepository handles database operations for URLs.
type URLRepository struct {
	db *gorm.DB
}

func NewURLRepository(db *gorm.DB) *URLRepository {
	return &URLRepository{db: db}
}

func (r *URLRepository) Create(url *model.URL) error {
	if err := r.db.Create(url).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ports.ErrAlreadyExist
		}
		return err
	}
	return nil
}

func (r *URLRepository) FindByID(id uuid.UUID) (*model.URL, error) {
	var url model.URL
	err := r.db.Where("id = ?", id).First(&url).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrURLNotFound
	}
	if err != nil {
		return nil, err
	}
	return &url, nil
}

func (r *URLRepository) FindByIDWithUser(id uuid.UUID) (*model.URL, error) {
	var url model.URL
	err := r.db.Preload("User").Where("id = ?", id).First(&url).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrURLNotFound
	}
	if err != nil {
		return nil, err
	}
	return &url, nil
}

func (r *URLRepository) FindByShortCode(shortCode string) (*model.URL, error) {
	var url model.URL
	err := r.db.Where("short_code = ? AND is_active = ?", shortCode, true).First(&url).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrURLNotFound
	}
	if err != nil {
		return nil, err
	}
	return &url, nil
}

func (r *URLRepository) FindByUserID(userID uuid.UUID) ([]model.URL, error) {
	var urls []model.URL
	if err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&urls).Error; err != nil {
		return nil, err
	}
	return urls, nil
}

func (r *URLRepository) FindAll(page, pageSize int) ([]model.URL, int64, error) {
	var urls []model.URL
	var total int64

	if err := r.db.Model(&model.URL{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := r.db.Preload("User").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&urls).Error
	if err != nil {
		return nil, 0, err
	}

	return urls, total, nil
}

func (r *URLRepository) FindAllWithUser() ([]model.URL, error) {
	var urls []model.URL
	if err := r.db.Preload("User").Order("created_at DESC").Find(&urls).Error; err != nil {
		return nil, err
	}
	return urls, nil
}

func (r *URLRepository) Update(url *model.URL) error {
	return r.db.Save(url).Error
}

func (r *URLRepository) Delete(id uuid.UUID) error {
	result := r.db.Delete(&model.URL{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrURLNotFound
	}
	return nil
}

func (r *URLRepository) IncrementClickCount(id uuid.UUID) error {
	return r.db.Model(&model.URL{}).Where("id = ?", id).UpdateColumn("click_count", gorm.Expr("click_count + ?", 1)).Error
}

func (r *URLRepository) IncrementClickCountByShortCode(shortCode string) error {
	return r.db.Model(&model.URL{}).Where("short_code = ?", shortCode).UpdateColumn("click_count", gorm.Expr("click_count + ?", 1)).Error
}

func (r *URLRepository) Deactivate(id uuid.UUID) error {
	result := r.db.Model(&model.URL{}).Where("id = ?", id).Update("is_active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrURLNotFound
	}
	return nil
}

func (r *URLRepository) Activate(id uuid.UUID) error {
	result := r.db.Model(&model.URL{}).Where("id = ?", id).Update("is_active", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrURLNotFound
	}
	return nil
}

func (r *URLRepository) CountByUserID(userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&model.URL{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *URLRepository) GetTotalClicks(userID uuid.UUID) (int64, error) {
	var total int64
	err := r.db.Model(&model.URL{}).Where("user_id = ?", userID).Select("COALESCE(SUM(click_count), 0)").Scan(&total).Error
	return total, err
}
