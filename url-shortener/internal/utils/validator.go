package utils

import (
	"net/url"
	"regexp"
	"strings"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors holds multiple validation errors
type ValidationErrors []ValidationError

// Error implements the error interface
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}
	var messages []string
	for _, e := range ve {
		messages = append(messages, e.Field+": "+e.Message)
	}
	return strings.Join(messages, "; ")
}

// HasErrors returns true if there are validation errors
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

// Validator handles input validation
type Validator struct {
	errors ValidationErrors
}

// NewValidator creates a new Validator instance
func NewValidator() *Validator {
	return &Validator{
		errors: make(ValidationErrors, 0),
	}
}

// AddError adds a validation error
func (v *Validator) AddError(field, message string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

// HasErrors returns true if there are validation errors
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// Errors returns all validation errors
func (v *Validator) Errors() ValidationErrors {
	return v.errors
}

// ValidateEmail validates an email address
func (v *Validator) ValidateEmail(field, value string) {
	if value == "" {
		v.AddError(field, "email is required")
		return
	}

	// Basic email regex pattern
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(value) {
		v.AddError(field, "invalid email format")
	}
}

// ValidatePassword validates a password
func (v *Validator) ValidatePassword(field, value string) {
	if value == "" {
		v.AddError(field, "password is required")
		return
	}

	if len(value) < 8 {
		v.AddError(field, "password must be at least 8 characters long")
	}
}

// ValidateRequired validates that a field is not empty
func (v *Validator) ValidateRequired(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.AddError(field, field+" is required")
	}
}

// ValidateMinLength validates minimum length
func (v *Validator) ValidateMinLength(field, value string, min int) {
	if len(value) < min {
		v.AddError(field, field+" must be at least "+string(rune(min+'0'))+" characters long")
	}
}

// ValidateMaxLength validates maximum length
func (v *Validator) ValidateMaxLength(field, value string, max int) {
	if len(value) > max {
		v.AddError(field, field+" must be at most "+string(rune(max+'0'))+" characters long")
	}
}

// ValidateURL validates a URL
func (v *Validator) ValidateURL(field, value string) {
	if value == "" {
		v.AddError(field, "URL is required")
		return
	}

	parsedURL, err := url.ParseRequestURI(value)
	if err != nil {
		v.AddError(field, "invalid URL format")
		return
	}

	// Check for valid scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		v.AddError(field, "URL must use http or https scheme")
		return
	}

	// Check for valid host
	if parsedURL.Host == "" {
		v.AddError(field, "URL must have a valid host")
	}
}

// IsValidURL checks if a URL is valid without adding errors
func IsValidURL(rawURL string) bool {
	if rawURL == "" {
		return false
	}

	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false
	}

	if parsedURL.Host == "" {
		return false
	}

	return true
}

// IsValidEmail checks if an email is valid without adding errors
func IsValidEmail(email string) bool {
	if email == "" {
		return false
	}
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// SanitizeString removes leading/trailing whitespace and normalizes spaces
func SanitizeString(s string) string {
	// Trim whitespace
	s = strings.TrimSpace(s)
	// Replace multiple spaces with single space
	spaceRegex := regexp.MustCompile(`\s+`)
	s = spaceRegex.ReplaceAllString(s, " ")
	return s
}
