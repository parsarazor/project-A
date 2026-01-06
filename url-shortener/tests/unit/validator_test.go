package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/url-shortener/internal/utils"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		hasError bool
	}{
		{
			name:     "valid email",
			email:    "test@example.com",
			hasError: false,
		},
		{
			name:     "valid email with subdomain",
			email:    "user@mail.example.com",
			hasError: false,
		},
		{
			name:     "valid email with plus",
			email:    "user+tag@example.com",
			hasError: false,
		},
		{
			name:     "empty email",
			email:    "",
			hasError: true,
		},
		{
			name:     "invalid email - no @",
			email:    "testexample.com",
			hasError: true,
		},
		{
			name:     "invalid email - no domain",
			email:    "test@",
			hasError: true,
		},
		{
			name:     "invalid email - no username",
			email:    "@example.com",
			hasError: true,
		},
		{
			name:     "invalid email - spaces",
			email:    "test @example.com",
			hasError: true,
		},
		{
			name:     "invalid email - no TLD",
			email:    "test@example",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := utils.NewValidator()
			v.ValidateEmail("email", tt.email)

			if tt.hasError {
				assert.True(t, v.HasErrors(), "expected validation error for email: %s", tt.email)
			} else {
				assert.False(t, v.HasErrors(), "unexpected validation error for email: %s", tt.email)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		hasError bool
	}{
		{
			name:     "valid password - 8 chars",
			password: "password",
			hasError: false,
		},
		{
			name:     "valid password - long",
			password: "verysecurepassword123!@#",
			hasError: false,
		},
		{
			name:     "invalid password - empty",
			password: "",
			hasError: true,
		},
		{
			name:     "invalid password - too short",
			password: "pass",
			hasError: true,
		},
		{
			name:     "invalid password - 7 chars",
			password: "passwor",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := utils.NewValidator()
			v.ValidatePassword("password", tt.password)

			if tt.hasError {
				assert.True(t, v.HasErrors(), "expected validation error for password: %s", tt.password)
			} else {
				assert.False(t, v.HasErrors(), "unexpected validation error for password: %s", tt.password)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		hasError bool
	}{
		{
			name:     "valid URL - http",
			url:      "http://example.com",
			hasError: false,
		},
		{
			name:     "valid URL - https",
			url:      "https://example.com",
			hasError: false,
		},
		{
			name:     "valid URL - with path",
			url:      "https://example.com/path/to/page",
			hasError: false,
		},
		{
			name:     "valid URL - with query params",
			url:      "https://example.com/search?q=test&page=1",
			hasError: false,
		},
		{
			name:     "valid URL - with port",
			url:      "https://example.com:8080/api",
			hasError: false,
		},
		{
			name:     "valid URL - subdomain",
			url:      "https://www.subdomain.example.com",
			hasError: false,
		},
		{
			name:     "invalid URL - empty",
			url:      "",
			hasError: true,
		},
		{
			name:     "invalid URL - no scheme",
			url:      "example.com",
			hasError: true,
		},
		{
			name:     "invalid URL - ftp scheme",
			url:      "ftp://example.com",
			hasError: true,
		},
		{
			name:     "invalid URL - no host",
			url:      "http://",
			hasError: true,
		},
		{
			name:     "invalid URL - javascript",
			url:      "javascript:alert('xss')",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := utils.NewValidator()
			v.ValidateURL("url", tt.url)

			if tt.hasError {
				assert.True(t, v.HasErrors(), "expected validation error for URL: %s", tt.url)
			} else {
				assert.False(t, v.HasErrors(), "unexpected validation error for URL: %s", tt.url)
			}
		})
	}
}

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		hasError bool
	}{
		{
			name:     "valid - has value",
			value:    "some value",
			hasError: false,
		},
		{
			name:     "invalid - empty string",
			value:    "",
			hasError: true,
		},
		{
			name:     "invalid - only spaces",
			value:    "   ",
			hasError: true,
		},
		{
			name:     "invalid - tabs only",
			value:    "\t\t",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := utils.NewValidator()
			v.ValidateRequired("field", tt.value)

			if tt.hasError {
				assert.True(t, v.HasErrors(), "expected validation error for value: '%s'", tt.value)
			} else {
				assert.False(t, v.HasErrors(), "unexpected validation error for value: '%s'", tt.value)
			}
		})
	}
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"valid http URL", "http://example.com", true},
		{"valid https URL", "https://example.com", true},
		{"valid URL with path", "https://example.com/path", true},
		{"empty URL", "", false},
		{"no scheme", "example.com", false},
		{"invalid scheme", "ftp://example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsValidURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		{"valid email", "test@example.com", true},
		{"empty email", "", false},
		{"invalid - no @", "testexample.com", false},
		{"invalid - no domain", "test@", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsValidEmail(tt.email)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no change needed", "hello world", "hello world"},
		{"trim leading spaces", "  hello", "hello"},
		{"trim trailing spaces", "hello  ", "hello"},
		{"trim both", "  hello  ", "hello"},
		{"normalize multiple spaces", "hello    world", "hello world"},
		{"complex case", "  hello   world  foo   bar  ", "hello world foo bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.SanitizeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidationErrors(t *testing.T) {
	t.Run("multiple errors", func(t *testing.T) {
		v := utils.NewValidator()
		v.ValidateEmail("email", "invalid")
		v.ValidatePassword("password", "short")

		assert.True(t, v.HasErrors())
		errors := v.Errors()
		assert.Equal(t, 2, len(errors))
	})

	t.Run("error string format", func(t *testing.T) {
		v := utils.NewValidator()
		v.AddError("field1", "error message 1")
		v.AddError("field2", "error message 2")

		errStr := v.Errors().Error()
		assert.Contains(t, errStr, "field1")
		assert.Contains(t, errStr, "error message 1")
		assert.Contains(t, errStr, "field2")
		assert.Contains(t, errStr, "error message 2")
	})

	t.Run("empty errors", func(t *testing.T) {
		v := utils.NewValidator()
		assert.False(t, v.HasErrors())
		assert.Empty(t, v.Errors())
	})
}
