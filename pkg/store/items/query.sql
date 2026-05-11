-- name: ItemCreate :one
INSERT INTO items (
    issue_id, source, tag, title, url,
    author_name, author_username, author_avatar_url, author_profile_url,
    score, summary, position
) VALUES (
    ?, ?, ?, ?, ?,
    ?, ?, ?, ?,
    ?, ?, ?
)
RETURNING *;

-- name: ItemByID :one
SELECT * FROM items WHERE id = ? LIMIT 1;

-- name: ItemListByIssue :many
SELECT * FROM items
WHERE issue_id = ?
ORDER BY position ASC;

-- name: ItemDeleteByIssue :exec
DELETE FROM items WHERE issue_id = ?;
