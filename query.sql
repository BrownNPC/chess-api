-- name: CreateUser :one
INSERT INTO users (username, password_hash)
VALUES (?, ?)
RETURNING *;

-- name: GetUserById :one
SELECT * FROM users
WHERE uid = ?;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = ?;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: UpdateUserPassword :one
UPDATE users
SET password_hash= ?
WHERE uid = ?
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE uid = ?;

-- name: StoreGame :one
INSERT INTO games (white_uid, black_uid, result, moves, finished_at)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetGameById :one
SELECT * FROM games
WHERE Id = ?;

-- name: ListGames :many
SELECT * FROM games
ORDER BY finished_at DESC
LIMIT ? OFFSET ?;

-- name: ListGamesByPlayer :many
SELECT * FROM games
WHERE white_uid = ? OR black_uid = ?
ORDER BY finished_at DESC
LIMIT ? OFFSET ?;

-- name: DeleteGame :exec
DELETE FROM games
WHERE id = ?;

