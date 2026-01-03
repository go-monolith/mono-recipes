package auth

import (
	"testing"
)

func TestPasswordHasher_Hash(t *testing.T) {
	hasher := NewPasswordHasher()

	tests := []struct {
		name     string
		password string
	}{
		{
			name:     "simple password",
			password: "password123",
		},
		{
			name:     "complex password",
			password: "P@ssw0rd!#$%^&*()",
		},
		{
			name:     "long password",
			password: "this-is-a-very-long-password-that-should-still-work-correctly",
		},
		{
			name:     "unicode password",
			password: "密码123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := hasher.Hash(tt.password)
			if err != nil {
				t.Fatalf("Hash() error = %v", err)
			}

			if hash == "" {
				t.Error("Hash() returned empty string")
			}

			// Hash should be different from the original password
			if hash == tt.password {
				t.Error("Hash() returned the original password")
			}

			// Verify the hash works
			if !hasher.Verify(tt.password, hash) {
				t.Error("Verify() returned false for correct password")
			}
		})
	}
}

func TestPasswordHasher_Verify(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "testpassword123"

	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{
			name:     "correct password",
			password: password,
			hash:     hash,
			want:     true,
		},
		{
			name:     "wrong password",
			password: "wrongpassword",
			hash:     hash,
			want:     false,
		},
		{
			name:     "empty password",
			password: "",
			hash:     hash,
			want:     false,
		},
		{
			name:     "similar password",
			password: password + "1",
			hash:     hash,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasher.Verify(tt.password, tt.hash)
			if got != tt.want {
				t.Errorf("Verify() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPasswordHasher_UniqueHashes(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "samepassword"

	hash1, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	hash2, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	// Same password should produce different hashes (due to salt)
	if hash1 == hash2 {
		t.Error("Hash() produced identical hashes for the same password")
	}

	// Both hashes should verify correctly
	if !hasher.Verify(password, hash1) {
		t.Error("Verify() failed for hash1")
	}
	if !hasher.Verify(password, hash2) {
		t.Error("Verify() failed for hash2")
	}
}
