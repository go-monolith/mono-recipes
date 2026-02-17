-- =====================
-- Post Queries
-- =====================

-- name: CreatePost :one
INSERT INTO blog.posts (title, content, slug, published)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetPostByID :one
SELECT * FROM blog.posts
WHERE id = $1;

-- name: GetPostBySlug :one
SELECT * FROM blog.posts
WHERE slug = $1;

-- name: ListPosts :many
SELECT * FROM blog.posts
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListPostsByPublished :many
SELECT * FROM blog.posts
WHERE published = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountPosts :one
SELECT COUNT(*) FROM blog.posts;

-- name: CountPostsByPublished :one
SELECT COUNT(*) FROM blog.posts
WHERE published = $1;

-- name: UpdatePost :one
UPDATE blog.posts
SET title = COALESCE(sqlc.narg('title'), title),
    content = COALESCE(sqlc.narg('content'), content),
    slug = COALESCE(sqlc.narg('slug'), slug),
    published = COALESCE(sqlc.narg('published'), published),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: PublishPost :one
UPDATE blog.posts
SET published = TRUE,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeletePost :exec
DELETE FROM blog.posts
WHERE id = $1;

-- =====================
-- Comment Queries
-- =====================

-- name: CreateComment :one
INSERT INTO blog.comments (post_id, author, content)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetCommentByID :one
SELECT * FROM blog.comments
WHERE id = $1;

-- name: ListCommentsByPostID :many
SELECT * FROM blog.comments
WHERE post_id = $1
ORDER BY created_at ASC
LIMIT $2 OFFSET $3;

-- name: CountCommentsByPostID :one
SELECT COUNT(*) FROM blog.comments
WHERE post_id = $1;

-- name: UpdateComment :one
UPDATE blog.comments
SET author = COALESCE(sqlc.narg('author'), author),
    content = COALESCE(sqlc.narg('content'), content),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteComment :exec
DELETE FROM blog.comments
WHERE id = $1;

-- name: DeleteCommentsByPostID :exec
DELETE FROM blog.comments
WHERE post_id = $1;
