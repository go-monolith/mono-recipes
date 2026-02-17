package blog

import "time"

// === Post Types ===

// CreatePostRequest is the request for creating a post.
type CreatePostRequest struct {
	Title     string `json:"title"`
	Content   string `json:"content"`
	Slug      string `json:"slug"`
	Published bool   `json:"published"`
}

// GetPostRequest is the request for getting a post.
// Supports lookup by ID or slug.
type GetPostRequest struct {
	ID   string `json:"id,omitempty"`
	Slug string `json:"slug,omitempty"`
}

// ListPostsRequest is the request for listing posts with pagination.
type ListPostsRequest struct {
	Limit     int32 `json:"limit"`               // Default: 10, Max: 100
	Offset    int32 `json:"offset"`              // Default: 0
	Published *bool `json:"published,omitempty"` // Optional filter: nil=all, true=published, false=drafts
}

// UpdatePostRequest is the request for updating a post.
type UpdatePostRequest struct {
	ID        string  `json:"id"`
	Title     *string `json:"title,omitempty"`
	Content   *string `json:"content,omitempty"`
	Slug      *string `json:"slug,omitempty"`
	Published *bool   `json:"published,omitempty"`
}

// DeletePostRequest is the request for deleting a post.
type DeletePostRequest struct {
	ID string `json:"id"`
}

// PublishPostRequest is the request for publishing a draft post.
type PublishPostRequest struct {
	ID string `json:"id"`
}

// PostResponse represents a post in API responses.
type PostResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Slug      string    `json:"slug"`
	Published bool      `json:"published"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListPostsResponse is the response containing paginated posts.
type ListPostsResponse struct {
	Posts  []PostResponse `json:"posts"`
	Total  int64          `json:"total"`
	Limit  int32          `json:"limit"`
	Offset int32          `json:"offset"`
}

// DeletePostResponse is the response after deleting a post.
type DeletePostResponse struct {
	Deleted bool   `json:"deleted"`
	ID      string `json:"id"`
}

// === Comment Types ===

// CreateCommentRequest is the request for creating a comment on a post.
type CreateCommentRequest struct {
	PostID  string `json:"post_id"`
	Author  string `json:"author"`
	Content string `json:"content"`
}

// GetCommentRequest is the request for getting a comment by ID.
type GetCommentRequest struct {
	ID string `json:"id"`
}

// ListCommentsRequest is the request for listing comments on a post with pagination.
type ListCommentsRequest struct {
	PostID string `json:"post_id"`
	Limit  int32  `json:"limit"`  // Default: 10, Max: 100
	Offset int32  `json:"offset"` // Default: 0
}

// UpdateCommentRequest is the request for updating a comment.
type UpdateCommentRequest struct {
	ID      string  `json:"id"`
	Author  *string `json:"author,omitempty"`
	Content *string `json:"content,omitempty"`
}

// DeleteCommentRequest is the request for deleting a comment.
type DeleteCommentRequest struct {
	ID string `json:"id"`
}

// CommentResponse represents a comment in API responses.
type CommentResponse struct {
	ID        string    `json:"id"`
	PostID    string    `json:"post_id"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListCommentsResponse is the response containing paginated comments.
type ListCommentsResponse struct {
	Comments []CommentResponse `json:"comments"`
	Total    int64             `json:"total"`
	Limit    int32             `json:"limit"`
	Offset   int32             `json:"offset"`
}

// DeleteCommentResponse is the response after deleting a comment.
type DeleteCommentResponse struct {
	Deleted bool   `json:"deleted"`
	ID      string `json:"id"`
}
