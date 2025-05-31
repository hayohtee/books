-- name: CreateBook :one
INSERT INTO books(user_id, name)
VALUES ($1, $2)
RETURNING id, created_at, updated_at;

-- name: ListBookForUser :many
SELECT count(*) OVER AS total_records,
       id,
       user_id,
       name,
       created_at,
       updated_at
FROM books
WHERE user_id = $1
  AND (to_tsvector('simple', name) @@ plainto_tsquery('simple', $2) OR $2 = '')
ORDER BY name
LIMIT $3 OFFSET $4;

-- name: GetBook :one
SELECT *
FROM books
WHERE id = $1;

-- name: UpdateBook :exec
UPDATE books
SET name       = $1,
    updated_at = now(),
    version    = version + 1
WHERE id = $2
  AND version = $4;

-- name: DeleteBook :exec
DELETE
FROM books
WHERE id = $1
  AND user_id = $2;