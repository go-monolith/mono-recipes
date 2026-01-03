package auth

import (
	"net/mail"
	"testing"
)

func TestEmailValidation(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{
			name:  "valid email",
			email: "user@example.com",
			want:  true,
		},
		{
			name:  "valid email with subdomain",
			email: "user@mail.example.com",
			want:  true,
		},
		{
			name:  "valid email with plus",
			email: "user+tag@example.com",
			want:  true,
		},
		{
			name:  "valid email with dots",
			email: "first.last@example.com",
			want:  true,
		},
		{
			name:  "missing @",
			email: "userexample.com",
			want:  false,
		},
		{
			name:  "missing domain",
			email: "user@",
			want:  false,
		},
		{
			name:  "missing local part",
			email: "@example.com",
			want:  false,
		},
		{
			name:  "empty string",
			email: "",
			want:  false,
		},
		{
			name:  "multiple @",
			email: "user@@example.com",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mail.ParseAddress(tt.email)
			got := err == nil
			if got != tt.want {
				t.Errorf("mail.ParseAddress(%q) valid = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

func TestPasswordValidation(t *testing.T) {
	tests := []struct {
		name     string
		password string
		minValid bool
		maxValid bool
	}{
		{
			name:     "8 characters exactly",
			password: "12345678",
			minValid: true,
			maxValid: true,
		},
		{
			name:     "more than 8 characters",
			password: "password123",
			minValid: true,
			maxValid: true,
		},
		{
			name:     "7 characters",
			password: "1234567",
			minValid: false,
			maxValid: true,
		},
		{
			name:     "empty password",
			password: "",
			minValid: false,
			maxValid: true,
		},
		{
			name:     "1 character",
			password: "a",
			minValid: false,
			maxValid: true,
		},
		{
			name:     "72 characters exactly (bcrypt max)",
			password: "aaaaaaaabbbbbbbbccccccccddddddddeeeeeeeeffffffffgggggggghhhhhhhhiiiiiiii",
			minValid: true,
			maxValid: true,
		},
		{
			name:     "73 characters (over bcrypt limit)",
			password: "aaaaaaaabbbbbbbbccccccccddddddddeeeeeeeeffffffffgggggggghhhhhhhhiiiiiiiii",
			minValid: true,
			maxValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minValid := len(tt.password) >= 8
			if minValid != tt.minValid {
				t.Errorf("min length validation for %q = %v, want %v", tt.password, minValid, tt.minValid)
			}

			maxValid := len(tt.password) <= 72
			if maxValid != tt.maxValid {
				t.Errorf("max length validation for %q = %v, want %v", tt.password, maxValid, tt.maxValid)
			}
		})
	}
}
