-- name: GetTicket :one
SELECT
    t.*,
    u.full_name AS user_name,
    u.email AS user_email,
    m.title AS movie_title,
    c.name AS cinema_name,
    s.start_time
FROM tickets t
JOIN users u ON u.user_id = t.user_id
JOIN showtimes s ON s.showtime_id = t.showtime_id
JOIN movies m ON m.movie_id = s.movie_id
JOIN cinemas c ON c.cinema_id = s.cinema_id
WHERE t.ticket_id = $1;

-- name: ListTickets :many
SELECT
    t.*,
    u.full_name AS user_name,
    u.email AS user_email,
    m.title AS movie_title,
    c.name AS cinema_name,
    s.start_time
FROM tickets t
JOIN users u ON u.user_id = t.user_id
JOIN showtimes s ON s.showtime_id = t.showtime_id
JOIN movies m ON m.movie_id = s.movie_id
JOIN cinemas c ON c.cinema_id = s.cinema_id
ORDER BY s.start_time DESC
LIMIT $1 OFFSET $2;

-- name: CreateTicket :one
INSERT INTO tickets (showtime_id, user_id, seat_number, payment_status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateTicketStatus :one
UPDATE tickets
SET payment_status = $2
WHERE ticket_id = $1
RETURNING *;

-- name: DeleteTicket :exec
DELETE FROM tickets WHERE ticket_id = $1;
