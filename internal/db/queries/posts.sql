-- name: GetPost :one
SELECT * FROM posts WHERE id = $1;

-- name: ListPosts :many
SELECT * FROM posts 
WHERE published = COALESCE(sqlc.narg('published'), published)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListPostsByUser :many
SELECT * FROM posts 
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: CreatePost :one
INSERT INTO posts (user_id, title, content, published)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdatePost :one
UPDATE posts 
SET title = COALESCE(sqlc.narg('title'), title),
    content = COALESCE(sqlc.narg('content'), content),
    published = COALESCE(sqlc.narg('published'), published),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: PublishPost :exec
UPDATE posts SET published = true, updated_at = CURRENT_TIMESTAMP WHERE id = $1;

-- name: DeletePost :exec
DELETE FROM posts WHERE id = $1;

-- name: GetPostWithAuthor :one
SELECT 
    p.id, p.title, p.content, p.published, p.created_at,
    u.id as author_id, u.name as author_name, u.email as author_email
FROM posts p
JOIN users u ON p.user_id = u.id
WHERE p.id = $1;

-- name: ListPostsWithAuthors :many
SELECT 
    p.id, p.title, p.content, p.published, p.created_at,
    u.id as author_id, u.name as author_name
FROM posts p
JOIN users u ON p.user_id = u.id
WHERE p.published = true
ORDER BY p.created_at DESC
LIMIT $1 OFFSET $2;
