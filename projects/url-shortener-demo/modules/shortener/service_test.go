package shortener

import (
	"testing"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid https", "https://example.com", false},
		{"valid http", "http://example.com", false},
		{"valid with path", "https://example.com/path/to/page", false},
		{"valid with query", "https://example.com?q=test", false},
		{"valid with fragment", "https://example.com#section", false},
		{"empty url", "", true},
		{"no scheme", "example.com", true},
		{"ftp scheme", "ftp://example.com", true},
		{"file scheme", "file:///etc/passwd", true},
		{"javascript scheme", "javascript:alert(1)", true},
		{"no host", "https://", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateURL(tc.url)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateURL(%q) error = %v, wantErr %v", tc.url, err, tc.wantErr)
			}
		})
	}
}

func TestValidateShortCode(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{"valid alphanumeric", "abc123XY", false},
		{"valid short", "a", false},
		{"valid numbers only", "12345678", false},
		{"valid letters only", "abcdefgh", false},
		{"empty code", "", true},
		{"too long", "abcdefghijklmnopqrs", true},
		{"special chars", "abc-123", true},
		{"spaces", "abc 123", true},
		{"unicode", "日本語", true},
		{"underscore", "abc_123", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateShortCode(tc.code)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateShortCode(%q) error = %v, wantErr %v", tc.code, err, tc.wantErr)
			}
		})
	}
}
