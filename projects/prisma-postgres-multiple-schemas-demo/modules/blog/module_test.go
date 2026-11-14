package blog_test

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

	"github.com/example/prisma-postgres-multiple-schemas-demo/modules/blog"
	"github.com/example/prisma-postgres-multiple-schemas-demo/modules/blog/contract"
)

// TestBlogModuleConformance is the one-per-module boundary check. The post and
// comment services are replaced by in-memory fakes (blog.NewModuleWithServices),
// so the test needs neither PostgreSQL nor NATS: it checks the module's name,
// that it registers its services through valid subjects, that every contract
// round trips, and that Start/Stop succeed.
func TestBlogModuleConformance(t *testing.T) {
	store := newFakeStore()
	monotest.CheckModule(t, blog.NewModuleWithServices(fakePosts{store}, fakeComments{store}),
		monotest.WithExercise(func(ctx context.Context, h *monotest.Harness) error {
			return exerciseBlog(ctx, h, "test-conformance")
		}))
}

// TestBlogModuleConformancePostgres runs the same check against the real
// PostgreSQL-backed module. It is skipped unless a database is reachable at
// TEST_DATABASE_URL (default: the local Prisma Postgres dev server).
func TestBlogModuleConformancePostgres(t *testing.T) {
	t.Setenv("DATABASE_URL", requireDatabase(t))
	slug := "test-" + uuid.NewString()
	monotest.CheckModule(t, blog.NewModule(),
		monotest.WithExercise(func(ctx context.Context, h *monotest.Harness) error {
			return exerciseBlog(ctx, h, slug)
		}))
}

// exerciseBlog calls every post and comment contract once.
func exerciseBlog(ctx context.Context, h *monotest.Harness, slug string) error {
	c := h.Container()

	// Posts
	post, err := contract.CreatePost.Call(ctx, c, contract.CreatePostRequest{Title: "Hello", Content: "World", Slug: slug})
	if err != nil {
		return fmt.Errorf("post-create: %w", err)
	}
	if got, err := contract.GetPost.Call(ctx, c, contract.GetPostRequest{Slug: slug}); err != nil {
		return fmt.Errorf("post-get: %w", err)
	} else if got.ID != post.ID {
		return fmt.Errorf("post-get by slug: id = %q, want %q", got.ID, post.ID)
	}
	if list, err := contract.ListPosts.Call(ctx, c, contract.ListPostsRequest{Limit: 100}); err != nil {
		return fmt.Errorf("post-list: %w", err)
	} else if list.Total < 1 {
		return fmt.Errorf("post-list: total = %d, want >= 1", list.Total)
	}
	title := "Hello again"
	if updated, err := contract.UpdatePost.Call(ctx, c, contract.UpdatePostRequest{ID: post.ID, Title: &title}); err != nil {
		return fmt.Errorf("post-update: %w", err)
	} else if updated.Title != title {
		return fmt.Errorf("post-update: title = %q, want %q", updated.Title, title)
	}
	if published, err := contract.PublishPost.Call(ctx, c, contract.PublishPostRequest{ID: post.ID}); err != nil {
		return fmt.Errorf("post-publish: %w", err)
	} else if !published.Published {
		return errors.New("post-publish: published = false")
	}

	// Comments
	comment, err := contract.CreateComment.Call(ctx, c, contract.CreateCommentRequest{PostID: post.ID, Author: "Ann", Content: "Nice"})
	if err != nil {
		return fmt.Errorf("comment-create: %w", err)
	}
	if _, err := contract.GetComment.Call(ctx, c, contract.GetCommentRequest{ID: comment.ID}); err != nil {
		return fmt.Errorf("comment-get: %w", err)
	}
	if list, err := contract.ListComments.Call(ctx, c, contract.ListCommentsRequest{PostID: post.ID}); err != nil {
		return fmt.Errorf("comment-list: %w", err)
	} else if list.Total != 1 {
		return fmt.Errorf("comment-list: total = %d, want 1", list.Total)
	}
	text := "Very nice"
	if updated, err := contract.UpdateComment.Call(ctx, c, contract.UpdateCommentRequest{ID: comment.ID, Content: &text}); err != nil {
		return fmt.Errorf("comment-update: %w", err)
	} else if updated.Content != text {
		return fmt.Errorf("comment-update: content = %q, want %q", updated.Content, text)
	}
	if _, err := contract.DeleteComment.Call(ctx, c, contract.DeleteCommentRequest{ID: comment.ID}); err != nil {
		return fmt.Errorf("comment-delete: %w", err)
	}

	if deleted, err := contract.DeletePost.Call(ctx, c, contract.DeletePostRequest{ID: post.ID}); err != nil {
		return fmt.Errorf("post-delete: %w", err)
	} else if !deleted.Deleted {
		return errors.New("post-delete: deleted = false")
	}
	if _, err := contract.GetPost.Call(ctx, c, contract.GetPostRequest{ID: post.ID}); err == nil {
		return errors.New("post-get after delete: expected an error")
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

// fakeStore is the in-memory state shared by the fake post and comment
// services. It is safe for concurrent use, like the real services, because
// request-reply handlers run on a worker pool.
type fakeStore struct {
	mu       sync.Mutex
	posts    map[string]contract.PostResponse
	comments map[string]contract.CommentResponse
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		posts:    make(map[string]contract.PostResponse),
		comments: make(map[string]contract.CommentResponse),
	}
}

type fakePosts struct{ s *fakeStore }

var _ blog.PostService = fakePosts{}

func (f fakePosts) Create(_ context.Context, req contract.CreatePostRequest) (contract.PostResponse, error) {
	if req.Title == "" || req.Content == "" || req.Slug == "" {
		return contract.PostResponse{}, errors.New("title, content and slug are required")
	}
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	now := time.Now()
	p := contract.PostResponse{
		ID: uuid.NewString(), Title: req.Title, Content: req.Content, Slug: req.Slug,
		Published: req.Published, CreatedAt: now, UpdatedAt: now,
	}
	f.s.posts[p.ID] = p
	return p, nil
}

func (f fakePosts) Get(_ context.Context, req contract.GetPostRequest) (contract.PostResponse, error) {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	for _, p := range f.s.posts {
		if (req.ID != "" && p.ID == req.ID) || (req.ID == "" && req.Slug != "" && p.Slug == req.Slug) {
			return p, nil
		}
	}
	return contract.PostResponse{}, blog.ErrPostNotFound
}

func (f fakePosts) List(_ context.Context, req contract.ListPostsRequest) (contract.ListPostsResponse, error) {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	resp := contract.ListPostsResponse{Posts: []contract.PostResponse{}, Limit: req.Limit, Offset: req.Offset}
	for _, p := range f.s.posts {
		if req.Published == nil || p.Published == *req.Published {
			resp.Posts = append(resp.Posts, p)
		}
	}
	resp.Total = int64(len(resp.Posts))
	return resp, nil
}

func (f fakePosts) Update(_ context.Context, req contract.UpdatePostRequest) (contract.PostResponse, error) {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	p, ok := f.s.posts[req.ID]
	if !ok {
		return contract.PostResponse{}, blog.ErrPostNotFound
	}
	if req.Title != nil {
		p.Title = *req.Title
	}
	if req.Content != nil {
		p.Content = *req.Content
	}
	if req.Slug != nil {
		p.Slug = *req.Slug
	}
	if req.Published != nil {
		p.Published = *req.Published
	}
	p.UpdatedAt = time.Now()
	f.s.posts[p.ID] = p
	return p, nil
}

func (f fakePosts) Delete(_ context.Context, req contract.DeletePostRequest) (contract.DeletePostResponse, error) {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	if _, ok := f.s.posts[req.ID]; !ok {
		return contract.DeletePostResponse{ID: req.ID}, blog.ErrPostNotFound
	}
	delete(f.s.posts, req.ID)
	for id, c := range f.s.comments {
		if c.PostID == req.ID {
			delete(f.s.comments, id) // ON DELETE CASCADE
		}
	}
	return contract.DeletePostResponse{Deleted: true, ID: req.ID}, nil
}

func (f fakePosts) Publish(ctx context.Context, req contract.PublishPostRequest) (contract.PostResponse, error) {
	published := true
	return f.Update(ctx, contract.UpdatePostRequest{ID: req.ID, Published: &published})
}

type fakeComments struct{ s *fakeStore }

var _ blog.CommentService = fakeComments{}

func (f fakeComments) Create(_ context.Context, req contract.CreateCommentRequest) (contract.CommentResponse, error) {
	if req.Author == "" || req.Content == "" {
		return contract.CommentResponse{}, errors.New("author and content are required")
	}
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	if _, ok := f.s.posts[req.PostID]; !ok {
		return contract.CommentResponse{}, blog.ErrPostDoesNotExist
	}
	now := time.Now()
	c := contract.CommentResponse{
		ID: uuid.NewString(), PostID: req.PostID, Author: req.Author, Content: req.Content,
		CreatedAt: now, UpdatedAt: now,
	}
	f.s.comments[c.ID] = c
	return c, nil
}

func (f fakeComments) Get(_ context.Context, req contract.GetCommentRequest) (contract.CommentResponse, error) {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	c, ok := f.s.comments[req.ID]
	if !ok {
		return contract.CommentResponse{}, blog.ErrCommentNotFound
	}
	return c, nil
}

func (f fakeComments) List(_ context.Context, req contract.ListCommentsRequest) (contract.ListCommentsResponse, error) {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	resp := contract.ListCommentsResponse{Comments: []contract.CommentResponse{}, Limit: req.Limit, Offset: req.Offset}
	for _, c := range f.s.comments {
		if c.PostID == req.PostID {
			resp.Comments = append(resp.Comments, c)
		}
	}
	resp.Total = int64(len(resp.Comments))
	return resp, nil
}

func (f fakeComments) Update(_ context.Context, req contract.UpdateCommentRequest) (contract.CommentResponse, error) {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	c, ok := f.s.comments[req.ID]
	if !ok {
		return contract.CommentResponse{}, blog.ErrCommentNotFound
	}
	if req.Author != nil {
		c.Author = *req.Author
	}
	if req.Content != nil {
		c.Content = *req.Content
	}
	c.UpdatedAt = time.Now()
	f.s.comments[c.ID] = c
	return c, nil
}

func (f fakeComments) Delete(_ context.Context, req contract.DeleteCommentRequest) (contract.DeleteCommentResponse, error) {
	f.s.mu.Lock()
	defer f.s.mu.Unlock()
	if _, ok := f.s.comments[req.ID]; !ok {
		return contract.DeleteCommentResponse{ID: req.ID}, blog.ErrCommentNotFound
	}
	delete(f.s.comments, req.ID)
	return contract.DeleteCommentResponse{Deleted: true, ID: req.ID}, nil
}
