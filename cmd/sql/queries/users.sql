-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1
)
RETURNING *;

-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;

-- name: GetAllChirps :many
SELECT c.id, c.created_at, c.updated_at, c.body, u.id AS user_id
FROM chirps c
JOIN users u ON c.user_id = u.id
ORDER BY c.created_at ASC
LIMIT $1 OFFSET $2;

-- name: Reset :exec
TRUNCATE TABLE chirps, users RESTART IDENTITY CASCADE;