-- name: GetShowtime :one
SELECT
    s.*,
    m.title AS movie_title,
    c.name AS cinema_name
FROM showtimes s
JOIN movies m ON m.movie_id = s.movie_id
JOIN cinemas c ON c.cinema_id = s.cinema_id
WHERE s.showtime_id = $1;

-- name: ListShowtimes :many
SELECT
    s.*,
    m.title AS movie_title,
    c.name AS cinema_name
FROM showtimes s
JOIN movies m ON m.movie_id = s.movie_id
JOIN cinemas c ON c.cinema_id = s.cinema_id
ORDER BY s.start_time DESC
LIMIT $1 OFFSET $2;

-- name: CreateShowtime :one
INSERT INTO showtimes (movie_id, cinema_id, start_time, price, duration)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateShowtime :one
UPDATE showtimes
SET movie_id = $2, cinema_id = $3, start_time = $4, price = $5, duration = $6
WHERE showtime_id = $1
RETURNING *;

-- name: DeleteShowtime :exec
DELETE FROM showtimes WHERE showtime_id = $1;
