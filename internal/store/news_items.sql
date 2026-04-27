-- name: CreateNewsItem :one
INSERT INTO news_items (
    issue_id, source, title, url, author, score, summary, position, raw_json
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetNewsItem :one
SELECT * FROM news_items WHERE id = ? LIMIT 1;

-- name: ListNewsItemsByIssue :many
SELECT * FROM news_items
WHERE issue_id = ?
ORDER BY position ASC;

-- name: DeleteNewsItemsByIssue :exec
DELETE FROM news_items WHERE issue_id = ?;
