package article

import (
	"context"
	"errors"

	"github.com/example/prisma-postgres-demo/modules/article/db/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Validation errors (exported for error checking via errors.Is).
var (
	ErrTitleRequired    = errors.New("title is required")
	ErrContentRequired  = errors.New("content is required")
	ErrSlugRequired     = errors.New("slug is required")
	ErrIDRequired       = errors.New("id is required")
	ErrIDInvalid        = errors.New("id is not a valid UUID")
	ErrIDOrSlugRequired = errors.New("id or slug is required")
)

// ArticleService defines the interface for article business operations.
type ArticleService interface {
	// Create creates a new article.
	Create(ctx context.Context, req CreateArticleRequest) (ArticleResponse, error)
	// Get retrieves an article by ID or slug.
	Get(ctx context.Context, req GetArticleRequest) (ArticleResponse, error)
	// List retrieves paginated articles with optional published filter.
	List(ctx context.Context, req ListArticlesRequest) (ListArticlesResponse, error)
	// Update updates an existing article.
	Update(ctx context.Context, req UpdateArticleRequest) (ArticleResponse, error)
	// Delete removes an article by ID.
	Delete(ctx context.Context, req DeleteArticleRequest) (DeleteArticleResponse, error)
	// Publish sets an article's published status to true.
	Publish(ctx context.Context, req PublishArticleRequest) (ArticleResponse, error)
}

// ArticleServiceImpl implements ArticleService using ArticleRepository.
type ArticleServiceImpl struct {
	repo ArticleRepository
}

// Compile-time interface check.
var _ ArticleService = (*ArticleServiceImpl)(nil)

// NewArticleService creates a new ArticleService with the given repository.
func NewArticleService(repo ArticleRepository) ArticleService {
	return &ArticleServiceImpl{
		repo: repo,
	}
}

// Create handles the article creation request.
func (s *ArticleServiceImpl) Create(ctx context.Context, req CreateArticleRequest) (ArticleResponse, error) {
	if req.Title == "" {
		return ArticleResponse{}, ErrTitleRequired
	}
	if req.Content == "" {
		return ArticleResponse{}, ErrContentRequired
	}
	if req.Slug == "" {
		return ArticleResponse{}, ErrSlugRequired
	}

	article, err := s.repo.Create(ctx, req.Title, req.Content, req.Slug, req.Published)
	if err != nil {
		return ArticleResponse{}, err
	}

	return toArticleResponse(article), nil
}

// Get handles the article retrieval request (by ID or slug).
func (s *ArticleServiceImpl) Get(ctx context.Context, req GetArticleRequest) (ArticleResponse, error) {
	if req.ID == "" && req.Slug == "" {
		return ArticleResponse{}, ErrIDOrSlugRequired
	}

	var article *generated.Article
	var err error

	if req.ID != "" {
		id, parseErr := uuid.Parse(req.ID)
		if parseErr != nil {
			return ArticleResponse{}, ErrIDInvalid
		}
		article, err = s.repo.FindByID(ctx, id)
	} else {
		article, err = s.repo.FindBySlug(ctx, req.Slug)
	}

	if err != nil {
		return ArticleResponse{}, err
	}

	return toArticleResponse(article), nil
}

// List handles the article list request with pagination and optional published filter.
func (s *ArticleServiceImpl) List(ctx context.Context, req ListArticlesRequest) (ListArticlesResponse, error) {
	limit := clampLimit(req.Limit)
	offset := clampOffset(req.Offset)

	var articles []generated.Article
	var total int64
	var err error

	if req.Published != nil {
		articles, err = s.repo.FindAllByPublished(ctx, *req.Published, limit, offset)
		if err != nil {
			return ListArticlesResponse{}, err
		}
		total, err = s.repo.CountByPublished(ctx, *req.Published)
	} else {
		articles, err = s.repo.FindAll(ctx, limit, offset)
		if err != nil {
			return ListArticlesResponse{}, err
		}
		total, err = s.repo.Count(ctx)
	}

	if err != nil {
		return ListArticlesResponse{}, err
	}

	articleResponses := make([]ArticleResponse, len(articles))
	for i := range articles {
		articleResponses[i] = toArticleResponse(&articles[i])
	}

	return ListArticlesResponse{
		Articles: articleResponses,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}, nil
}

// Update handles the article update request.
func (s *ArticleServiceImpl) Update(ctx context.Context, req UpdateArticleRequest) (ArticleResponse, error) {
	if req.ID == "" {
		return ArticleResponse{}, ErrIDRequired
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return ArticleResponse{}, ErrIDInvalid
	}

	// Validate non-empty values when pointers are provided
	if req.Title != nil && *req.Title == "" {
		return ArticleResponse{}, ErrTitleRequired
	}
	if req.Content != nil && *req.Content == "" {
		return ArticleResponse{}, ErrContentRequired
	}
	if req.Slug != nil && *req.Slug == "" {
		return ArticleResponse{}, ErrSlugRequired
	}

	article, err := s.repo.Update(ctx, id, req.Title, req.Content, req.Slug, req.Published)
	if err != nil {
		return ArticleResponse{}, err
	}

	return toArticleResponse(article), nil
}

// Delete handles the article deletion request.
func (s *ArticleServiceImpl) Delete(ctx context.Context, req DeleteArticleRequest) (DeleteArticleResponse, error) {
	if req.ID == "" {
		return DeleteArticleResponse{}, ErrIDRequired
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return DeleteArticleResponse{}, ErrIDInvalid
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return DeleteArticleResponse{ID: req.ID}, err
	}

	return DeleteArticleResponse{Deleted: true, ID: req.ID}, nil
}

// Publish handles the article publish request.
func (s *ArticleServiceImpl) Publish(ctx context.Context, req PublishArticleRequest) (ArticleResponse, error) {
	if req.ID == "" {
		return ArticleResponse{}, ErrIDRequired
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return ArticleResponse{}, ErrIDInvalid
	}

	article, err := s.repo.Publish(ctx, id)
	if err != nil {
		return ArticleResponse{}, err
	}

	return toArticleResponse(article), nil
}

// clampLimit ensures limit is within valid bounds (1-100, default 10).
func clampLimit(limit int32) int32 {
	if limit <= 0 {
		return 10
	}
	if limit > 100 {
		return 100
	}
	return limit
}

// clampOffset ensures offset is non-negative.
func clampOffset(offset int32) int32 {
	if offset < 0 {
		return 0
	}
	return offset
}

func toArticleResponse(article *generated.Article) ArticleResponse {
	return ArticleResponse{
		ID:        uuidToString(article.ID),
		Title:     article.Title,
		Content:   article.Content,
		Slug:      article.Slug,
		Published: article.Published,
		CreatedAt: article.CreatedAt.Time,
		UpdatedAt: article.UpdatedAt.Time,
	}
}

func uuidToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return uuid.UUID(id.Bytes).String()
}
