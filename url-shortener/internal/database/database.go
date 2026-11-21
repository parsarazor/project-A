package database

import (
	"fmt"
	"log"

	"github.com/url-shortener/internal/config"
	"github.com/url-shortener/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Connect establishes connection to the PostgreSQL database
func Connect(cfg *config.Config) error {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	var gormConfig *gorm.Config

	// TranslateError maps driver-level errors (e.g. unique constraint
	// violations) to gorm.ErrDuplicatedKey / gorm.ErrRecordNotFound so the
	// repository layer can handle them without importing the driver.
	if cfg.IsProduction() {
		gormConfig = &gorm.Config{
			Logger:         logger.Default.LogMode(logger.Silent),
			TranslateError: true,
		}
	} else {
		gormConfig = &gorm.Config{
			Logger:         logger.Default.LogMode(logger.Info),
			TranslateError: true,
		}
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying DB: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	DB = db
	log.Println("Database connected successfully")

	return nil
}

// Migrate runs database migrations
func Migrate() error {
	log.Println("Running database migrations...")

	err := DB.AutoMigrate(
		&model.User{},
		&model.URL{},
	)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// Close closes the database connection
func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}
