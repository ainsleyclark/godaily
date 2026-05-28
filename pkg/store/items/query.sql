-- name: ItemCreate :one
INSERT INTO items (
    issue_id, source, tag, title, url, original_url,
    author_name, author_username, author_avatar_url, author_profile_url,
    score, summary, position, published
) VALUES (
    ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?,
    ?, ?, ?, ?
)
ON CONFLICT (url, tag) DO UPDATE SET
    issue_id = COALESCE(excluded.issue_id, items.issue_id),
    position = excluded.position
RETURNING *;

-- name: ItemByID :one
SELECT * FROM items WHERE id = ? LIMIT 1;

-- name: ItemFindByURLInIssue :one
SELECT id FROM items
WHERE issue_id = @issue_id
  AND (url = @url OR (original_url IS NOT NULL AND original_url = @url))
LIMIT 1;

-- name: ItemListByIssue :many
SELECT * FROM items
WHERE issue_id = ?
ORDER BY position ASC;

-- name: ItemDeleteByIssue :exec
DELETE FROM items WHERE issue_id = ?;

-- name: ItemCount :one
SELECT COUNT(*) FROM items;

-- name: ItemSourceCounts :many
SELECT source, COUNT(*) AS count
FROM items
GROUP BY source
ORDER BY count DESC;

-- name: ItemTagCounts :many
SELECT tag, COUNT(*) AS count
FROM items
GROUP BY tag
ORDER BY count DESC;
