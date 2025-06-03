-- name: CreateUser :one
INSERT INTO users(first_name, last_name, email, password_hash)
VALUES ($1, $2, $3, $4)
RETURNING id, email_verified, created_at;

-- name: FindUserByEmail :one
SELECT *
FROM users
WHERE email = $1;

-- name: VerifyUserEmail :exec
UPDATE users
SET email_verified = true
WHERE id = $1;

-- name: GetUser :one
SELECT *
FROM users
WHERE id = $1;