package unit

import (
	"testing"

	"github.com/url-shortener/internal/utils"
)

func TestGenerateShortCode(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantLen int
	}{
		{
			name:    "default length",
			length:  0,
			wantLen: utils.DefaultCodeLength,
		},
		{
			name:    "custom length 8",
			length:  8,
			wantLen: 8,
		},
		{
			name:    "custom length 4",
			length:  4,
			wantLen: 4,
		},
		{
			name:    "negative length defaults",
			length:  -1,
			wantLen: utils.DefaultCodeLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := utils.GenerateShortCode(tt.length)
			if err != nil {
				t.Errorf("GenerateShortCode() error = %v", err)
				return
			}
			if len(code) != tt.wantLen {
				t.Errorf("GenerateShortCode() length = %d, want %d", len(code), tt.wantLen)
			}
		})
	}
}

func TestGenerateShortCodeUniqueness(t *testing.T) {
	// Generate multiple codes and check for uniqueness
	codes := make(map[string]bool)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		code, err := utils.GenerateShortCode(6)
		if err != nil {
			t.Fatalf("GenerateShortCode() error = %v", err)
		}

		if codes[code] {
			t.Errorf("Duplicate code generated: %s", code)
		}
		codes[code] = true
	}
}

func TestGenerateShortCodeCharset(t *testing.T) {
	// Generate multiple codes and verify they only contain valid characters
	for i := 0; i < 100; i++ {
		code, err := utils.GenerateShortCode(6)
		if err != nil {
			t.Fatalf("GenerateShortCode() error = %v", err)
		}

		for _, char := range code {
			if !containsRune(utils.Charset, char) {
				t.Errorf("Code contains invalid character: %c in code %s", char, code)
			}
		}
	}
}

func TestGenerateDefaultShortCode(t *testing.T) {
	code, err := utils.GenerateDefaultShortCode()
	if err != nil {
		t.Errorf("GenerateDefaultShortCode() error = %v", err)
		return
	}

	if len(code) != utils.DefaultCodeLength {
		t.Errorf("GenerateDefaultShortCode() length = %d, want %d", len(code), utils.DefaultCodeLength)
	}
}

func TestIsValidShortCode(t *testing.T) {
	tests := []struct {
		name  string
		code  string
		valid bool
	}{
		{
			name:  "valid code",
			code:  "abc234",
			valid: true,
		},
		{
			name:  "valid code with uppercase",
			code:  "AbC234",
			valid: true,
		},
		{
			name:  "empty code",
			code:  "",
			valid: false,
		},
		{
			name:  "too long code",
			code:  "abcdefghijk",
			valid: false,
		},
		{
			name:  "code with invalid char (0)",
			code:  "abc0234",
			valid: false,
		},
		{
			name:  "code with invalid char (O)",
			code:  "abcO234",
			valid: false,
		},
		{
			name:  "code with invalid char (I)",
			code:  "abcI234",
			valid: false,
		},
		{
			name:  "code with invalid char (l)",
			code:  "abcd234l",
			valid: false,
		},
		{
			name:  "code with invalid char (1)",
			code:  "abc123",
			valid: false,
		},
		{
			name:  "code with special char",
			code:  "abc!23",
			valid: false,
		},
		{
			name:  "code with space",
			code:  "abc 23",
			valid: false,
		},
		{
			name:  "single character",
			code:  "a",
			valid: true,
		},
		{
			name:  "max length 10",
			code:  "abcdefghJK",
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utils.IsValidShortCode(tt.code); got != tt.valid {
				t.Errorf("IsValidShortCode(%q) = %v, want %v", tt.code, got, tt.valid)
			}
		})
	}
}

// containsRune checks if a string contains a specific rune
func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}
