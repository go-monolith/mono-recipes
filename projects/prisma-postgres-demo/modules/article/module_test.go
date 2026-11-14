package article_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/go-monolith/mono/monotest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/example/prisma-postgres-demo/modules/article"
	"github.com/example/prisma-postgres-demo/modules/article/contract"
)

// TestArticleModuleConformance is the one-per-module boundary check. The
// service layer is replaced by an in-memory fake (article.NewModuleWithService),
// so the test needs neither PostgreSQL nor NATS: it checks the module's name,
// that it registers its services through valid subjects, that every contract
// round trips, and that Start/Stop succeed.
func TestArticleModuleConformance(t *testing.T) {
	monotest.CheckModule(t, article.NewModuleWithService(newFakeService()),
		monotest.WithExercise(func(ctx context.Context, h *monotest.Harness) error {
			return exerciseCRUD(ctx, h, "test-conformance")
		}))
}

// TestArticleModuleConformancePostgres runs the same check against the real
// PostgreSQL-backed module. It is skipped unless a database is reachable at
// TEST_DATABASE_URL (default: the local Prisma Postgres dev server).
func TestArticleModuleConformancePostgres(t *testing.T) {
	t.Setenv("DATABASE_URL", requireDatabase(t))
	slug := "test-" + uuid.NewString()
	monotest.CheckModule(t, article.NewModule(),
		monotest.WithExercise(func(ctx context.Context, h *monotest.Harness) error {
			return exerciseCRUD(ctx, h, slug)
		}))
}

// exerciseCRUD calls every article contract once.
func exerciseCRUD(ctx context.Context, h *monotest.Harness, slug string) error {
	c := h.Container()

	created, err := contract.CreateArticle.Call(ctx, c, contract.CreateArticleRequest{
		Title: "Hello", Content: "World", Slug: slug,
	})
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}

	got, err := contract.GetArticle.Call(ctx, c, contract.GetArticleRequest{Slug: slug})
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}
	if got.ID != created.ID {
		return fmt.Errorf("get by slug: id = %q, want %q", got.ID, created.ID)
	}

	drafts := false
	list, err := contract.ListArticles.Call(ctx, c, contract.ListArticlesRequest{Limit: 100, Published: &drafts})
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}
	if list.Total < 1 {
		return fmt.Errorf("list: total = %d, want >= 1", list.Total)
	}

	title := "Hello again"
	updated, err := contract.UpdateArticle.Call(ctx, c, contract.UpdateArticleRequest{ID: created.ID, Title: &title})
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}
	if updated.Title != title {
		return fmt.Errorf("update: title = %q, want %q", updated.Title, title)
	}

	published, err := contract.PublishArticle.Call(ctx, c, contract.PublishArticleRequest{ID: created.ID})
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	if !published.Published {
		return errors.New("publish: published = false")
	}

	deleted, err := contract.DeleteArticle.Call(ctx, c, contract.DeleteArticleRequest{ID: created.ID})
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if !deleted.Deleted {
		return errors.New("delete: deleted = false")
	}

	if _, err := contract.GetArticle.Call(ctx, c, contract.GetArticleRequest{ID: created.ID}); err == nil {
		return errors.New("get after delete: expected an error")
	}
	return nil
}

// requireDatabase returns the test database URL, or skips the test when no
// PostgreSQL server answers there.
func requireDatabase(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://prisma:prisma@localhost:51213/postgres?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("Skipping test: database ping failed: %v", err)
	}
	return url
}

// fakeService is an in-memory article.ArticleService. It is safe for
// concurrent use, like the real service, because request-reply handlers run
// on a worker pool.
type fakeService struct {
	mu       sync.Mutex
	articles map[string]contract.ArticleResponse
}

var _ article.ArticleService = (*fakeService)(nil)

func newFakeService() *fakeService {
	return &fakeService{articles: make(map[string]contract.ArticleResponse)}
}

func (s *fakeService) Create(_ context.Context, req contract.CreateArticleRequest) (contract.ArticleResponse, error) {
	if req.Title == "" || req.Content == "" || req.Slug == "" {
		return contract.ArticleResponse{}, errors.New("title, content and slug are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	a := contract.ArticleResponse{
		ID: uuid.NewString(), Title: req.Title, Content: req.Content, Slug: req.Slug,
		Published: req.Published, CreatedAt: now, UpdatedAt: now,
	}
	s.articles[a.ID] = a
	return a, nil
}

func (s *fakeService) Get(_ context.Context, req contract.GetArticleRequest) (contract.ArticleResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, a := range s.articles {
		if (req.ID != "" && a.ID == req.ID) || (req.ID == "" && req.Slug != "" && a.Slug == req.Slug) {
			return a, nil
		}
	}
	return contract.ArticleResponse{}, article.ErrNotFound
}

func (s *fakeService) List(_ context.Context, req contract.ListArticlesRequest) (contract.ListArticlesResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	resp := contract.ListArticlesResponse{Articles: []contract.ArticleResponse{}, Limit: req.Limit, Offset: req.Offset}
	for _, a := range s.articles {
		if req.Published == nil || a.Published == *req.Published {
			resp.Articles = append(resp.Articles, a)
		}
	}
	resp.Total = int64(len(resp.Articles))
	return resp, nil
}

func (s *fakeService) Update(_ context.Context, req contract.UpdateArticleRequest) (contract.ArticleResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.articles[req.ID]
	if !ok {
		return contract.ArticleResponse{}, article.ErrNotFound
	}
	if req.Title != nil {
		a.Title = *req.Title
	}
	if req.Content != nil {
		a.Content = *req.Content
	}
	if req.Slug != nil {
		a.Slug = *req.Slug
	}
	if req.Published != nil {
		a.Published = *req.Published
	}
	a.UpdatedAt = time.Now()
	s.articles[a.ID] = a
	return a, nil
}

func (s *fakeService) Delete(_ context.Context, req contract.DeleteArticleRequest) (contract.DeleteArticleResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.articles[req.ID]; !ok {
		return contract.DeleteArticleResponse{ID: req.ID}, article.ErrNotFound
	}
	delete(s.articles, req.ID)
	return contract.DeleteArticleResponse{Deleted: true, ID: req.ID}, nil
}

func (s *fakeService) Publish(ctx context.Context, req contract.PublishArticleRequest) (contract.ArticleResponse, error) {
	published := true
	return s.Update(ctx, contract.UpdateArticleRequest{ID: req.ID, Published: &published})
}
