package shortener

import (
	"testing"
)

func TestGenerateShortCode(t *testing.T) {
	tests := []struct {
		name           string
		length         int
		expectedLength int
	}{
		{
			name:           "default length when zero",
			length:         0,
			expectedLength: DefaultCodeLength,
		},
		{
			name:           "default length when negative",
			length:         -1,
			expectedLength: DefaultCodeLength,
		},
		{
			name:           "custom length 5",
			length:         5,
			expectedLength: 5,
		},
		{
			name:           "custom length 10",
			length:         10,
			expectedLength: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := GenerateShortCode(tt.length)
			if err != nil {
				t.Fatalf("GenerateShortCode() error = %v", err)
			}

			if len(code) != tt.expectedLength {
				t.Errorf("GenerateShortCode() length = %d, want %d", len(code), tt.expectedLength)
			}

			// Verify all characters are valid base62
			if !IsValidShortCode(code) {
				t.Errorf("GenerateShortCode() generated invalid code: %s", code)
			}
		})
	}
}

func TestGenerateShortCode_Uniqueness(t *testing.T) {
	// Generate multiple codes and check they are unique
	codes := make(map[string]bool)
	count := 100

	for i := 0; i < count; i++ {
		code, err := GenerateShortCode(DefaultCodeLength)
		if err != nil {
			t.Fatalf("GenerateShortCode() error = %v", err)
		}

		if codes[code] {
			t.Errorf("GenerateShortCode() generated duplicate code: %s", code)
		}
		codes[code] = true
	}
}

func TestGenerateShortCode_Base62Characters(t *testing.T) {
	// Generate a long code and verify all characters are from base62 set
	code, err := GenerateShortCode(100)
	if err != nil {
		t.Fatalf("GenerateShortCode() error = %v", err)
	}

	for i, c := range code {
		if !isAlphanumeric(c) {
			t.Errorf("GenerateShortCode() generated non-base62 character at position %d: %c", i, c)
		}
	}
}

func TestIsValidShortCode(t *testing.T) {
	tests := []struct {
		name  string
		code  string
		valid bool
	}{
		{
			name:  "valid alphanumeric lowercase",
			code:  "abc123",
			valid: true,
		},
		{
			name:  "valid alphanumeric uppercase",
			code:  "ABC123",
			valid: true,
		},
		{
			name:  "valid mixed case",
			code:  "AbCdEf123",
			valid: true,
		},
		{
			name:  "valid numbers only",
			code:  "123456",
			valid: true,
		},
		{
			name:  "valid letters only",
			code:  "abcDEF",
			valid: true,
		},
		{
			name:  "valid max length 20",
			code:  "12345678901234567890",
			valid: true,
		},
		{
			name:  "empty string",
			code:  "",
			valid: false,
		},
		{
			name:  "too long (21 chars)",
			code:  "123456789012345678901",
			valid: false,
		},
		{
			name:  "contains underscore",
			code:  "abc_123",
			valid: false,
		},
		{
			name:  "contains hyphen",
			code:  "abc-123",
			valid: false,
		},
		{
			name:  "contains space",
			code:  "abc 123",
			valid: false,
		},
		{
			name:  "contains special characters",
			code:  "abc@123",
			valid: false,
		},
		{
			name:  "contains unicode",
			code:  "abc日本語",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidShortCode(tt.code)
			if got != tt.valid {
				t.Errorf("IsValidShortCode(%q) = %v, want %v", tt.code, got, tt.valid)
			}
		})
	}
}

func TestIsAlphanumeric(t *testing.T) {
	tests := []struct {
		char  rune
		valid bool
	}{
		{'a', true},
		{'z', true},
		{'A', true},
		{'Z', true},
		{'0', true},
		{'9', true},
		{'_', false},
		{'-', false},
		{' ', false},
		{'@', false},
		{'日', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.char), func(t *testing.T) {
			got := isAlphanumeric(tt.char)
			if got != tt.valid {
				t.Errorf("isAlphanumeric(%q) = %v, want %v", tt.char, got, tt.valid)
			}
		})
	}
}

func BenchmarkGenerateShortCode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateShortCode(DefaultCodeLength)
	}
}

func BenchmarkIsValidShortCode(b *testing.B) {
	code := "abc123DEF"
	for i := 0; i < b.N; i++ {
		_ = IsValidShortCode(code)
	}
}
