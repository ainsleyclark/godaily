-- name: CreateNewsItem :one
INSERT INTO items (
    issue_id, source, title, url,
    author_name, author_username, author_avatar_url, author_profile_url,
    score, summary, position, raw_json
) VALUES (
    ?, ?, ?, ?,
    ?, ?, ?, ?,
    ?, ?, ?, ?
)
RETURNING *;

-- name: GetNewsItem :one
SELECT * FROM items WHERE id = ? LIMIT 1;

-- name: ListNewsItemsByIssue :many
SELECT * FROM items
WHERE issue_id = ?
ORDER BY position ASC;

-- name: DeleteNewsItemsByIssue :exec
DELETE FROM items WHERE issue_id = ?;
