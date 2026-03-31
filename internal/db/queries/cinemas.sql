-- name: GetCinema :one
SELECT * FROM cinemas WHERE cinema_id = $1;

-- name: ListCinemas :many
SELECT * FROM cinemas ORDER BY cinema_id LIMIT $1 OFFSET $2;

-- name: CreateCinema :one
INSERT INTO cinemas (name, address, location_coordinates)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateCinema :one
UPDATE cinemas
SET name = $2, address = $3, location_coordinates = $4
WHERE cinema_id = $1
RETURNING *;

-- name: DeleteCinema :exec
DELETE FROM cinemas WHERE cinema_id = $1;
