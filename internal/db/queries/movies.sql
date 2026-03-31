-- name: GetMovie :one
SELECT * FROM movies WHERE movie_id = $1;

-- name: ListMovies :many
SELECT * FROM movies ORDER BY movie_id LIMIT $1 OFFSET $2;

-- name: CreateMovie :one
INSERT INTO movies (title, description, genre, trailer_url)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateMovie :one
UPDATE movies
SET title = $2, description = $3, genre = $4, trailer_url = $5
WHERE movie_id = $1
RETURNING *;

-- name: DeleteMovie :exec
DELETE FROM movies WHERE movie_id = $1;
