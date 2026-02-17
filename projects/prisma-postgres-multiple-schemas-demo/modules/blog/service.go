package blog

import (
	"context"
	"errors"

	"github.com/example/prisma-postgres-demo/modules/blog/db/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Post validation errors.
var (
	ErrPostTitleRequired    = errors.New("title is required")
	ErrPostContentRequired  = errors.New("content is required")
	ErrPostSlugRequired     = errors.New("slug is required")
	ErrPostIDRequired       = errors.New("id is required")
	ErrPostIDInvalid        = errors.New("id is not a valid UUID")
	ErrPostIDOrSlugRequired = errors.New("id or slug is required")
)

// Comment validation errors.
var (
	ErrCommentPostIDRequired  = errors.New("post_id is required")
	ErrCommentPostIDInvalid   = errors.New("post_id is not a valid UUID")
	ErrCommentAuthorRequired  = errors.New("author is required")
	ErrCommentContentRequired = errors.New("content is required")
	ErrCommentIDRequired      = errors.New("id is required")
	ErrCommentIDInvalid       = errors.New("id is not a valid UUID")
)

// PostService defines the interface for post business operations.
type PostService interface {
	Create(ctx context.Context, req CreatePostRequest) (PostResponse, error)
	Get(ctx context.Context, req GetPostRequest) (PostResponse, error)
	List(ctx context.Context, req ListPostsRequest) (ListPostsResponse, error)
	Update(ctx context.Context, req UpdatePostRequest) (PostResponse, error)
	Delete(ctx context.Context, req DeletePostRequest) (DeletePostResponse, error)
	Publish(ctx context.Context, req PublishPostRequest) (PostResponse, error)
}

// CommentService defines the interface for comment business operations.
type CommentService interface {
	Create(ctx context.Context, req CreateCommentRequest) (CommentResponse, error)
	Get(ctx context.Context, req GetCommentRequest) (CommentResponse, error)
	List(ctx context.Context, req ListCommentsRequest) (ListCommentsResponse, error)
	Update(ctx context.Context, req UpdateCommentRequest) (CommentResponse, error)
	Delete(ctx context.Context, req DeleteCommentRequest) (DeleteCommentResponse, error)
}

// PostServiceImpl implements PostService using PostRepository.
type PostServiceImpl struct {
	repo PostRepository
}

// Compile-time interface check.
var _ PostService = (*PostServiceImpl)(nil)

// NewPostService creates a new PostService with the given repository.
func NewPostService(repo PostRepository) PostService {
	return &PostServiceImpl{repo: repo}
}

// Create handles the post creation request.
func (s *PostServiceImpl) Create(ctx context.Context, req CreatePostRequest) (PostResponse, error) {
	if req.Title == "" {
		return PostResponse{}, ErrPostTitleRequired
	}
	if req.Content == "" {
		return PostResponse{}, ErrPostContentRequired
	}
	if req.Slug == "" {
		return PostResponse{}, ErrPostSlugRequired
	}

	post, err := s.repo.Create(ctx, req.Title, req.Content, req.Slug, req.Published)
	if err != nil {
		return PostResponse{}, err
	}

	return toPostResponse(post), nil
}

// Get handles the post retrieval request (by ID or slug).
func (s *PostServiceImpl) Get(ctx context.Context, req GetPostRequest) (PostResponse, error) {
	if req.ID == "" && req.Slug == "" {
		return PostResponse{}, ErrPostIDOrSlugRequired
	}

	var post *generated.BlogPost
	var err error

	if req.ID != "" {
		id, parseErr := uuid.Parse(req.ID)
		if parseErr != nil {
			return PostResponse{}, ErrPostIDInvalid
		}
		post, err = s.repo.FindByID(ctx, id)
	} else {
		post, err = s.repo.FindBySlug(ctx, req.Slug)
	}

	if err != nil {
		return PostResponse{}, err
	}

	return toPostResponse(post), nil
}

// List handles the post list request with pagination and optional published filter.
func (s *PostServiceImpl) List(ctx context.Context, req ListPostsRequest) (ListPostsResponse, error) {
	limit := clampLimit(req.Limit)
	offset := clampOffset(req.Offset)

	var posts []generated.BlogPost
	var total int64
	var err error

	if req.Published != nil {
		posts, err = s.repo.FindAllByPublished(ctx, *req.Published, limit, offset)
		if err != nil {
			return ListPostsResponse{}, err
		}
		total, err = s.repo.CountByPublished(ctx, *req.Published)
	} else {
		posts, err = s.repo.FindAll(ctx, limit, offset)
		if err != nil {
			return ListPostsResponse{}, err
		}
		total, err = s.repo.Count(ctx)
	}

	if err != nil {
		return ListPostsResponse{}, err
	}

	postResponses := make([]PostResponse, len(posts))
	for i := range posts {
		postResponses[i] = toPostResponse(&posts[i])
	}

	return ListPostsResponse{
		Posts:  postResponses,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// Update handles the post update request.
func (s *PostServiceImpl) Update(ctx context.Context, req UpdatePostRequest) (PostResponse, error) {
	if req.ID == "" {
		return PostResponse{}, ErrPostIDRequired
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return PostResponse{}, ErrPostIDInvalid
	}

	if req.Title != nil && *req.Title == "" {
		return PostResponse{}, ErrPostTitleRequired
	}
	if req.Content != nil && *req.Content == "" {
		return PostResponse{}, ErrPostContentRequired
	}
	if req.Slug != nil && *req.Slug == "" {
		return PostResponse{}, ErrPostSlugRequired
	}

	post, err := s.repo.Update(ctx, id, req.Title, req.Content, req.Slug, req.Published)
	if err != nil {
		return PostResponse{}, err
	}

	return toPostResponse(post), nil
}

// Delete handles the post deletion request.
func (s *PostServiceImpl) Delete(ctx context.Context, req DeletePostRequest) (DeletePostResponse, error) {
	if req.ID == "" {
		return DeletePostResponse{}, ErrPostIDRequired
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return DeletePostResponse{}, ErrPostIDInvalid
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return DeletePostResponse{ID: req.ID}, err
	}

	return DeletePostResponse{Deleted: true, ID: req.ID}, nil
}

// Publish handles the post publish request.
func (s *PostServiceImpl) Publish(ctx context.Context, req PublishPostRequest) (PostResponse, error) {
	if req.ID == "" {
		return PostResponse{}, ErrPostIDRequired
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return PostResponse{}, ErrPostIDInvalid
	}

	post, err := s.repo.Publish(ctx, id)
	if err != nil {
		return PostResponse{}, err
	}

	return toPostResponse(post), nil
}

// CommentServiceImpl implements CommentService using CommentRepository and PostRepository.
type CommentServiceImpl struct {
	commentRepo CommentRepository
	postRepo    PostRepository
}

// Compile-time interface check.
var _ CommentService = (*CommentServiceImpl)(nil)

// NewCommentService creates a new CommentService with the given repositories.
func NewCommentService(commentRepo CommentRepository, postRepo PostRepository) CommentService {
	return &CommentServiceImpl{
		commentRepo: commentRepo,
		postRepo:    postRepo,
	}
}

// Create handles the comment creation request.
func (s *CommentServiceImpl) Create(ctx context.Context, req CreateCommentRequest) (CommentResponse, error) {
	if req.PostID == "" {
		return CommentResponse{}, ErrCommentPostIDRequired
	}
	if req.Author == "" {
		return CommentResponse{}, ErrCommentAuthorRequired
	}
	if req.Content == "" {
		return CommentResponse{}, ErrCommentContentRequired
	}

	postID, err := uuid.Parse(req.PostID)
	if err != nil {
		return CommentResponse{}, ErrCommentPostIDInvalid
	}

	// Repository handles foreign key constraint violation and returns ErrPostDoesNotExist
	comment, err := s.commentRepo.Create(ctx, postID, req.Author, req.Content)
	if err != nil {
		return CommentResponse{}, err
	}

	return toCommentResponse(comment), nil
}

// Get handles the comment retrieval request.
func (s *CommentServiceImpl) Get(ctx context.Context, req GetCommentRequest) (CommentResponse, error) {
	if req.ID == "" {
		return CommentResponse{}, ErrCommentIDRequired
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return CommentResponse{}, ErrCommentIDInvalid
	}

	comment, err := s.commentRepo.FindByID(ctx, id)
	if err != nil {
		return CommentResponse{}, err
	}

	return toCommentResponse(comment), nil
}

// List handles the comment list request with pagination.
func (s *CommentServiceImpl) List(ctx context.Context, req ListCommentsRequest) (ListCommentsResponse, error) {
	if req.PostID == "" {
		return ListCommentsResponse{}, ErrCommentPostIDRequired
	}

	postID, err := uuid.Parse(req.PostID)
	if err != nil {
		return ListCommentsResponse{}, ErrCommentPostIDInvalid
	}

	limit := clampLimit(req.Limit)
	offset := clampOffset(req.Offset)

	comments, err := s.commentRepo.FindAllByPostID(ctx, postID, limit, offset)
	if err != nil {
		return ListCommentsResponse{}, err
	}

	total, err := s.commentRepo.CountByPostID(ctx, postID)
	if err != nil {
		return ListCommentsResponse{}, err
	}

	commentResponses := make([]CommentResponse, len(comments))
	for i := range comments {
		commentResponses[i] = toCommentResponse(&comments[i])
	}

	return ListCommentsResponse{
		Comments: commentResponses,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}, nil
}

// Update handles the comment update request.
func (s *CommentServiceImpl) Update(ctx context.Context, req UpdateCommentRequest) (CommentResponse, error) {
	if req.ID == "" {
		return CommentResponse{}, ErrCommentIDRequired
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return CommentResponse{}, ErrCommentIDInvalid
	}

	if req.Author != nil && *req.Author == "" {
		return CommentResponse{}, ErrCommentAuthorRequired
	}
	if req.Content != nil && *req.Content == "" {
		return CommentResponse{}, ErrCommentContentRequired
	}

	comment, err := s.commentRepo.Update(ctx, id, req.Author, req.Content)
	if err != nil {
		return CommentResponse{}, err
	}

	return toCommentResponse(comment), nil
}

// Delete handles the comment deletion request.
func (s *CommentServiceImpl) Delete(ctx context.Context, req DeleteCommentRequest) (DeleteCommentResponse, error) {
	if req.ID == "" {
		return DeleteCommentResponse{}, ErrCommentIDRequired
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return DeleteCommentResponse{}, ErrCommentIDInvalid
	}

	if err := s.commentRepo.Delete(ctx, id); err != nil {
		return DeleteCommentResponse{ID: req.ID}, err
	}

	return DeleteCommentResponse{Deleted: true, ID: req.ID}, nil
}

// Pagination constants.
const (
	defaultLimit = 10
	maxLimit     = 100
)

// clampLimit ensures limit is within valid bounds (1-100, default 10).
func clampLimit(limit int32) int32 {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
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

func toPostResponse(post *generated.BlogPost) PostResponse {
	return PostResponse{
		ID:        uuidToString(post.ID),
		Title:     post.Title,
		Content:   post.Content,
		Slug:      post.Slug,
		Published: post.Published,
		CreatedAt: post.CreatedAt.Time,
		UpdatedAt: post.UpdatedAt.Time,
	}
}

func toCommentResponse(comment *generated.BlogComment) CommentResponse {
	return CommentResponse{
		ID:        uuidToString(comment.ID),
		PostID:    uuidToString(comment.PostID),
		Author:    comment.Author,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt.Time,
		UpdatedAt: comment.UpdatedAt.Time,
	}
}

func uuidToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return uuid.UUID(id.Bytes).String()
}
