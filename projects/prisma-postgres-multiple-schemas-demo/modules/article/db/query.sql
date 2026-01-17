-- name: CreateArticle :one
INSERT INTO article_module.articles (title, content, slug, published)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetArticleByID :one
SELECT * FROM article_module.articles
WHERE id = $1;

-- name: GetArticleBySlug :one
SELECT * FROM article_module.articles
WHERE slug = $1;

-- name: ListArticles :many
SELECT * FROM article_module.articles
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListArticlesByPublished :many
SELECT * FROM article_module.articles
WHERE published = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountArticles :one
SELECT COUNT(*) FROM article_module.articles;

-- name: CountArticlesByPublished :one
SELECT COUNT(*) FROM article_module.articles
WHERE published = $1;

-- name: UpdateArticle :one
UPDATE article_module.articles
SET title = COALESCE(sqlc.narg('title'), title),
    content = COALESCE(sqlc.narg('content'), content),
    slug = COALESCE(sqlc.narg('slug'), slug),
    published = COALESCE(sqlc.narg('published'), published),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: PublishArticle :one
UPDATE article_module.articles
SET published = TRUE,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteArticle :exec
DELETE FROM article_module.articles
WHERE id = $1;
