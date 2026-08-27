-- name: CreateUser :one
INSERT INTO users (email, password_hash, full_name, phone, location, bio)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UpdateUser :one
UPDATE users
SET full_name = $2, phone = $3, location = $4, bio = $5, avatar_url = $6, updated_at = now()
WHERE id = $1
RETURNING *;
