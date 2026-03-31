-- name: GetReview :one
SELECT * FROM reviews WHERE review_id = $1;

-- name: ListReviews :many
SELECT
    r.*,
    u.full_name AS user_name,
    m.title AS movie_title
FROM reviews r
JOIN users u ON u.user_id = r.user_id
JOIN movies m ON m.movie_id = r.movie_id
ORDER BY r.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CreateReview :one
INSERT INTO reviews (user_id, movie_id, rating, content)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateReview :one
UPDATE reviews
SET rating = $2, content = $3
WHERE review_id = $1
RETURNING *;

-- name: DeleteReview :exec
DELETE FROM reviews WHERE review_id = $1;
