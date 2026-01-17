package article

import (
	"context"
	"errors"

	"github.com/example/prisma-postgres-demo/modules/article/db/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// Repository errors.
var (
	ErrNotFound      = errors.New("article not found")
	ErrDuplicateSlug = errors.New("slug already exists")
)

// ArticleRepository defines the interface for article data access.
type ArticleRepository interface {
	// Create saves a new article to the storage.
	Create(ctx context.Context, title, content, slug string, published bool) (*generated.ArticleModuleArticle, error)
	// FindByID retrieves an article by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*generated.ArticleModuleArticle, error)
	// FindBySlug retrieves an article by slug.
	FindBySlug(ctx context.Context, slug string) (*generated.ArticleModuleArticle, error)
	// FindAll retrieves paginated articles.
	FindAll(ctx context.Context, limit, offset int32) ([]generated.ArticleModuleArticle, error)
	// FindAllByPublished retrieves paginated articles filtered by published status.
	FindAllByPublished(ctx context.Context, published bool, limit, offset int32) ([]generated.ArticleModuleArticle, error)
	// Count returns the total number of articles.
	Count(ctx context.Context) (int64, error)
	// CountByPublished returns the count of articles by published status.
	CountByPublished(ctx context.Context, published bool) (int64, error)
	// Update updates an existing article.
	Update(ctx context.Context, id uuid.UUID, title, content, slug *string, published *bool) (*generated.ArticleModuleArticle, error)
	// Publish sets an article's published status to true.
	Publish(ctx context.Context, id uuid.UUID) (*generated.ArticleModuleArticle, error)
	// Delete removes an article by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}

// PostgresRepository provides PostgreSQL-based article storage using sqlc.
type PostgresRepository struct {
	queries *generated.Queries
}

// Compile-time interface check.
var _ ArticleRepository = (*PostgresRepository)(nil)

// NewPostgresRepository creates a new PostgreSQL article repository.
func NewPostgresRepository(db generated.DBTX) *PostgresRepository {
	return &PostgresRepository{
		queries: generated.New(db),
	}
}

// Create saves a new article to the database.
func (r *PostgresRepository) Create(ctx context.Context, title, content, slug string, published bool) (*generated.ArticleModuleArticle, error) {
	article, err := r.queries.CreateArticle(ctx, generated.CreateArticleParams{
		Title:     title,
		Content:   content,
		Slug:      slug,
		Published: published,
	})
	if err != nil {
		if isPgDuplicateKeyError(err) {
			return nil, ErrDuplicateSlug
		}
		return nil, err
	}
	return &article, nil
}

// FindByID retrieves an article by ID.
func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*generated.ArticleModuleArticle, error) {
	article, err := r.queries.GetArticleByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &article, nil
}

// FindBySlug retrieves an article by slug.
func (r *PostgresRepository) FindBySlug(ctx context.Context, slug string) (*generated.ArticleModuleArticle, error) {
	article, err := r.queries.GetArticleBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &article, nil
}

// FindAll retrieves paginated articles.
func (r *PostgresRepository) FindAll(ctx context.Context, limit, offset int32) ([]generated.ArticleModuleArticle, error) {
	return r.queries.ListArticles(ctx, generated.ListArticlesParams{
		Limit:  limit,
		Offset: offset,
	})
}

// FindAllByPublished retrieves paginated articles filtered by published status.
func (r *PostgresRepository) FindAllByPublished(ctx context.Context, published bool, limit, offset int32) ([]generated.ArticleModuleArticle, error) {
	return r.queries.ListArticlesByPublished(ctx, generated.ListArticlesByPublishedParams{
		Published: published,
		Limit:     limit,
		Offset:    offset,
	})
}

// Count returns the total number of articles.
func (r *PostgresRepository) Count(ctx context.Context) (int64, error) {
	return r.queries.CountArticles(ctx)
}

// CountByPublished returns the count of articles by published status.
func (r *PostgresRepository) CountByPublished(ctx context.Context, published bool) (int64, error) {
	return r.queries.CountArticlesByPublished(ctx, published)
}

// Update updates an existing article.
func (r *PostgresRepository) Update(ctx context.Context, id uuid.UUID, title, content, slug *string, published *bool) (*generated.ArticleModuleArticle, error) {
	params := generated.UpdateArticleParams{
		ID: pgtype.UUID{Bytes: id, Valid: true},
	}
	if title != nil {
		params.Title = pgtype.Text{String: *title, Valid: true}
	}
	if content != nil {
		params.Content = pgtype.Text{String: *content, Valid: true}
	}
	if slug != nil {
		params.Slug = pgtype.Text{String: *slug, Valid: true}
	}
	if published != nil {
		params.Published = pgtype.Bool{Bool: *published, Valid: true}
	}

	article, err := r.queries.UpdateArticle(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if isPgDuplicateKeyError(err) {
			return nil, ErrDuplicateSlug
		}
		return nil, err
	}
	return &article, nil
}

// Publish sets an article's published status to true.
func (r *PostgresRepository) Publish(ctx context.Context, id uuid.UUID) (*generated.ArticleModuleArticle, error) {
	article, err := r.queries.PublishArticle(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &article, nil
}

// Delete removes an article by ID.
func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Check existence first to return ErrNotFound for non-existent articles
	_, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}
	return r.queries.DeleteArticle(ctx, pgtype.UUID{Bytes: id, Valid: true})
}

// isPgDuplicateKeyError checks if error is a PostgreSQL unique violation.
func isPgDuplicateKeyError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
