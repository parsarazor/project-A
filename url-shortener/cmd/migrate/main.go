package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/url-shortener/internal/config"
	"github.com/url-shortener/internal/database"
	"github.com/url-shortener/internal/model"
	"github.com/url-shortener/internal/repository"
	"github.com/url-shortener/internal/service"
)

func main() {
	// Define command-line flags
	seedAdmin := flag.Bool("seed-admin", false, "Create an admin user")
	adminEmail := flag.String("admin-email", "admin@example.com", "Admin email address")
	adminPassword := flag.String("admin-password", "", "Admin password (falls back to the ADMIN_PASSWORD environment variable; required and must be strong outside development)")
	adminName := flag.String("admin-name", "Administrator", "Admin display name")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Fail fast on an unsafe admin password before touching the database
	seedPassword := ""
	if *seedAdmin {
		seedPassword, err = resolveAdminPassword(cfg.App.Env, *adminPassword, os.Getenv("ADMIN_PASSWORD"))
		if err != nil {
			log.Fatalf("Invalid admin seed configuration: %v", err)
		}
	}

	// Connect to database
	if err := database.Connect(cfg); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run migrations
	log.Println("Running database migrations...")
	if err := runMigrations(); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("Migrations completed successfully!")

	// Seed admin user if requested
	if *seedAdmin {
		log.Println("Creating admin user...")
		if err := seedAdminUser(cfg, *adminEmail, seedPassword, *adminName); err != nil {
			log.Fatalf("Failed to create admin user: %v", err)
		}
		log.Printf("Admin user created successfully! Email: %s", *adminEmail)
	}

	log.Println("Database setup completed!")
}

// runMigrations runs all database migrations
func runMigrations() error {
	db := database.GetDB()

	// AutoMigrate will create tables, missing foreign keys, constraints, columns and indexes
	err := db.AutoMigrate(
		&model.User{},
		&model.URL{},
	)
	if err != nil {
		return fmt.Errorf("auto migration failed: %w", err)
	}

	// Create indexes if they don't exist
	// Index on users.email (already handled by uniqueIndex in model)
	// Index on urls.short_code (already handled by uniqueIndex in model)
	// Index on urls.user_id
	if !db.Migrator().HasIndex(&model.URL{}, "idx_urls_user_id") {
		if err := db.Migrator().CreateIndex(&model.URL{}, "UserID"); err != nil {
			log.Printf("Warning: Could not create index on urls.user_id: %v", err)
		}
	}

	return nil
}

// seedAdminUser creates an admin user if one doesn't exist
func seedAdminUser(cfg *config.Config, email, password, name string) error {
	db := database.GetDB()

	// Initialize repository and service
	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepo, cfg)

	// Check if admin already exists
	exists, err := userRepo.ExistsByEmail(email)
	if err != nil {
		return fmt.Errorf("failed to check for existing admin: %w", err)
	}

	if exists {
		log.Printf("Admin user with email %s already exists, skipping creation", email)
		return nil
	}

	// Create admin user
	_, err = authService.CreateAdmin(email, password, name)
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	return nil
}

const (
	// devAdminPassword keeps local development ergonomic: `make seed-admin`
	// still works with no flags when APP_ENV=development. It is rejected
	// everywhere else.
	devAdminPassword = "admin123456"

	// minAdminPasswordLen is the minimum admin password length outside
	// development.
	minAdminPasswordLen = 12
)

// knownWeakAdminPasswords are publicly known values that must never seed the
// admin account outside development.
var knownWeakAdminPasswords = map[string]bool{
	devAdminPassword:     true, // the former in-code default
	"admin1234567890":    true,
	"administrator":      true,
	"password123456":     true,
	"password1234567890": true,
}

// resolveAdminPassword resolves the admin seed password and validates it.
// Precedence: the -admin-password flag, then the ADMIN_PASSWORD environment
// variable. Outside development, seeding requires an explicitly provided
// password of at least minAdminPasswordLen characters that is not publicly
// known — the same fail-fast pattern as JWT_SECRET validation in
// internal/config. In development an empty password falls back to
// devAdminPassword so the default seed flow keeps working.
func resolveAdminPassword(env, flagPassword, envPassword string) (string, error) {
	password := flagPassword
	if password == "" {
		password = envPassword
	}

	if env == "development" {
		if password == "" {
			log.Println("Warning: no admin password provided; using the development-only default. Never use it outside development.")
			return devAdminPassword, nil
		}
		return password, nil
	}

	if password == "" {
		return "", fmt.Errorf("no admin password provided: set -admin-password or ADMIN_PASSWORD (at least %d characters) when APP_ENV=%q", minAdminPasswordLen, env)
	}
	if len(password) < minAdminPasswordLen {
		return "", fmt.Errorf("admin password is too short: use at least %d characters when APP_ENV=%q (generate one with: openssl rand -base64 24)", minAdminPasswordLen, env)
	}
	if knownWeakAdminPasswords[password] {
		return "", fmt.Errorf("admin password is a publicly known value (generate one with: openssl rand -base64 24)")
	}
	return password, nil
}

// printUsage prints usage instructions
func printUsage() {
	fmt.Println("URL Shortener Database Migration Tool")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  go run cmd/migrate/main.go [flags]")
	fmt.Println("")
	fmt.Println("Flags:")
	flag.PrintDefaults()
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  # Run migrations only")
	fmt.Println("  go run cmd/migrate/main.go")
	fmt.Println("")
	fmt.Println("  # Run migrations and create admin user")
	fmt.Println("  go run cmd/migrate/main.go -seed-admin")
	fmt.Println("")
	fmt.Println("  # Run migrations and create admin with custom credentials")
	fmt.Println("  go run cmd/migrate/main.go -seed-admin -admin-email=admin@mysite.com -admin-password=securepass123 -admin-name=\"Super Admin\"")
}

func init() {
	flag.Usage = func() {
		printUsage()
		os.Exit(0)
	}
}
