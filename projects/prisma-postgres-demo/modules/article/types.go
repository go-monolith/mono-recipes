package article

import "time"

// CreateArticleRequest is the request for creating an article.
type CreateArticleRequest struct {
	Title     string `json:"title"`
	Content   string `json:"content"`
	Slug      string `json:"slug"`
	Published bool   `json:"published"`
}

// GetArticleRequest is the request for getting an article.
// Supports lookup by ID or slug.
type GetArticleRequest struct {
	ID   string `json:"id,omitempty"`
	Slug string `json:"slug,omitempty"`
}

// ListArticlesRequest is the request for listing articles with pagination.
type ListArticlesRequest struct {
	Limit     int32 `json:"limit"`               // Default: 10, Max: 100
	Offset    int32 `json:"offset"`              // Default: 0
	Published *bool `json:"published,omitempty"` // Optional filter: nil=all, true=published, false=drafts
}

// UpdateArticleRequest is the request for updating an article.
type UpdateArticleRequest struct {
	ID        string  `json:"id"`
	Title     *string `json:"title,omitempty"`
	Content   *string `json:"content,omitempty"`
	Slug      *string `json:"slug,omitempty"`
	Published *bool   `json:"published,omitempty"`
}

// DeleteArticleRequest is the request for deleting an article.
type DeleteArticleRequest struct {
	ID string `json:"id"`
}

// PublishArticleRequest is the request for publishing a draft article.
type PublishArticleRequest struct {
	ID string `json:"id"`
}

// ArticleResponse represents an article in API responses.
type ArticleResponse struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Slug      string    `json:"slug"`
	Published bool      `json:"published"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListArticlesResponse is the response containing paginated articles.
type ListArticlesResponse struct {
	Articles []ArticleResponse `json:"articles"`
	Total    int64             `json:"total"`
	Limit    int32             `json:"limit"`
	Offset   int32             `json:"offset"`
}

// DeleteArticleResponse is the response after deleting an article.
type DeleteArticleResponse struct {
	Deleted bool   `json:"deleted"`
	ID      string `json:"id"`
}
