package blog

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/example/prisma-postgres-demo/modules/blog/db/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// === Mock Post Repository ===

type mockPostRepository struct {
	posts                    map[uuid.UUID]*generated.BlogPost
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

var _ PostRepository = (*mockPostRepository)(nil)

func newMockPostRepository() *mockPostRepository {
	return &mockPostRepository{
		posts:     make(map[uuid.UUID]*generated.BlogPost),
		slugIndex: make(map[string]uuid.UUID),
	}
}

func (m *mockPostRepository) Create(_ context.Context, title, content, slug string, published bool) (*generated.BlogPost, error) {
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
	post := &generated.BlogPost{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		Title:     title,
		Content:   content,
		Slug:      slug,
		Published: published,
		CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
	}
	m.posts[id] = post
	m.slugIndex[slug] = id
	return post, nil
}

func (m *mockPostRepository) FindByID(_ context.Context, id uuid.UUID) (*generated.BlogPost, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	post, exists := m.posts[id]
	if !exists {
		return nil, ErrPostNotFound
	}
	return post, nil
}

func (m *mockPostRepository) FindBySlug(_ context.Context, slug string) (*generated.BlogPost, error) {
	if m.findBySlugErr != nil {
		return nil, m.findBySlugErr
	}
	id, exists := m.slugIndex[slug]
	if !exists {
		return nil, ErrPostNotFound
	}
	return m.posts[id], nil
}

func (m *mockPostRepository) FindAll(_ context.Context, limit, offset int32) ([]generated.BlogPost, error) {
	if m.findAllErr != nil {
		return nil, m.findAllErr
	}

	posts := make([]generated.BlogPost, 0, len(m.posts))
	for _, p := range m.posts {
		posts = append(posts, *p)
	}

	start := int(offset)
	if start >= len(posts) {
		return []generated.BlogPost{}, nil
	}

	end := start + int(limit)
	if end > len(posts) {
		end = len(posts)
	}

	return posts[start:end], nil
}

func (m *mockPostRepository) FindAllByPublished(_ context.Context, published bool, limit, offset int32) ([]generated.BlogPost, error) {
	if m.findAllByPublishedErr != nil {
		return nil, m.findAllByPublishedErr
	}

	posts := make([]generated.BlogPost, 0)
	for _, p := range m.posts {
		if p.Published == published {
			posts = append(posts, *p)
		}
	}

	start := int(offset)
	if start >= len(posts) {
		return []generated.BlogPost{}, nil
	}

	end := start + int(limit)
	if end > len(posts) {
		end = len(posts)
	}

	return posts[start:end], nil
}

func (m *mockPostRepository) Count(_ context.Context) (int64, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	if m.returnedCount > 0 {
		return m.returnedCount, nil
	}
	return int64(len(m.posts)), nil
}

func (m *mockPostRepository) CountByPublished(_ context.Context, published bool) (int64, error) {
	if m.countByPublishedErr != nil {
		return 0, m.countByPublishedErr
	}
	if m.returnedCountByPublished > 0 {
		return m.returnedCountByPublished, nil
	}
	count := int64(0)
	for _, p := range m.posts {
		if p.Published == published {
			count++
		}
	}
	return count, nil
}

func (m *mockPostRepository) Update(_ context.Context, id uuid.UUID, title, content, slug *string, published *bool) (*generated.BlogPost, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	if m.slugExists && slug != nil {
		return nil, ErrDuplicateSlug
	}

	post, exists := m.posts[id]
	if !exists {
		return nil, ErrPostNotFound
	}

	if slug != nil && *slug != post.Slug {
		if _, exists := m.slugIndex[*slug]; exists {
			return nil, ErrDuplicateSlug
		}
		delete(m.slugIndex, post.Slug)
		m.slugIndex[*slug] = id
		post.Slug = *slug
	}

	if title != nil {
		post.Title = *title
	}
	if content != nil {
		post.Content = *content
	}
	if published != nil {
		post.Published = *published
	}
	post.UpdatedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}

	return post, nil
}

func (m *mockPostRepository) Publish(_ context.Context, id uuid.UUID) (*generated.BlogPost, error) {
	if m.publishErr != nil {
		return nil, m.publishErr
	}

	post, exists := m.posts[id]
	if !exists {
		return nil, ErrPostNotFound
	}

	post.Published = true
	post.UpdatedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}

	return post, nil
}

func (m *mockPostRepository) Delete(_ context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	post, exists := m.posts[id]
	if exists {
		delete(m.slugIndex, post.Slug)
	}
	delete(m.posts, id)
	return nil
}

// === Mock Comment Repository ===

type mockCommentRepository struct {
	comments         map[uuid.UUID]*generated.BlogComment
	postComments     map[uuid.UUID][]uuid.UUID
	createErr        error
	findByIDErr      error
	findAllErr       error
	countErr         error
	updateErr        error
	deleteErr        error
	returnedCount    int64
}

var _ CommentRepository = (*mockCommentRepository)(nil)

func newMockCommentRepository() *mockCommentRepository {
	return &mockCommentRepository{
		comments:     make(map[uuid.UUID]*generated.BlogComment),
		postComments: make(map[uuid.UUID][]uuid.UUID),
	}
}

func (m *mockCommentRepository) Create(_ context.Context, postID uuid.UUID, author, content string) (*generated.BlogComment, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}

	id := uuid.New()
	now := time.Now()
	comment := &generated.BlogComment{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		PostID:    pgtype.UUID{Bytes: postID, Valid: true},
		Author:    author,
		Content:   content,
		CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
	}
	m.comments[id] = comment
	m.postComments[postID] = append(m.postComments[postID], id)
	return comment, nil
}

func (m *mockCommentRepository) FindByID(_ context.Context, id uuid.UUID) (*generated.BlogComment, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	comment, exists := m.comments[id]
	if !exists {
		return nil, ErrCommentNotFound
	}
	return comment, nil
}

func (m *mockCommentRepository) FindAllByPostID(_ context.Context, postID uuid.UUID, limit, offset int32) ([]generated.BlogComment, error) {
	if m.findAllErr != nil {
		return nil, m.findAllErr
	}

	commentIDs := m.postComments[postID]
	comments := make([]generated.BlogComment, 0)
	for _, id := range commentIDs {
		if c, exists := m.comments[id]; exists {
			comments = append(comments, *c)
		}
	}

	start := int(offset)
	if start >= len(comments) {
		return []generated.BlogComment{}, nil
	}

	end := start + int(limit)
	if end > len(comments) {
		end = len(comments)
	}

	return comments[start:end], nil
}

func (m *mockCommentRepository) CountByPostID(_ context.Context, postID uuid.UUID) (int64, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	if m.returnedCount > 0 {
		return m.returnedCount, nil
	}
	return int64(len(m.postComments[postID])), nil
}

func (m *mockCommentRepository) Update(_ context.Context, id uuid.UUID, author, content *string) (*generated.BlogComment, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}

	comment, exists := m.comments[id]
	if !exists {
		return nil, ErrCommentNotFound
	}

	if author != nil {
		comment.Author = *author
	}
	if content != nil {
		comment.Content = *content
	}
	comment.UpdatedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}

	return comment, nil
}

func (m *mockCommentRepository) Delete(_ context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	comment, exists := m.comments[id]
	if exists {
		postID := uuid.UUID(comment.PostID.Bytes)
		ids := m.postComments[postID]
		for i, cid := range ids {
			if cid == id {
				m.postComments[postID] = append(ids[:i], ids[i+1:]...)
				break
			}
		}
	}
	delete(m.comments, id)
	return nil
}

// === Post Service Tests ===

func TestPostService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		resp, err := svc.Create(context.Background(), CreatePostRequest{
			Title:     "Test Post",
			Content:   "Test content here",
			Slug:      "test-post",
			Published: false,
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if resp.Title != "Test Post" {
			t.Errorf("expected title %q, got %q", "Test Post", resp.Title)
		}
		if resp.Slug != "test-post" {
			t.Errorf("expected slug %q, got %q", "test-post", resp.Slug)
		}
		if resp.Published {
			t.Error("expected Published to be false")
		}
		if resp.ID == "" {
			t.Error("expected non-empty ID")
		}
	})

	t.Run("missing title", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		_, err := svc.Create(context.Background(), CreatePostRequest{
			Content: "Test content",
			Slug:    "test-slug",
		})
		if err != ErrPostTitleRequired {
			t.Errorf("expected ErrPostTitleRequired, got %v", err)
		}
	})

	t.Run("missing content", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		_, err := svc.Create(context.Background(), CreatePostRequest{
			Title: "Test Post",
			Slug:  "test-slug",
		})
		if err != ErrPostContentRequired {
			t.Errorf("expected ErrPostContentRequired, got %v", err)
		}
	})

	t.Run("missing slug", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		_, err := svc.Create(context.Background(), CreatePostRequest{
			Title:   "Test Post",
			Content: "Test content",
		})
		if err != ErrPostSlugRequired {
			t.Errorf("expected ErrPostSlugRequired, got %v", err)
		}
	})

	t.Run("duplicate slug", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		_, err := svc.Create(context.Background(), CreatePostRequest{
			Title:   "First Post",
			Content: "Content",
			Slug:    "same-slug",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		_, err = svc.Create(context.Background(), CreatePostRequest{
			Title:   "Second Post",
			Content: "Content",
			Slug:    "same-slug",
		})
		if err != ErrDuplicateSlug {
			t.Errorf("expected ErrDuplicateSlug, got %v", err)
		}
	})
}

func TestPostService_Get(t *testing.T) {
	t.Run("success by ID", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		created, err := svc.Create(context.Background(), CreatePostRequest{
			Title:   "Test Post",
			Content: "Test content",
			Slug:    "test-post",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		resp, err := svc.Get(context.Background(), GetPostRequest{ID: created.ID})
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if resp.Title != created.Title {
			t.Errorf("expected title %q, got %q", created.Title, resp.Title)
		}
	})

	t.Run("success by slug", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		created, err := svc.Create(context.Background(), CreatePostRequest{
			Title:   "Test Post",
			Content: "Test content",
			Slug:    "test-post",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		resp, err := svc.Get(context.Background(), GetPostRequest{Slug: "test-post"})
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if resp.Title != created.Title {
			t.Errorf("expected title %q, got %q", created.Title, resp.Title)
		}
	})

	t.Run("missing ID and slug", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		_, err := svc.Get(context.Background(), GetPostRequest{})
		if err != ErrPostIDOrSlugRequired {
			t.Errorf("expected ErrPostIDOrSlugRequired, got %v", err)
		}
	})

	t.Run("invalid UUID", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		_, err := svc.Get(context.Background(), GetPostRequest{ID: "not-a-uuid"})
		if err != ErrPostIDInvalid {
			t.Errorf("expected ErrPostIDInvalid, got %v", err)
		}
	})

	t.Run("not found by ID", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		_, err := svc.Get(context.Background(), GetPostRequest{
			ID: uuid.New().String(),
		})
		if err != ErrPostNotFound {
			t.Errorf("expected ErrPostNotFound, got %v", err)
		}
	})
}

func TestPostService_List(t *testing.T) {
	t.Run("success with pagination", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		for i := range 5 {
			_, err := svc.Create(context.Background(), CreatePostRequest{
				Title:   "Test Post",
				Content: "Content",
				Slug:    fmt.Sprintf("test-post-%d", i),
			})
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
		}

		resp, err := svc.List(context.Background(), ListPostsRequest{
			Limit:  2,
			Offset: 0,
		})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(resp.Posts) != 2 {
			t.Errorf("expected 2 posts, got %d", len(resp.Posts))
		}
		if resp.Total != 5 {
			t.Errorf("expected total 5, got %d", resp.Total)
		}
		if resp.Limit != 2 {
			t.Errorf("expected limit 2, got %d", resp.Limit)
		}
	})

	t.Run("filter by published", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		for i := range 3 {
			_, err := svc.Create(context.Background(), CreatePostRequest{
				Title:     "Published Post",
				Content:   "Content",
				Slug:      fmt.Sprintf("published-%d", i),
				Published: true,
			})
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
		}
		for i := range 2 {
			_, err := svc.Create(context.Background(), CreatePostRequest{
				Title:     "Draft Post",
				Content:   "Content",
				Slug:      fmt.Sprintf("draft-%d", i),
				Published: false,
			})
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
		}

		published := true
		resp, err := svc.List(context.Background(), ListPostsRequest{
			Published: &published,
		})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if resp.Total != 3 {
			t.Errorf("expected 3 published posts, got %d", resp.Total)
		}
	})

	t.Run("default pagination", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		resp, err := svc.List(context.Background(), ListPostsRequest{})
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
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		resp, err := svc.List(context.Background(), ListPostsRequest{
			Limit: 200,
		})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if resp.Limit != 100 {
			t.Errorf("expected clamped limit 100, got %d", resp.Limit)
		}
	})
}

func TestPostService_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		created, err := svc.Create(context.Background(), CreatePostRequest{
			Title:   "Original Title",
			Content: "Original content",
			Slug:    "original-slug",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		newTitle := "Updated Title"
		resp, err := svc.Update(context.Background(), UpdatePostRequest{
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
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		newTitle := "Updated Title"
		_, err := svc.Update(context.Background(), UpdatePostRequest{
			Title: &newTitle,
		})
		if err != ErrPostIDRequired {
			t.Errorf("expected ErrPostIDRequired, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		newTitle := "Updated Title"
		_, err := svc.Update(context.Background(), UpdatePostRequest{
			ID:    uuid.New().String(),
			Title: &newTitle,
		})
		if !errors.Is(err, ErrPostNotFound) {
			t.Errorf("expected ErrPostNotFound, got %v", err)
		}
	})

	t.Run("empty title rejected", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		created, err := svc.Create(context.Background(), CreatePostRequest{
			Title:   "Original Title",
			Content: "Content",
			Slug:    "test-slug",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		emptyTitle := ""
		_, err = svc.Update(context.Background(), UpdatePostRequest{
			ID:    created.ID,
			Title: &emptyTitle,
		})
		if !errors.Is(err, ErrPostTitleRequired) {
			t.Errorf("expected ErrPostTitleRequired, got %v", err)
		}
	})
}

func TestPostService_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		created, err := svc.Create(context.Background(), CreatePostRequest{
			Title:   "Test Post",
			Content: "Content",
			Slug:    "test-slug",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		resp, err := svc.Delete(context.Background(), DeletePostRequest{
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

		_, err = svc.Get(context.Background(), GetPostRequest{ID: created.ID})
		if err != ErrPostNotFound {
			t.Errorf("expected ErrPostNotFound after delete, got %v", err)
		}
	})

	t.Run("missing ID", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		_, err := svc.Delete(context.Background(), DeletePostRequest{})
		if err != ErrPostIDRequired {
			t.Errorf("expected ErrPostIDRequired, got %v", err)
		}
	})

	t.Run("invalid UUID", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		_, err := svc.Delete(context.Background(), DeletePostRequest{ID: "not-a-uuid"})
		if err != ErrPostIDInvalid {
			t.Errorf("expected ErrPostIDInvalid, got %v", err)
		}
	})
}

func TestPostService_Publish(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		created, err := svc.Create(context.Background(), CreatePostRequest{
			Title:     "Draft Post",
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

		resp, err := svc.Publish(context.Background(), PublishPostRequest{
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
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		_, err := svc.Publish(context.Background(), PublishPostRequest{})
		if err != ErrPostIDRequired {
			t.Errorf("expected ErrPostIDRequired, got %v", err)
		}
	})

	t.Run("invalid UUID", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		_, err := svc.Publish(context.Background(), PublishPostRequest{ID: "not-a-uuid"})
		if err != ErrPostIDInvalid {
			t.Errorf("expected ErrPostIDInvalid, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := newMockPostRepository()
		svc := NewPostService(repo)

		_, err := svc.Publish(context.Background(), PublishPostRequest{
			ID: uuid.New().String(),
		})
		if err != ErrPostNotFound {
			t.Errorf("expected ErrPostNotFound, got %v", err)
		}
	})
}

// === Comment Service Tests ===

func TestCommentService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		postRepo := newMockPostRepository()
		commentRepo := newMockCommentRepository()
		postSvc := NewPostService(postRepo)
		commentSvc := NewCommentService(commentRepo, postRepo)

		post, err := postSvc.Create(context.Background(), CreatePostRequest{
			Title:   "Test Post",
			Content: "Content",
			Slug:    "test-post",
		})
		if err != nil {
			t.Fatalf("Create post error = %v", err)
		}

		resp, err := commentSvc.Create(context.Background(), CreateCommentRequest{
			PostID:  post.ID,
			Author:  "Test Author",
			Content: "Test comment content",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if resp.Author != "Test Author" {
			t.Errorf("expected author %q, got %q", "Test Author", resp.Author)
		}
		if resp.PostID != post.ID {
			t.Errorf("expected post_id %q, got %q", post.ID, resp.PostID)
		}
	})

	t.Run("missing post_id", func(t *testing.T) {
		postRepo := newMockPostRepository()
		commentRepo := newMockCommentRepository()
		commentSvc := NewCommentService(commentRepo, postRepo)

		_, err := commentSvc.Create(context.Background(), CreateCommentRequest{
			Author:  "Test Author",
			Content: "Test content",
		})
		if err != ErrCommentPostIDRequired {
			t.Errorf("expected ErrCommentPostIDRequired, got %v", err)
		}
	})

	t.Run("missing author", func(t *testing.T) {
		postRepo := newMockPostRepository()
		commentRepo := newMockCommentRepository()
		commentSvc := NewCommentService(commentRepo, postRepo)

		_, err := commentSvc.Create(context.Background(), CreateCommentRequest{
			PostID:  uuid.New().String(),
			Content: "Test content",
		})
		if err != ErrCommentAuthorRequired {
			t.Errorf("expected ErrCommentAuthorRequired, got %v", err)
		}
	})

	t.Run("missing content", func(t *testing.T) {
		postRepo := newMockPostRepository()
		commentRepo := newMockCommentRepository()
		commentSvc := NewCommentService(commentRepo, postRepo)

		_, err := commentSvc.Create(context.Background(), CreateCommentRequest{
			PostID: uuid.New().String(),
			Author: "Test Author",
		})
		if err != ErrCommentContentRequired {
			t.Errorf("expected ErrCommentContentRequired, got %v", err)
		}
	})

	t.Run("post not found", func(t *testing.T) {
		postRepo := newMockPostRepository()
		commentRepo := newMockCommentRepository()
		// Simulate foreign key constraint error when post doesn't exist
		commentRepo.createErr = ErrPostDoesNotExist
		commentSvc := NewCommentService(commentRepo, postRepo)

		_, err := commentSvc.Create(context.Background(), CreateCommentRequest{
			PostID:  uuid.New().String(),
			Author:  "Test Author",
			Content: "Test content",
		})
		if err != ErrPostDoesNotExist {
			t.Errorf("expected ErrPostDoesNotExist, got %v", err)
		}
	})
}

func TestCommentService_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		postRepo := newMockPostRepository()
		commentRepo := newMockCommentRepository()
		postSvc := NewPostService(postRepo)
		commentSvc := NewCommentService(commentRepo, postRepo)

		post, err := postSvc.Create(context.Background(), CreatePostRequest{
			Title:   "Test Post",
			Content: "Content",
			Slug:    "test-post",
		})
		if err != nil {
			t.Fatalf("Create post error = %v", err)
		}

		created, err := commentSvc.Create(context.Background(), CreateCommentRequest{
			PostID:  post.ID,
			Author:  "Test Author",
			Content: "Test content",
		})
		if err != nil {
			t.Fatalf("Create comment error = %v", err)
		}

		resp, err := commentSvc.Get(context.Background(), GetCommentRequest{ID: created.ID})
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if resp.Author != created.Author {
			t.Errorf("expected author %q, got %q", created.Author, resp.Author)
		}
	})

	t.Run("missing ID", func(t *testing.T) {
		postRepo := newMockPostRepository()
		commentRepo := newMockCommentRepository()
		commentSvc := NewCommentService(commentRepo, postRepo)

		_, err := commentSvc.Get(context.Background(), GetCommentRequest{})
		if err != ErrCommentIDRequired {
			t.Errorf("expected ErrCommentIDRequired, got %v", err)
		}
	})

	t.Run("invalid UUID", func(t *testing.T) {
		postRepo := newMockPostRepository()
		commentRepo := newMockCommentRepository()
		commentSvc := NewCommentService(commentRepo, postRepo)

		_, err := commentSvc.Get(context.Background(), GetCommentRequest{ID: "not-a-uuid"})
		if err != ErrCommentIDInvalid {
			t.Errorf("expected ErrCommentIDInvalid, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		postRepo := newMockPostRepository()
		commentRepo := newMockCommentRepository()
		commentSvc := NewCommentService(commentRepo, postRepo)

		_, err := commentSvc.Get(context.Background(), GetCommentRequest{ID: uuid.New().String()})
		if err != ErrCommentNotFound {
			t.Errorf("expected ErrCommentNotFound, got %v", err)
		}
	})
}

func TestCommentService_List(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		postRepo := newMockPostRepository()
		commentRepo := newMockCommentRepository()
		postSvc := NewPostService(postRepo)
		commentSvc := NewCommentService(commentRepo, postRepo)

		post, err := postSvc.Create(context.Background(), CreatePostRequest{
			Title:   "Test Post",
			Content: "Content",
			Slug:    "test-post",
		})
		if err != nil {
			t.Fatalf("Create post error = %v", err)
		}

		for i := range 5 {
			_, err := commentSvc.Create(context.Background(), CreateCommentRequest{
				PostID:  post.ID,
				Author:  fmt.Sprintf("Author %d", i),
				Content: fmt.Sprintf("Comment %d", i),
			})
			if err != nil {
				t.Fatalf("Create comment error = %v", err)
			}
		}

		resp, err := commentSvc.List(context.Background(), ListCommentsRequest{
			PostID: post.ID,
			Limit:  2,
			Offset: 0,
		})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(resp.Comments) != 2 {
			t.Errorf("expected 2 comments, got %d", len(resp.Comments))
		}
		if resp.Total != 5 {
			t.Errorf("expected total 5, got %d", resp.Total)
		}
	})

	t.Run("missing post_id", func(t *testing.T) {
		postRepo := newMockPostRepository()
		commentRepo := newMockCommentRepository()
		commentSvc := NewCommentService(commentRepo, postRepo)

		_, err := commentSvc.List(context.Background(), ListCommentsRequest{})
		if err != ErrCommentPostIDRequired {
			t.Errorf("expected ErrCommentPostIDRequired, got %v", err)
		}
	})

	t.Run("invalid post_id", func(t *testing.T) {
		postRepo := newMockPostRepository()
		commentRepo := newMockCommentRepository()
		commentSvc := NewCommentService(commentRepo, postRepo)

		_, err := commentSvc.List(context.Background(), ListCommentsRequest{PostID: "not-a-uuid"})
		if err != ErrCommentPostIDInvalid {
			t.Errorf("expected ErrCommentPostIDInvalid, got %v", err)
		}
	})
}

func TestCommentService_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		postRepo := newMockPostRepository()
		commentRepo := newMockCommentRepository()
		postSvc := NewPostService(postRepo)
		commentSvc := NewCommentService(commentRepo, postRepo)

		post, err := postSvc.Create(context.Background(), CreatePostRequest{
			Title:   "Test Post",
			Content: "Content",
			Slug:    "test-post",
		})
		if err != nil {
			t.Fatalf("Create post error = %v", err)
		}

		created, err := commentSvc.Create(context.Background(), CreateCommentRequest{
			PostID:  post.ID,
			Author:  "Original Author",
			Content: "Original content",
		})
		if err != nil {
			t.Fatalf("Create comment error = %v", err)
		}

		newAuthor := "Updated Author"
		resp, err := commentSvc.Update(context.Background(), UpdateCommentRequest{
			ID:     created.ID,
			Author: &newAuthor,
		})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		if resp.Author != newAuthor {
			t.Errorf("expected author %q, got %q", newAuthor, resp.Author)
		}
	})

	t.Run("missing ID", func(t *testing.T) {
		postRepo := newMockPostRepository()
		commentRepo := newMockCommentRepository()
		commentSvc := NewCommentService(commentRepo, postRepo)

		newAuthor := "Updated Author"
		_, err := commentSvc.Update(context.Background(), UpdateCommentRequest{
			Author: &newAuthor,
		})
		if err != ErrCommentIDRequired {
			t.Errorf("expected ErrCommentIDRequired, got %v", err)
		}
	})

	t.Run("empty author rejected", func(t *testing.T) {
		postRepo := newMockPostRepository()
		commentRepo := newMockCommentRepository()
		postSvc := NewPostService(postRepo)
		commentSvc := NewCommentService(commentRepo, postRepo)

		post, err := postSvc.Create(context.Background(), CreatePostRequest{
			Title:   "Test Post",
			Content: "Content",
			Slug:    "test-post",
		})
		if err != nil {
			t.Fatalf("Create post error = %v", err)
		}

		created, err := commentSvc.Create(context.Background(), CreateCommentRequest{
			PostID:  post.ID,
			Author:  "Original Author",
			Content: "Original content",
		})
		if err != nil {
			t.Fatalf("Create comment error = %v", err)
		}

		emptyAuthor := ""
		_, err = commentSvc.Update(context.Background(), UpdateCommentRequest{
			ID:     created.ID,
			Author: &emptyAuthor,
		})
		if !errors.Is(err, ErrCommentAuthorRequired) {
			t.Errorf("expected ErrCommentAuthorRequired, got %v", err)
		}
	})
}

func TestCommentService_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		postRepo := newMockPostRepository()
		commentRepo := newMockCommentRepository()
		postSvc := NewPostService(postRepo)
		commentSvc := NewCommentService(commentRepo, postRepo)

		post, err := postSvc.Create(context.Background(), CreatePostRequest{
			Title:   "Test Post",
			Content: "Content",
			Slug:    "test-post",
		})
		if err != nil {
			t.Fatalf("Create post error = %v", err)
		}

		created, err := commentSvc.Create(context.Background(), CreateCommentRequest{
			PostID:  post.ID,
			Author:  "Test Author",
			Content: "Test content",
		})
		if err != nil {
			t.Fatalf("Create comment error = %v", err)
		}

		resp, err := commentSvc.Delete(context.Background(), DeleteCommentRequest{ID: created.ID})
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		if !resp.Deleted {
			t.Error("expected Deleted to be true")
		}

		_, err = commentSvc.Get(context.Background(), GetCommentRequest{ID: created.ID})
		if err != ErrCommentNotFound {
			t.Errorf("expected ErrCommentNotFound after delete, got %v", err)
		}
	})

	t.Run("missing ID", func(t *testing.T) {
		postRepo := newMockPostRepository()
		commentRepo := newMockCommentRepository()
		commentSvc := NewCommentService(commentRepo, postRepo)

		_, err := commentSvc.Delete(context.Background(), DeleteCommentRequest{})
		if err != ErrCommentIDRequired {
			t.Errorf("expected ErrCommentIDRequired, got %v", err)
		}
	})

	t.Run("invalid UUID", func(t *testing.T) {
		postRepo := newMockPostRepository()
		commentRepo := newMockCommentRepository()
		commentSvc := NewCommentService(commentRepo, postRepo)

		_, err := commentSvc.Delete(context.Background(), DeleteCommentRequest{ID: "not-a-uuid"})
		if err != ErrCommentIDInvalid {
			t.Errorf("expected ErrCommentIDInvalid, got %v", err)
		}
	})
}

// === Helper Function Tests ===

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
