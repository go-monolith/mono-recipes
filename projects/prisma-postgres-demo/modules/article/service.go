package article

import (
	"context"
	"errors"

	"github.com/example/prisma-postgres-demo/modules/article/db/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/example/prisma-postgres-demo/modules/article/contract"
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
	Create(ctx context.Context, req contract.CreateArticleRequest) (contract.ArticleResponse, error)
	// Get retrieves an article by ID or slug.
	Get(ctx context.Context, req contract.GetArticleRequest) (contract.ArticleResponse, error)
	// List retrieves paginated articles with optional published filter.
	List(ctx context.Context, req contract.ListArticlesRequest) (contract.ListArticlesResponse, error)
	// Update updates an existing article.
	Update(ctx context.Context, req contract.UpdateArticleRequest) (contract.ArticleResponse, error)
	// Delete removes an article by ID.
	Delete(ctx context.Context, req contract.DeleteArticleRequest) (contract.DeleteArticleResponse, error)
	// Publish sets an article's published status to true.
	Publish(ctx context.Context, req contract.PublishArticleRequest) (contract.ArticleResponse, error)
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
func (s *ArticleServiceImpl) Create(ctx context.Context, req contract.CreateArticleRequest) (contract.ArticleResponse, error) {
	if req.Title == "" {
		return contract.ArticleResponse{}, ErrTitleRequired
	}
	if req.Content == "" {
		return contract.ArticleResponse{}, ErrContentRequired
	}
	if req.Slug == "" {
		return contract.ArticleResponse{}, ErrSlugRequired
	}

	article, err := s.repo.Create(ctx, req.Title, req.Content, req.Slug, req.Published)
	if err != nil {
		return contract.ArticleResponse{}, err
	}

	return toArticleResponse(article), nil
}

// Get handles the article retrieval request (by ID or slug).
func (s *ArticleServiceImpl) Get(ctx context.Context, req contract.GetArticleRequest) (contract.ArticleResponse, error) {
	if req.ID == "" && req.Slug == "" {
		return contract.ArticleResponse{}, ErrIDOrSlugRequired
	}

	var article *generated.Article
	var err error

	if req.ID != "" {
		id, parseErr := uuid.Parse(req.ID)
		if parseErr != nil {
			return contract.ArticleResponse{}, ErrIDInvalid
		}
		article, err = s.repo.FindByID(ctx, id)
	} else {
		article, err = s.repo.FindBySlug(ctx, req.Slug)
	}

	if err != nil {
		return contract.ArticleResponse{}, err
	}

	return toArticleResponse(article), nil
}

// List handles the article list request with pagination and optional published filter.
func (s *ArticleServiceImpl) List(ctx context.Context, req contract.ListArticlesRequest) (contract.ListArticlesResponse, error) {
	limit := clampLimit(req.Limit)
	offset := clampOffset(req.Offset)

	var articles []generated.Article
	var total int64
	var err error

	if req.Published != nil {
		articles, err = s.repo.FindAllByPublished(ctx, *req.Published, limit, offset)
		if err != nil {
			return contract.ListArticlesResponse{}, err
		}
		total, err = s.repo.CountByPublished(ctx, *req.Published)
	} else {
		articles, err = s.repo.FindAll(ctx, limit, offset)
		if err != nil {
			return contract.ListArticlesResponse{}, err
		}
		total, err = s.repo.Count(ctx)
	}

	if err != nil {
		return contract.ListArticlesResponse{}, err
	}

	articleResponses := make([]contract.ArticleResponse, len(articles))
	for i := range articles {
		articleResponses[i] = toArticleResponse(&articles[i])
	}

	return contract.ListArticlesResponse{
		Articles: articleResponses,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}, nil
}

// Update handles the article update request.
func (s *ArticleServiceImpl) Update(ctx context.Context, req contract.UpdateArticleRequest) (contract.ArticleResponse, error) {
	if req.ID == "" {
		return contract.ArticleResponse{}, ErrIDRequired
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return contract.ArticleResponse{}, ErrIDInvalid
	}

	// Validate non-empty values when pointers are provided
	if req.Title != nil && *req.Title == "" {
		return contract.ArticleResponse{}, ErrTitleRequired
	}
	if req.Content != nil && *req.Content == "" {
		return contract.ArticleResponse{}, ErrContentRequired
	}
	if req.Slug != nil && *req.Slug == "" {
		return contract.ArticleResponse{}, ErrSlugRequired
	}

	article, err := s.repo.Update(ctx, id, req.Title, req.Content, req.Slug, req.Published)
	if err != nil {
		return contract.ArticleResponse{}, err
	}

	return toArticleResponse(article), nil
}

// Delete handles the article deletion request.
func (s *ArticleServiceImpl) Delete(ctx context.Context, req contract.DeleteArticleRequest) (contract.DeleteArticleResponse, error) {
	if req.ID == "" {
		return contract.DeleteArticleResponse{}, ErrIDRequired
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return contract.DeleteArticleResponse{}, ErrIDInvalid
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return contract.DeleteArticleResponse{ID: req.ID}, err
	}

	return contract.DeleteArticleResponse{Deleted: true, ID: req.ID}, nil
}

// Publish handles the article publish request.
func (s *ArticleServiceImpl) Publish(ctx context.Context, req contract.PublishArticleRequest) (contract.ArticleResponse, error) {
	if req.ID == "" {
		return contract.ArticleResponse{}, ErrIDRequired
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return contract.ArticleResponse{}, ErrIDInvalid
	}

	article, err := s.repo.Publish(ctx, id)
	if err != nil {
		return contract.ArticleResponse{}, err
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

func toArticleResponse(article *generated.Article) contract.ArticleResponse {
	return contract.ArticleResponse{
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
