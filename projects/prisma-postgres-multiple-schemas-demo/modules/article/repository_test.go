package article

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
		url = "postgres://prisma:prisma@localhost:51213/postgres?sslmode=disable"
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
	_, err = pool.Exec(ctx, "DELETE FROM articles WHERE slug LIKE 'test-%'")
	if err != nil {
		pool.Close()
		t.Fatalf("failed to clean up test data: %v", err)
	}

	return pool
}

func TestRepository_Create(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	article, err := repo.Create(ctx, "Test Article", "Test content here", "test-create-article", false)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if article.Title != "Test Article" {
		t.Errorf("expected title %q, got %q", "Test Article", article.Title)
	}
	if article.Slug != "test-create-article" {
		t.Errorf("expected slug %q, got %q", "test-create-article", article.Slug)
	}
	if article.Published {
		t.Error("expected Published to be false")
	}
	if !article.ID.Valid {
		t.Error("expected valid UUID, got invalid")
	}
}

func TestRepository_Create_DuplicateSlug(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	// Create first article
	_, err := repo.Create(ctx, "First Article", "Content", "test-duplicate-slug", false)
	if err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	// Attempt to create duplicate
	_, err = repo.Create(ctx, "Second Article", "Content", "test-duplicate-slug", false)
	if err != ErrDuplicateSlug {
		t.Errorf("expected ErrDuplicateSlug, got %v", err)
	}
}

func TestRepository_FindByID(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	// Create a test article
	created, err := repo.Create(ctx, "FindByID Test", "Content", "test-findbyid", false)
	if err != nil {
		t.Fatalf("failed to create test article: %v", err)
	}

	t.Run("existing article", func(t *testing.T) {
		id := uuid.UUID(created.ID.Bytes)
		found, err := repo.FindByID(ctx, id)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}

		if found.Title != created.Title {
			t.Errorf("expected title %q, got %q", created.Title, found.Title)
		}
	})

	t.Run("non-existent article", func(t *testing.T) {
		nonExistentID := uuid.MustParse("00000000-0000-0000-0000-000000000000")
		_, err := repo.FindByID(ctx, nonExistentID)
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestRepository_FindBySlug(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	// Create a test article
	created, err := repo.Create(ctx, "FindBySlug Test", "Content", "test-findbyslug", false)
	if err != nil {
		t.Fatalf("failed to create test article: %v", err)
	}

	t.Run("existing slug", func(t *testing.T) {
		found, err := repo.FindBySlug(ctx, "test-findbyslug")
		if err != nil {
			t.Fatalf("FindBySlug() error = %v", err)
		}

		if found.Title != created.Title {
			t.Errorf("expected title %q, got %q", created.Title, found.Title)
		}
	})

	t.Run("non-existent slug", func(t *testing.T) {
		_, err := repo.FindBySlug(ctx, "non-existent-slug")
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestRepository_FindAll(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	// Create test articles
	for i := 0; i < 5; i++ {
		_, err := repo.Create(ctx, fmt.Sprintf("Article %d", i), "Content", fmt.Sprintf("test-list-%d", i), false)
		if err != nil {
			t.Fatalf("failed to create test article: %v", err)
		}
	}

	t.Run("with pagination", func(t *testing.T) {
		articles, err := repo.FindAll(ctx, 2, 0)
		if err != nil {
			t.Fatalf("FindAll() error = %v", err)
		}
		if len(articles) != 2 {
			t.Errorf("expected 2 articles, got %d", len(articles))
		}
	})

	t.Run("count", func(t *testing.T) {
		count, err := repo.Count(ctx)
		if err != nil {
			t.Fatalf("Count() error = %v", err)
		}
		if count < 5 {
			t.Errorf("expected at least 5 articles, got %d", count)
		}
	})
}

func TestRepository_FindAllByPublished(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	// Create published and draft articles
	for i := 0; i < 3; i++ {
		_, err := repo.Create(ctx, fmt.Sprintf("Published %d", i), "Content", fmt.Sprintf("test-published-%d", i), true)
		if err != nil {
			t.Fatalf("failed to create published article: %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		_, err := repo.Create(ctx, fmt.Sprintf("Draft %d", i), "Content", fmt.Sprintf("test-draft-%d", i), false)
		if err != nil {
			t.Fatalf("failed to create draft article: %v", err)
		}
	}

	t.Run("find published", func(t *testing.T) {
		articles, err := repo.FindAllByPublished(ctx, true, 10, 0)
		if err != nil {
			t.Fatalf("FindAllByPublished() error = %v", err)
		}
		if len(articles) < 3 {
			t.Errorf("expected at least 3 published articles, got %d", len(articles))
		}
	})

	t.Run("find drafts", func(t *testing.T) {
		articles, err := repo.FindAllByPublished(ctx, false, 10, 0)
		if err != nil {
			t.Fatalf("FindAllByPublished() error = %v", err)
		}
		if len(articles) < 2 {
			t.Errorf("expected at least 2 draft articles, got %d", len(articles))
		}
	})

	t.Run("count by published", func(t *testing.T) {
		count, err := repo.CountByPublished(ctx, true)
		if err != nil {
			t.Fatalf("CountByPublished() error = %v", err)
		}
		if count < 3 {
			t.Errorf("expected at least 3 published articles, got %d", count)
		}
	})
}

func TestRepository_Update(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	// Create a test article
	created, err := repo.Create(ctx, "Original Title", "Original content", "test-update-slug", false)
	if err != nil {
		t.Fatalf("failed to create test article: %v", err)
	}

	t.Run("update title", func(t *testing.T) {
		id := uuid.UUID(created.ID.Bytes)
		newTitle := "Updated Title"
		updated, err := repo.Update(ctx, id, &newTitle, nil, nil, nil)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		if updated.Title != newTitle {
			t.Errorf("expected title %q, got %q", newTitle, updated.Title)
		}
	})

	t.Run("update published status", func(t *testing.T) {
		id := uuid.UUID(created.ID.Bytes)
		published := true
		updated, err := repo.Update(ctx, id, nil, nil, nil, &published)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		if !updated.Published {
			t.Error("expected Published to be true after update")
		}
	})

	t.Run("update non-existent", func(t *testing.T) {
		nonExistentID := uuid.MustParse("00000000-0000-0000-0000-000000000000")
		newTitle := "Should Not Work"
		_, err := repo.Update(ctx, nonExistentID, &newTitle, nil, nil, nil)
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestRepository_Publish(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	// Create a draft article
	created, err := repo.Create(ctx, "Draft Article", "Content", "test-publish-slug", false)
	if err != nil {
		t.Fatalf("failed to create test article: %v", err)
	}

	if created.Published {
		t.Error("expected initial Published to be false")
	}

	t.Run("publish article", func(t *testing.T) {
		id := uuid.UUID(created.ID.Bytes)
		published, err := repo.Publish(ctx, id)
		if err != nil {
			t.Fatalf("Publish() error = %v", err)
		}

		if !published.Published {
			t.Error("expected Published to be true after publish")
		}
	})

	t.Run("publish non-existent", func(t *testing.T) {
		nonExistentID := uuid.MustParse("00000000-0000-0000-0000-000000000000")
		_, err := repo.Publish(ctx, nonExistentID)
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestRepository_Delete(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewPostgresRepository(pool)
	ctx := context.Background()

	// Create a test article
	created, err := repo.Create(ctx, "To Be Deleted", "Content", "test-delete-slug", false)
	if err != nil {
		t.Fatalf("failed to create test article: %v", err)
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

	t.Run("delete non-existent returns error", func(t *testing.T) {
		nonExistentID := uuid.MustParse("00000000-0000-0000-0000-000000000000")
		err := repo.Delete(ctx, nonExistentID)
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}
