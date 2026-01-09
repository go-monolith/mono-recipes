package user

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// getTestDatabaseURL returns the test database URL.
func getTestDatabaseURL() string {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://demo:demo123@localhost:5432/users_db?sslmode=disable"
	}
	return url
}

// setupTestDB creates a test database connection.
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, getTestDatabaseURL())
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}

	// Ping to verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("Skipping test: database ping failed: %v", err)
	}

	// Clean up test data before each test
	_, err = pool.Exec(ctx, "DELETE FROM users WHERE email LIKE 'test-%@example.com'")
	if err != nil {
		pool.Close()
		t.Fatalf("failed to clean up test data: %v", err)
	}

	return pool
}

func TestRepository_Create(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewRepository(pool)
	ctx := context.Background()

	user, err := repo.Create(ctx, "Test User", "test-create@example.com")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if user.Name != "Test User" {
		t.Errorf("expected name %q, got %q", "Test User", user.Name)
	}
	if user.Email != "test-create@example.com" {
		t.Errorf("expected email %q, got %q", "test-create@example.com", user.Email)
	}
	if !user.ID.Valid {
		t.Error("expected valid UUID, got invalid")
	}
}

func TestRepository_Create_DuplicateEmail(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewRepository(pool)
	ctx := context.Background()

	// Create first user
	_, err := repo.Create(ctx, "First User", "test-duplicate@example.com")
	if err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	// Attempt to create duplicate
	_, err = repo.Create(ctx, "Second User", "test-duplicate@example.com")
	if err != ErrDuplicateEmail {
		t.Errorf("expected ErrDuplicateEmail, got %v", err)
	}
}

func TestRepository_FindByID(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewRepository(pool)
	ctx := context.Background()

	// Create a test user
	created, err := repo.Create(ctx, "FindByID Test", "test-findbyid@example.com")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	t.Run("existing user", func(t *testing.T) {
		id := uuid.UUID(created.ID.Bytes)
		found, err := repo.FindByID(ctx, id)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}

		if found.Name != created.Name {
			t.Errorf("expected name %q, got %q", created.Name, found.Name)
		}
	})

	t.Run("non-existent user", func(t *testing.T) {
		nonExistentID := uuid.MustParse("00000000-0000-0000-0000-000000000000")
		_, err := repo.FindByID(ctx, nonExistentID)
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestRepository_FindAll(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewRepository(pool)
	ctx := context.Background()

	// Create test users
	for i := 0; i < 5; i++ {
		_, err := repo.Create(ctx, fmt.Sprintf("User %d", i), fmt.Sprintf("test-list-%d@example.com", i))
		if err != nil {
			t.Fatalf("failed to create test user: %v", err)
		}
	}

	t.Run("with pagination", func(t *testing.T) {
		users, err := repo.FindAll(ctx, 2, 0)
		if err != nil {
			t.Fatalf("FindAll() error = %v", err)
		}
		if len(users) != 2 {
			t.Errorf("expected 2 users, got %d", len(users))
		}
	})

	t.Run("count", func(t *testing.T) {
		count, err := repo.Count(ctx)
		if err != nil {
			t.Fatalf("Count() error = %v", err)
		}
		if count < 5 {
			t.Errorf("expected at least 5 users, got %d", count)
		}
	})
}

func TestRepository_Update(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewRepository(pool)
	ctx := context.Background()

	// Create a test user
	created, err := repo.Create(ctx, "Original Name", "test-update@example.com")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	t.Run("update name", func(t *testing.T) {
		id := uuid.UUID(created.ID.Bytes)
		newName := "Updated Name"
		updated, err := repo.Update(ctx, id, &newName, nil)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		if updated.Name != newName {
			t.Errorf("expected name %q, got %q", newName, updated.Name)
		}
	})

	t.Run("update email", func(t *testing.T) {
		id := uuid.UUID(created.ID.Bytes)
		newEmail := "test-update-new@example.com"
		updated, err := repo.Update(ctx, id, nil, &newEmail)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		if updated.Email != newEmail {
			t.Errorf("expected email %q, got %q", newEmail, updated.Email)
		}
	})

	t.Run("update non-existent", func(t *testing.T) {
		nonExistentID := uuid.MustParse("00000000-0000-0000-0000-000000000000")
		newName := "Should Not Work"
		_, err := repo.Update(ctx, nonExistentID, &newName, nil)
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestRepository_Delete(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewRepository(pool)
	ctx := context.Background()

	// Create a test user
	created, err := repo.Create(ctx, "To Be Deleted", "test-delete@example.com")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	t.Run("delete existing", func(t *testing.T) {
		id := uuid.UUID(created.ID.Bytes)
		err := repo.Delete(ctx, id)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Verify deletion
		_, err = repo.FindByID(ctx, id)
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound after delete, got %v", err)
		}
	})

	t.Run("delete non-existent (no error)", func(t *testing.T) {
		nonExistentID := uuid.MustParse("00000000-0000-0000-0000-000000000000")
		err := repo.Delete(ctx, nonExistentID)
		// Delete should not return error for non-existent user
		if err != nil {
			t.Errorf("unexpected error for delete non-existent: %v", err)
		}
	})
}
