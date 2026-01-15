-- name: CreateArticle :one
INSERT INTO articles (title, content, slug, published)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetArticleByID :one
SELECT * FROM articles
WHERE id = $1;

-- name: GetArticleBySlug :one
SELECT * FROM articles
WHERE slug = $1;

-- name: ListArticles :many
SELECT * FROM articles
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListArticlesByPublished :many
SELECT * FROM articles
WHERE published = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountArticles :one
SELECT COUNT(*) FROM articles;

-- name: CountArticlesByPublished :one
SELECT COUNT(*) FROM articles
WHERE published = $1;

-- name: UpdateArticle :one
UPDATE articles
SET title = COALESCE(sqlc.narg('title'), title),
    content = COALESCE(sqlc.narg('content'), content),
    slug = COALESCE(sqlc.narg('slug'), slug),
    published = COALESCE(sqlc.narg('published'), published),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: PublishArticle :one
UPDATE articles
SET published = TRUE,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteArticle :exec
DELETE FROM articles
WHERE id = $1;
