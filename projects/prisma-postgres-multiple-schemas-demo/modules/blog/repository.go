package blog

import (
	"context"
	"errors"

	"github.com/example/prisma-postgres-demo/modules/blog/db/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// Repository errors.
var (
	ErrPostNotFound     = errors.New("post not found")
	ErrCommentNotFound  = errors.New("comment not found")
	ErrDuplicateSlug    = errors.New("slug already exists")
	ErrPostDoesNotExist = errors.New("post does not exist")
)

// PostRepository defines the interface for post data access.
type PostRepository interface {
	Create(ctx context.Context, title, content, slug string, published bool) (*generated.BlogPost, error)
	FindByID(ctx context.Context, id uuid.UUID) (*generated.BlogPost, error)
	FindBySlug(ctx context.Context, slug string) (*generated.BlogPost, error)
	FindAll(ctx context.Context, limit, offset int32) ([]generated.BlogPost, error)
	FindAllByPublished(ctx context.Context, published bool, limit, offset int32) ([]generated.BlogPost, error)
	Count(ctx context.Context) (int64, error)
	CountByPublished(ctx context.Context, published bool) (int64, error)
	Update(ctx context.Context, id uuid.UUID, title, content, slug *string, published *bool) (*generated.BlogPost, error)
	Publish(ctx context.Context, id uuid.UUID) (*generated.BlogPost, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// CommentRepository defines the interface for comment data access.
type CommentRepository interface {
	Create(ctx context.Context, postID uuid.UUID, author, content string) (*generated.BlogComment, error)
	FindByID(ctx context.Context, id uuid.UUID) (*generated.BlogComment, error)
	FindAllByPostID(ctx context.Context, postID uuid.UUID, limit, offset int32) ([]generated.BlogComment, error)
	CountByPostID(ctx context.Context, postID uuid.UUID) (int64, error)
	Update(ctx context.Context, id uuid.UUID, author, content *string) (*generated.BlogComment, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// PostgresPostRepository provides PostgreSQL-based post storage using sqlc.
type PostgresPostRepository struct {
	queries *generated.Queries
}

// Compile-time interface check.
var _ PostRepository = (*PostgresPostRepository)(nil)

// NewPostgresPostRepository creates a new PostgreSQL post repository.
func NewPostgresPostRepository(db generated.DBTX) *PostgresPostRepository {
	return &PostgresPostRepository{
		queries: generated.New(db),
	}
}

// Create saves a new post to the database.
func (r *PostgresPostRepository) Create(ctx context.Context, title, content, slug string, published bool) (*generated.BlogPost, error) {
	post, err := r.queries.CreatePost(ctx, generated.CreatePostParams{
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
	return &post, nil
}

// FindByID retrieves a post by ID.
func (r *PostgresPostRepository) FindByID(ctx context.Context, id uuid.UUID) (*generated.BlogPost, error) {
	post, err := r.queries.GetPostByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}
	return &post, nil
}

// FindBySlug retrieves a post by slug.
func (r *PostgresPostRepository) FindBySlug(ctx context.Context, slug string) (*generated.BlogPost, error) {
	post, err := r.queries.GetPostBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}
	return &post, nil
}

// FindAll retrieves paginated posts.
func (r *PostgresPostRepository) FindAll(ctx context.Context, limit, offset int32) ([]generated.BlogPost, error) {
	return r.queries.ListPosts(ctx, generated.ListPostsParams{
		Limit:  limit,
		Offset: offset,
	})
}

// FindAllByPublished retrieves paginated posts filtered by published status.
func (r *PostgresPostRepository) FindAllByPublished(ctx context.Context, published bool, limit, offset int32) ([]generated.BlogPost, error) {
	return r.queries.ListPostsByPublished(ctx, generated.ListPostsByPublishedParams{
		Published: published,
		Limit:     limit,
		Offset:    offset,
	})
}

// Count returns the total number of posts.
func (r *PostgresPostRepository) Count(ctx context.Context) (int64, error) {
	return r.queries.CountPosts(ctx)
}

// CountByPublished returns the count of posts by published status.
func (r *PostgresPostRepository) CountByPublished(ctx context.Context, published bool) (int64, error) {
	return r.queries.CountPostsByPublished(ctx, published)
}

// Update updates an existing post.
func (r *PostgresPostRepository) Update(ctx context.Context, id uuid.UUID, title, content, slug *string, published *bool) (*generated.BlogPost, error) {
	params := generated.UpdatePostParams{
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

	post, err := r.queries.UpdatePost(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPostNotFound
		}
		if isPgDuplicateKeyError(err) {
			return nil, ErrDuplicateSlug
		}
		return nil, err
	}
	return &post, nil
}

// Publish sets a post's published status to true.
func (r *PostgresPostRepository) Publish(ctx context.Context, id uuid.UUID) (*generated.BlogPost, error) {
	post, err := r.queries.PublishPost(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}
	return &post, nil
}

// Delete removes a post by ID.
func (r *PostgresPostRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}
	return r.queries.DeletePost(ctx, pgtype.UUID{Bytes: id, Valid: true})
}

// PostgresCommentRepository provides PostgreSQL-based comment storage using sqlc.
type PostgresCommentRepository struct {
	queries *generated.Queries
}

// Compile-time interface check.
var _ CommentRepository = (*PostgresCommentRepository)(nil)

// NewPostgresCommentRepository creates a new PostgreSQL comment repository.
func NewPostgresCommentRepository(db generated.DBTX) *PostgresCommentRepository {
	return &PostgresCommentRepository{
		queries: generated.New(db),
	}
}

// Create saves a new comment to the database.
func (r *PostgresCommentRepository) Create(ctx context.Context, postID uuid.UUID, author, content string) (*generated.BlogComment, error) {
	comment, err := r.queries.CreateComment(ctx, generated.CreateCommentParams{
		PostID:  pgtype.UUID{Bytes: postID, Valid: true},
		Author:  author,
		Content: content,
	})
	if err != nil {
		if isPgForeignKeyError(err) {
			return nil, ErrPostDoesNotExist
		}
		return nil, err
	}
	return &comment, nil
}

// FindByID retrieves a comment by ID.
func (r *PostgresCommentRepository) FindByID(ctx context.Context, id uuid.UUID) (*generated.BlogComment, error) {
	comment, err := r.queries.GetCommentByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}
	return &comment, nil
}

// FindAllByPostID retrieves paginated comments for a post.
func (r *PostgresCommentRepository) FindAllByPostID(ctx context.Context, postID uuid.UUID, limit, offset int32) ([]generated.BlogComment, error) {
	return r.queries.ListCommentsByPostID(ctx, generated.ListCommentsByPostIDParams{
		PostID: pgtype.UUID{Bytes: postID, Valid: true},
		Limit:  limit,
		Offset: offset,
	})
}

// CountByPostID returns the count of comments for a post.
func (r *PostgresCommentRepository) CountByPostID(ctx context.Context, postID uuid.UUID) (int64, error) {
	return r.queries.CountCommentsByPostID(ctx, pgtype.UUID{Bytes: postID, Valid: true})
}

// Update updates an existing comment.
func (r *PostgresCommentRepository) Update(ctx context.Context, id uuid.UUID, author, content *string) (*generated.BlogComment, error) {
	params := generated.UpdateCommentParams{
		ID: pgtype.UUID{Bytes: id, Valid: true},
	}
	if author != nil {
		params.Author = pgtype.Text{String: *author, Valid: true}
	}
	if content != nil {
		params.Content = pgtype.Text{String: *content, Valid: true}
	}

	comment, err := r.queries.UpdateComment(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCommentNotFound
		}
		return nil, err
	}
	return &comment, nil
}

// Delete removes a comment by ID.
func (r *PostgresCommentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}
	return r.queries.DeleteComment(ctx, pgtype.UUID{Bytes: id, Valid: true})
}

// PostgreSQL error codes.
const (
	pgErrUniqueViolation     = "23505"
	pgErrForeignKeyViolation = "23503"
)

// isPgDuplicateKeyError checks if error is a PostgreSQL unique violation.
func isPgDuplicateKeyError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgErrUniqueViolation
	}
	return false
}

// isPgForeignKeyError checks if error is a PostgreSQL foreign key violation.
func isPgForeignKeyError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgErrForeignKeyViolation
	}
	return false
}
