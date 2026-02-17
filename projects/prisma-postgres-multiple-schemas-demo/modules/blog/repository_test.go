package blog

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getTestDatabaseURL() string {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://prisma:prisma@localhost:51213/postgres?sslmode=disable"
	}
	return url
}

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, getTestDatabaseURL())
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("Skipping test: database ping failed: %v", err)
	}

	// Clean up test data before each test
	_, err = pool.Exec(ctx, "DELETE FROM blog.comments WHERE author LIKE 'test-%'")
	if err != nil {
		pool.Close()
		t.Fatalf("failed to clean up test comments: %v", err)
	}
	_, err = pool.Exec(ctx, "DELETE FROM blog.posts WHERE slug LIKE 'test-%'")
	if err != nil {
		pool.Close()
		t.Fatalf("failed to clean up test posts: %v", err)
	}

	return pool
}

// === Post Repository Tests ===

func TestPostRepository_Create(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewPostgresPostRepository(pool)
	ctx := context.Background()

	post, err := repo.Create(ctx, "Test Post", "Test content here", "test-create-post", false)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if post.Title != "Test Post" {
		t.Errorf("expected title %q, got %q", "Test Post", post.Title)
	}
	if post.Slug != "test-create-post" {
		t.Errorf("expected slug %q, got %q", "test-create-post", post.Slug)
	}
	if post.Published {
		t.Error("expected Published to be false")
	}
	if !post.ID.Valid {
		t.Error("expected valid UUID, got invalid")
	}
}

func TestPostRepository_Create_DuplicateSlug(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewPostgresPostRepository(pool)
	ctx := context.Background()

	_, err := repo.Create(ctx, "First Post", "Content", "test-duplicate-slug", false)
	if err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	_, err = repo.Create(ctx, "Second Post", "Content", "test-duplicate-slug", false)
	if err != ErrDuplicateSlug {
		t.Errorf("expected ErrDuplicateSlug, got %v", err)
	}
}

func TestPostRepository_FindByID(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewPostgresPostRepository(pool)
	ctx := context.Background()

	created, err := repo.Create(ctx, "FindByID Test", "Content", "test-findbyid-post", false)
	if err != nil {
		t.Fatalf("failed to create test post: %v", err)
	}

	t.Run("existing post", func(t *testing.T) {
		id := uuid.UUID(created.ID.Bytes)
		found, err := repo.FindByID(ctx, id)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}

		if found.Title != created.Title {
			t.Errorf("expected title %q, got %q", created.Title, found.Title)
		}
	})

	t.Run("non-existent post", func(t *testing.T) {
		nonExistentID := uuid.MustParse("00000000-0000-0000-0000-000000000000")
		_, err := repo.FindByID(ctx, nonExistentID)
		if err != ErrPostNotFound {
			t.Errorf("expected ErrPostNotFound, got %v", err)
		}
	})
}

func TestPostRepository_FindBySlug(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewPostgresPostRepository(pool)
	ctx := context.Background()

	created, err := repo.Create(ctx, "FindBySlug Test", "Content", "test-findbyslug-post", false)
	if err != nil {
		t.Fatalf("failed to create test post: %v", err)
	}

	t.Run("existing slug", func(t *testing.T) {
		found, err := repo.FindBySlug(ctx, "test-findbyslug-post")
		if err != nil {
			t.Fatalf("FindBySlug() error = %v", err)
		}

		if found.Title != created.Title {
			t.Errorf("expected title %q, got %q", created.Title, found.Title)
		}
	})

	t.Run("non-existent slug", func(t *testing.T) {
		_, err := repo.FindBySlug(ctx, "non-existent-slug")
		if err != ErrPostNotFound {
			t.Errorf("expected ErrPostNotFound, got %v", err)
		}
	})
}

func TestPostRepository_FindAll(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewPostgresPostRepository(pool)
	ctx := context.Background()

	for i := range 5 {
		_, err := repo.Create(ctx, fmt.Sprintf("Post %d", i), "Content", fmt.Sprintf("test-list-%d", i), false)
		if err != nil {
			t.Fatalf("failed to create test post: %v", err)
		}
	}

	t.Run("with pagination", func(t *testing.T) {
		posts, err := repo.FindAll(ctx, 2, 0)
		if err != nil {
			t.Fatalf("FindAll() error = %v", err)
		}
		if len(posts) != 2 {
			t.Errorf("expected 2 posts, got %d", len(posts))
		}
	})

	t.Run("count", func(t *testing.T) {
		count, err := repo.Count(ctx)
		if err != nil {
			t.Fatalf("Count() error = %v", err)
		}
		if count < 5 {
			t.Errorf("expected at least 5 posts, got %d", count)
		}
	})
}

func TestPostRepository_FindAllByPublished(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewPostgresPostRepository(pool)
	ctx := context.Background()

	for i := range 3 {
		_, err := repo.Create(ctx, fmt.Sprintf("Published %d", i), "Content", fmt.Sprintf("test-published-%d", i), true)
		if err != nil {
			t.Fatalf("failed to create published post: %v", err)
		}
	}
	for i := range 2 {
		_, err := repo.Create(ctx, fmt.Sprintf("Draft %d", i), "Content", fmt.Sprintf("test-draft-%d", i), false)
		if err != nil {
			t.Fatalf("failed to create draft post: %v", err)
		}
	}

	t.Run("find published", func(t *testing.T) {
		posts, err := repo.FindAllByPublished(ctx, true, 10, 0)
		if err != nil {
			t.Fatalf("FindAllByPublished() error = %v", err)
		}
		if len(posts) < 3 {
			t.Errorf("expected at least 3 published posts, got %d", len(posts))
		}
	})

	t.Run("find drafts", func(t *testing.T) {
		posts, err := repo.FindAllByPublished(ctx, false, 10, 0)
		if err != nil {
			t.Fatalf("FindAllByPublished() error = %v", err)
		}
		if len(posts) < 2 {
			t.Errorf("expected at least 2 draft posts, got %d", len(posts))
		}
	})

	t.Run("count by published", func(t *testing.T) {
		count, err := repo.CountByPublished(ctx, true)
		if err != nil {
			t.Fatalf("CountByPublished() error = %v", err)
		}
		if count < 3 {
			t.Errorf("expected at least 3 published posts, got %d", count)
		}
	})
}

func TestPostRepository_Update(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewPostgresPostRepository(pool)
	ctx := context.Background()

	created, err := repo.Create(ctx, "Original Title", "Original content", "test-update-slug", false)
	if err != nil {
		t.Fatalf("failed to create test post: %v", err)
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
		if err != ErrPostNotFound {
			t.Errorf("expected ErrPostNotFound, got %v", err)
		}
	})
}

func TestPostRepository_Publish(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewPostgresPostRepository(pool)
	ctx := context.Background()

	created, err := repo.Create(ctx, "Draft Post", "Content", "test-publish-slug", false)
	if err != nil {
		t.Fatalf("failed to create test post: %v", err)
	}

	if created.Published {
		t.Error("expected initial Published to be false")
	}

	t.Run("publish post", func(t *testing.T) {
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
		if err != ErrPostNotFound {
			t.Errorf("expected ErrPostNotFound, got %v", err)
		}
	})
}

func TestPostRepository_Delete(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewPostgresPostRepository(pool)
	ctx := context.Background()

	created, err := repo.Create(ctx, "To Be Deleted", "Content", "test-delete-slug", false)
	if err != nil {
		t.Fatalf("failed to create test post: %v", err)
	}

	t.Run("delete existing", func(t *testing.T) {
		id := uuid.UUID(created.ID.Bytes)
		err := repo.Delete(ctx, id)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		_, err = repo.FindByID(ctx, id)
		if err != ErrPostNotFound {
			t.Errorf("expected ErrPostNotFound after delete, got %v", err)
		}
	})

	t.Run("delete non-existent returns error", func(t *testing.T) {
		nonExistentID := uuid.MustParse("00000000-0000-0000-0000-000000000000")
		err := repo.Delete(ctx, nonExistentID)
		if err != ErrPostNotFound {
			t.Errorf("expected ErrPostNotFound, got %v", err)
		}
	})
}

// === Comment Repository Tests ===

func TestCommentRepository_Create(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	postRepo := NewPostgresPostRepository(pool)
	commentRepo := NewPostgresCommentRepository(pool)
	ctx := context.Background()

	// Create a post first
	post, err := postRepo.Create(ctx, "Post for Comments", "Content", "test-comment-post", false)
	if err != nil {
		t.Fatalf("failed to create test post: %v", err)
	}

	t.Run("create comment", func(t *testing.T) {
		postID := uuid.UUID(post.ID.Bytes)
		comment, err := commentRepo.Create(ctx, postID, "test-author", "Test comment content")
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if comment.Author != "test-author" {
			t.Errorf("expected author %q, got %q", "test-author", comment.Author)
		}
		if comment.Content != "Test comment content" {
			t.Errorf("expected content %q, got %q", "Test comment content", comment.Content)
		}
	})

	t.Run("create comment for non-existent post", func(t *testing.T) {
		nonExistentPostID := uuid.MustParse("00000000-0000-0000-0000-000000000000")
		_, err := commentRepo.Create(ctx, nonExistentPostID, "test-author", "Should fail")
		if err != ErrPostDoesNotExist {
			t.Errorf("expected ErrPostDoesNotExist, got %v", err)
		}
	})
}

func TestCommentRepository_FindByID(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	postRepo := NewPostgresPostRepository(pool)
	commentRepo := NewPostgresCommentRepository(pool)
	ctx := context.Background()

	post, err := postRepo.Create(ctx, "Post for FindByID", "Content", "test-findbyid-comment-post", false)
	if err != nil {
		t.Fatalf("failed to create test post: %v", err)
	}

	postID := uuid.UUID(post.ID.Bytes)
	created, err := commentRepo.Create(ctx, postID, "test-author", "Test content")
	if err != nil {
		t.Fatalf("failed to create test comment: %v", err)
	}

	t.Run("existing comment", func(t *testing.T) {
		id := uuid.UUID(created.ID.Bytes)
		found, err := commentRepo.FindByID(ctx, id)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}

		if found.Author != created.Author {
			t.Errorf("expected author %q, got %q", created.Author, found.Author)
		}
	})

	t.Run("non-existent comment", func(t *testing.T) {
		nonExistentID := uuid.MustParse("00000000-0000-0000-0000-000000000000")
		_, err := commentRepo.FindByID(ctx, nonExistentID)
		if err != ErrCommentNotFound {
			t.Errorf("expected ErrCommentNotFound, got %v", err)
		}
	})
}

func TestCommentRepository_FindAllByPostID(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	postRepo := NewPostgresPostRepository(pool)
	commentRepo := NewPostgresCommentRepository(pool)
	ctx := context.Background()

	post, err := postRepo.Create(ctx, "Post for Listing Comments", "Content", "test-list-comments-post", false)
	if err != nil {
		t.Fatalf("failed to create test post: %v", err)
	}

	postID := uuid.UUID(post.ID.Bytes)
	for i := range 5 {
		_, err := commentRepo.Create(ctx, postID, fmt.Sprintf("test-author-%d", i), fmt.Sprintf("Comment %d", i))
		if err != nil {
			t.Fatalf("failed to create test comment: %v", err)
		}
	}

	t.Run("with pagination", func(t *testing.T) {
		comments, err := commentRepo.FindAllByPostID(ctx, postID, 2, 0)
		if err != nil {
			t.Fatalf("FindAllByPostID() error = %v", err)
		}
		if len(comments) != 2 {
			t.Errorf("expected 2 comments, got %d", len(comments))
		}
	})

	t.Run("count", func(t *testing.T) {
		count, err := commentRepo.CountByPostID(ctx, postID)
		if err != nil {
			t.Fatalf("CountByPostID() error = %v", err)
		}
		if count != 5 {
			t.Errorf("expected 5 comments, got %d", count)
		}
	})
}

func TestCommentRepository_Update(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	postRepo := NewPostgresPostRepository(pool)
	commentRepo := NewPostgresCommentRepository(pool)
	ctx := context.Background()

	post, err := postRepo.Create(ctx, "Post for Update Comment", "Content", "test-update-comment-post", false)
	if err != nil {
		t.Fatalf("failed to create test post: %v", err)
	}

	postID := uuid.UUID(post.ID.Bytes)
	created, err := commentRepo.Create(ctx, postID, "test-original-author", "Original content")
	if err != nil {
		t.Fatalf("failed to create test comment: %v", err)
	}

	t.Run("update author", func(t *testing.T) {
		id := uuid.UUID(created.ID.Bytes)
		newAuthor := "test-updated-author"
		updated, err := commentRepo.Update(ctx, id, &newAuthor, nil)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		if updated.Author != newAuthor {
			t.Errorf("expected author %q, got %q", newAuthor, updated.Author)
		}
	})

	t.Run("update content", func(t *testing.T) {
		id := uuid.UUID(created.ID.Bytes)
		newContent := "Updated content"
		updated, err := commentRepo.Update(ctx, id, nil, &newContent)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		if updated.Content != newContent {
			t.Errorf("expected content %q, got %q", newContent, updated.Content)
		}
	})

	t.Run("update non-existent", func(t *testing.T) {
		nonExistentID := uuid.MustParse("00000000-0000-0000-0000-000000000000")
		newAuthor := "Should Not Work"
		_, err := commentRepo.Update(ctx, nonExistentID, &newAuthor, nil)
		if err != ErrCommentNotFound {
			t.Errorf("expected ErrCommentNotFound, got %v", err)
		}
	})
}

func TestCommentRepository_Delete(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	postRepo := NewPostgresPostRepository(pool)
	commentRepo := NewPostgresCommentRepository(pool)
	ctx := context.Background()

	post, err := postRepo.Create(ctx, "Post for Delete Comment", "Content", "test-delete-comment-post", false)
	if err != nil {
		t.Fatalf("failed to create test post: %v", err)
	}

	postID := uuid.UUID(post.ID.Bytes)
	created, err := commentRepo.Create(ctx, postID, "test-delete-author", "To be deleted")
	if err != nil {
		t.Fatalf("failed to create test comment: %v", err)
	}

	t.Run("delete existing", func(t *testing.T) {
		id := uuid.UUID(created.ID.Bytes)
		err := commentRepo.Delete(ctx, id)
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		_, err = commentRepo.FindByID(ctx, id)
		if err != ErrCommentNotFound {
			t.Errorf("expected ErrCommentNotFound after delete, got %v", err)
		}
	})

	t.Run("delete non-existent returns error", func(t *testing.T) {
		nonExistentID := uuid.MustParse("00000000-0000-0000-0000-000000000000")
		err := commentRepo.Delete(ctx, nonExistentID)
		if err != ErrCommentNotFound {
			t.Errorf("expected ErrCommentNotFound, got %v", err)
		}
	})
}
