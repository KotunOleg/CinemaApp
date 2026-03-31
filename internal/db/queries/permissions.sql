-- name: ListPermissions :many
SELECT * FROM permissions ORDER BY permission_id;

-- name: GetPermission :one
SELECT * FROM permissions WHERE permission_id = $1;
