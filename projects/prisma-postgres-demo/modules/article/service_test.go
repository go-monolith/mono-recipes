package article

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/example/prisma-postgres-demo/modules/article/db/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// mockRepository is a test double implementing ArticleRepository.
type mockRepository struct {
	articles                 map[uuid.UUID]*generated.Article
	slugIndex                map[string]uuid.UUID
	createErr                error
	findByIDErr              error
	findBySlugErr            error
	findAllErr               error
	findAllByPublishedErr    error
	countErr                 error
	countByPublishedErr      error
	updateErr                error
	publishErr               error
	deleteErr                error
	slugExists               bool
	returnedCount            int64
	returnedCountByPublished int64
}

// Compile-time interface check.
var _ ArticleRepository = (*mockRepository)(nil)

func newMockRepository() *mockRepository {
	return &mockRepository{
		articles:  make(map[uuid.UUID]*generated.Article),
		slugIndex: make(map[string]uuid.UUID),
	}
}

func (m *mockRepository) Create(_ context.Context, title, content, slug string, published bool) (*generated.Article, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.slugExists {
		return nil, ErrDuplicateSlug
	}
	if _, exists := m.slugIndex[slug]; exists {
		return nil, ErrDuplicateSlug
	}

	id := uuid.New()
	now := time.Now()
	article := &generated.Article{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		Title:     title,
		Content:   content,
		Slug:      slug,
		Published: published,
		CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
	}
	m.articles[id] = article
	m.slugIndex[slug] = id
	return article, nil
}

func (m *mockRepository) FindByID(_ context.Context, id uuid.UUID) (*generated.Article, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	article, exists := m.articles[id]
	if !exists {
		return nil, ErrNotFound
	}
	return article, nil
}

func (m *mockRepository) FindBySlug(_ context.Context, slug string) (*generated.Article, error) {
	if m.findBySlugErr != nil {
		return nil, m.findBySlugErr
	}
	id, exists := m.slugIndex[slug]
	if !exists {
		return nil, ErrNotFound
	}
	return m.articles[id], nil
}

func (m *mockRepository) FindAll(_ context.Context, limit, offset int32) ([]generated.Article, error) {
	if m.findAllErr != nil {
		return nil, m.findAllErr
	}

	articles := make([]generated.Article, 0, len(m.articles))
	for _, a := range m.articles {
		articles = append(articles, *a)
	}

	// Apply pagination
	start := int(offset)
	if start >= len(articles) {
		return []generated.Article{}, nil
	}

	end := start + int(limit)
	if end > len(articles) {
		end = len(articles)
	}

	return articles[start:end], nil
}

func (m *mockRepository) FindAllByPublished(_ context.Context, published bool, limit, offset int32) ([]generated.Article, error) {
	if m.findAllByPublishedErr != nil {
		return nil, m.findAllByPublishedErr
	}

	articles := make([]generated.Article, 0)
	for _, a := range m.articles {
		if a.Published == published {
			articles = append(articles, *a)
		}
	}

	// Apply pagination
	start := int(offset)
	if start >= len(articles) {
		return []generated.Article{}, nil
	}

	end := start + int(limit)
	if end > len(articles) {
		end = len(articles)
	}

	return articles[start:end], nil
}

func (m *mockRepository) Count(_ context.Context) (int64, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	if m.returnedCount > 0 {
		return m.returnedCount, nil
	}
	return int64(len(m.articles)), nil
}

func (m *mockRepository) CountByPublished(_ context.Context, published bool) (int64, error) {
	if m.countByPublishedErr != nil {
		return 0, m.countByPublishedErr
	}
	if m.returnedCountByPublished > 0 {
		return m.returnedCountByPublished, nil
	}
	count := int64(0)
	for _, a := range m.articles {
		if a.Published == published {
			count++
		}
	}
	return count, nil
}

func (m *mockRepository) Update(_ context.Context, id uuid.UUID, title, content, slug *string, published *bool) (*generated.Article, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	if m.slugExists && slug != nil {
		return nil, ErrDuplicateSlug
	}

	article, exists := m.articles[id]
	if !exists {
		return nil, ErrNotFound
	}

	// Check if new slug conflicts with existing
	if slug != nil && *slug != article.Slug {
		if _, exists := m.slugIndex[*slug]; exists {
			return nil, ErrDuplicateSlug
		}
		delete(m.slugIndex, article.Slug)
		m.slugIndex[*slug] = id
		article.Slug = *slug
	}

	if title != nil {
		article.Title = *title
	}
	if content != nil {
		article.Content = *content
	}
	if published != nil {
		article.Published = *published
	}
	article.UpdatedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}

	return article, nil
}

func (m *mockRepository) Publish(_ context.Context, id uuid.UUID) (*generated.Article, error) {
	if m.publishErr != nil {
		return nil, m.publishErr
	}

	article, exists := m.articles[id]
	if !exists {
		return nil, ErrNotFound
	}

	article.Published = true
	article.UpdatedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}

	return article, nil
}

func (m *mockRepository) Delete(_ context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	article, exists := m.articles[id]
	if exists {
		delete(m.slugIndex, article.Slug)
	}
	delete(m.articles, id)
	return nil
}

func TestArticleService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		resp, err := svc.Create(context.Background(), CreateArticleRequest{
			Title:     "Test Article",
			Content:   "Test content here",
			Slug:      "test-article",
			Published: false,
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if resp.Title != "Test Article" {
			t.Errorf("expected title %q, got %q", "Test Article", resp.Title)
		}
		if resp.Slug != "test-article" {
			t.Errorf("expected slug %q, got %q", "test-article", resp.Slug)
		}
		if resp.Published {
			t.Error("expected Published to be false")
		}
		if resp.ID == "" {
			t.Error("expected non-empty ID")
		}
	})

	t.Run("missing title", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		_, err := svc.Create(context.Background(), CreateArticleRequest{
			Content: "Test content",
			Slug:    "test-slug",
		})
		if err != ErrTitleRequired {
			t.Errorf("expected ErrTitleRequired, got %v", err)
		}
	})

	t.Run("missing content", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		_, err := svc.Create(context.Background(), CreateArticleRequest{
			Title: "Test Article",
			Slug:  "test-slug",
		})
		if err != ErrContentRequired {
			t.Errorf("expected ErrContentRequired, got %v", err)
		}
	})

	t.Run("missing slug", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		_, err := svc.Create(context.Background(), CreateArticleRequest{
			Title:   "Test Article",
			Content: "Test content",
		})
		if err != ErrSlugRequired {
			t.Errorf("expected ErrSlugRequired, got %v", err)
		}
	})

	t.Run("duplicate slug", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		// Create first article
		_, err := svc.Create(context.Background(), CreateArticleRequest{
			Title:   "First Article",
			Content: "Content",
			Slug:    "same-slug",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Try to create another with same slug
		_, err = svc.Create(context.Background(), CreateArticleRequest{
			Title:   "Second Article",
			Content: "Content",
			Slug:    "same-slug",
		})
		if err != ErrDuplicateSlug {
			t.Errorf("expected ErrDuplicateSlug, got %v", err)
		}
	})
}

func TestArticleService_Get(t *testing.T) {
	t.Run("success by ID", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		created, err := svc.Create(context.Background(), CreateArticleRequest{
			Title:   "Test Article",
			Content: "Test content",
			Slug:    "test-article",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		resp, err := svc.Get(context.Background(), GetArticleRequest{ID: created.ID})
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if resp.Title != created.Title {
			t.Errorf("expected title %q, got %q", created.Title, resp.Title)
		}
	})

	t.Run("success by slug", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		created, err := svc.Create(context.Background(), CreateArticleRequest{
			Title:   "Test Article",
			Content: "Test content",
			Slug:    "test-article",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		resp, err := svc.Get(context.Background(), GetArticleRequest{Slug: "test-article"})
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if resp.Title != created.Title {
			t.Errorf("expected title %q, got %q", created.Title, resp.Title)
		}
	})

	t.Run("missing ID and slug", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		_, err := svc.Get(context.Background(), GetArticleRequest{})
		if err != ErrIDOrSlugRequired {
			t.Errorf("expected ErrIDOrSlugRequired, got %v", err)
		}
	})

	t.Run("invalid UUID", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		_, err := svc.Get(context.Background(), GetArticleRequest{ID: "not-a-uuid"})
		if err != ErrIDInvalid {
			t.Errorf("expected ErrIDInvalid, got %v", err)
		}
	})

	t.Run("not found by ID", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		_, err := svc.Get(context.Background(), GetArticleRequest{
			ID: uuid.New().String(),
		})
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("not found by slug", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		_, err := svc.Get(context.Background(), GetArticleRequest{
			Slug: "non-existent-slug",
		})
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestArticleService_List(t *testing.T) {
	t.Run("success with pagination", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		// Create multiple articles
		for i := range 5 {
			_, err := svc.Create(context.Background(), CreateArticleRequest{
				Title:   "Test Article",
				Content: "Content",
				Slug:    fmt.Sprintf("test-article-%d", i),
			})
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
		}

		resp, err := svc.List(context.Background(), ListArticlesRequest{
			Limit:  2,
			Offset: 0,
		})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(resp.Articles) != 2 {
			t.Errorf("expected 2 articles, got %d", len(resp.Articles))
		}
		if resp.Total != 5 {
			t.Errorf("expected total 5, got %d", resp.Total)
		}
		if resp.Limit != 2 {
			t.Errorf("expected limit 2, got %d", resp.Limit)
		}
	})

	t.Run("filter by published", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		// Create published and draft articles
		for i := range 3 {
			_, err := svc.Create(context.Background(), CreateArticleRequest{
				Title:     "Published Article",
				Content:   "Content",
				Slug:      fmt.Sprintf("published-%d", i),
				Published: true,
			})
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
		}
		for i := range 2 {
			_, err := svc.Create(context.Background(), CreateArticleRequest{
				Title:     "Draft Article",
				Content:   "Content",
				Slug:      fmt.Sprintf("draft-%d", i),
				Published: false,
			})
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
		}

		published := true
		resp, err := svc.List(context.Background(), ListArticlesRequest{
			Published: &published,
		})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if resp.Total != 3 {
			t.Errorf("expected 3 published articles, got %d", resp.Total)
		}

		draft := false
		resp, err = svc.List(context.Background(), ListArticlesRequest{
			Published: &draft,
		})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if resp.Total != 2 {
			t.Errorf("expected 2 draft articles, got %d", resp.Total)
		}
	})

	t.Run("default pagination", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		resp, err := svc.List(context.Background(), ListArticlesRequest{})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if resp.Limit != 10 {
			t.Errorf("expected default limit 10, got %d", resp.Limit)
		}
		if resp.Offset != 0 {
			t.Errorf("expected default offset 0, got %d", resp.Offset)
		}
	})

	t.Run("clamped limit", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		resp, err := svc.List(context.Background(), ListArticlesRequest{
			Limit: 200, // Exceeds max
		})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if resp.Limit != 100 {
			t.Errorf("expected clamped limit 100, got %d", resp.Limit)
		}
	})
}

func TestArticleService_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		created, err := svc.Create(context.Background(), CreateArticleRequest{
			Title:   "Original Title",
			Content: "Original content",
			Slug:    "original-slug",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		newTitle := "Updated Title"
		resp, err := svc.Update(context.Background(), UpdateArticleRequest{
			ID:    created.ID,
			Title: &newTitle,
		})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		if resp.Title != newTitle {
			t.Errorf("expected title %q, got %q", newTitle, resp.Title)
		}
	})

	t.Run("missing ID", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		newTitle := "Updated Title"
		_, err := svc.Update(context.Background(), UpdateArticleRequest{
			Title: &newTitle,
		})
		if err != ErrIDRequired {
			t.Errorf("expected ErrIDRequired, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		newTitle := "Updated Title"
		_, err := svc.Update(context.Background(), UpdateArticleRequest{
			ID:    uuid.New().String(),
			Title: &newTitle,
		})
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("empty title rejected", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		created, err := svc.Create(context.Background(), CreateArticleRequest{
			Title:   "Original Title",
			Content: "Content",
			Slug:    "test-slug",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		emptyTitle := ""
		_, err = svc.Update(context.Background(), UpdateArticleRequest{
			ID:    created.ID,
			Title: &emptyTitle,
		})
		if !errors.Is(err, ErrTitleRequired) {
			t.Errorf("expected ErrTitleRequired, got %v", err)
		}
	})

	t.Run("empty content rejected", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		created, err := svc.Create(context.Background(), CreateArticleRequest{
			Title:   "Title",
			Content: "Original content",
			Slug:    "test-slug",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		emptyContent := ""
		_, err = svc.Update(context.Background(), UpdateArticleRequest{
			ID:      created.ID,
			Content: &emptyContent,
		})
		if !errors.Is(err, ErrContentRequired) {
			t.Errorf("expected ErrContentRequired, got %v", err)
		}
	})

	t.Run("empty slug rejected", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		created, err := svc.Create(context.Background(), CreateArticleRequest{
			Title:   "Title",
			Content: "Content",
			Slug:    "original-slug",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		emptySlug := ""
		_, err = svc.Update(context.Background(), UpdateArticleRequest{
			ID:   created.ID,
			Slug: &emptySlug,
		})
		if !errors.Is(err, ErrSlugRequired) {
			t.Errorf("expected ErrSlugRequired, got %v", err)
		}
	})
}

func TestArticleService_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		created, err := svc.Create(context.Background(), CreateArticleRequest{
			Title:   "Test Article",
			Content: "Content",
			Slug:    "test-slug",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		resp, err := svc.Delete(context.Background(), DeleteArticleRequest{
			ID: created.ID,
		})
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		if !resp.Deleted {
			t.Error("expected Deleted to be true")
		}
		if resp.ID != created.ID {
			t.Errorf("expected ID %q, got %q", created.ID, resp.ID)
		}

		// Verify deletion
		_, err = svc.Get(context.Background(), GetArticleRequest{ID: created.ID})
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound after delete, got %v", err)
		}
	})

	t.Run("missing ID", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		_, err := svc.Delete(context.Background(), DeleteArticleRequest{})
		if err != ErrIDRequired {
			t.Errorf("expected ErrIDRequired, got %v", err)
		}
	})

	t.Run("invalid UUID", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		_, err := svc.Delete(context.Background(), DeleteArticleRequest{ID: "not-a-uuid"})
		if err != ErrIDInvalid {
			t.Errorf("expected ErrIDInvalid, got %v", err)
		}
	})
}

func TestArticleService_Publish(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		created, err := svc.Create(context.Background(), CreateArticleRequest{
			Title:     "Draft Article",
			Content:   "Content",
			Slug:      "draft-slug",
			Published: false,
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if created.Published {
			t.Error("expected initial Published to be false")
		}

		resp, err := svc.Publish(context.Background(), PublishArticleRequest{
			ID: created.ID,
		})
		if err != nil {
			t.Fatalf("Publish() error = %v", err)
		}

		if !resp.Published {
			t.Error("expected Published to be true after publish")
		}
	})

	t.Run("missing ID", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		_, err := svc.Publish(context.Background(), PublishArticleRequest{})
		if err != ErrIDRequired {
			t.Errorf("expected ErrIDRequired, got %v", err)
		}
	})

	t.Run("invalid UUID", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		_, err := svc.Publish(context.Background(), PublishArticleRequest{ID: "not-a-uuid"})
		if err != ErrIDInvalid {
			t.Errorf("expected ErrIDInvalid, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		_, err := svc.Publish(context.Background(), PublishArticleRequest{
			ID: uuid.New().String(),
		})
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestClampLimit(t *testing.T) {
	tests := []struct {
		name     string
		input    int32
		expected int32
	}{
		{"zero uses default", 0, 10},
		{"negative uses default", -5, 10},
		{"within range", 50, 50},
		{"at max", 100, 100},
		{"over max is clamped", 150, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clampLimit(tt.input)
			if result != tt.expected {
				t.Errorf("clampLimit(%d) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestClampOffset(t *testing.T) {
	tests := []struct {
		name     string
		input    int32
		expected int32
	}{
		{"zero", 0, 0},
		{"negative is clamped", -5, 0},
		{"positive value", 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clampOffset(tt.input)
			if result != tt.expected {
				t.Errorf("clampOffset(%d) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestArticleService_ErrorPropagation(t *testing.T) {
	t.Run("create propagates repository error", func(t *testing.T) {
		repo := newMockRepository()
		repo.createErr = errors.New("db connection failed")
		svc := NewArticleService(repo)

		_, err := svc.Create(context.Background(), CreateArticleRequest{
			Title:   "Test Article",
			Content: "Content",
			Slug:    "test-slug",
		})
		if err == nil || err.Error() != "db connection failed" {
			t.Errorf("expected repository error, got %v", err)
		}
	})

	t.Run("get propagates repository error", func(t *testing.T) {
		repo := newMockRepository()
		repo.findByIDErr = errors.New("db query failed")
		svc := NewArticleService(repo)

		id := uuid.New()
		repo.articles[id] = &generated.Article{
			ID: pgtype.UUID{Bytes: id, Valid: true},
		}

		_, err := svc.Get(context.Background(), GetArticleRequest{ID: id.String()})
		if err == nil || err.Error() != "db query failed" {
			t.Errorf("expected repository error, got %v", err)
		}
	})

	t.Run("list propagates findAll error", func(t *testing.T) {
		repo := newMockRepository()
		repo.findAllErr = errors.New("db query failed")
		svc := NewArticleService(repo)

		_, err := svc.List(context.Background(), ListArticlesRequest{Limit: 10})
		if err == nil || err.Error() != "db query failed" {
			t.Errorf("expected repository error, got %v", err)
		}
	})

	t.Run("list propagates count error", func(t *testing.T) {
		repo := newMockRepository()
		repo.countErr = errors.New("count query failed")
		svc := NewArticleService(repo)

		_, err := svc.List(context.Background(), ListArticlesRequest{Limit: 10})
		if err == nil || err.Error() != "count query failed" {
			t.Errorf("expected repository error, got %v", err)
		}
	})

	t.Run("publish propagates repository error", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewArticleService(repo)

		created, err := svc.Create(context.Background(), CreateArticleRequest{
			Title:   "Test Article",
			Content: "Content",
			Slug:    "test-slug",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		repo.publishErr = errors.New("db publish failed")

		_, err = svc.Publish(context.Background(), PublishArticleRequest{ID: created.ID})
		if err == nil || err.Error() != "db publish failed" {
			t.Errorf("expected repository error, got %v", err)
		}
	})
}
