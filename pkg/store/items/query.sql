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

-- name: ItemUnlinkFromIssue :execrows
UPDATE items
SET issue_id = NULL, position = 0
WHERE items.id = sqlc.arg('item_id')
  AND items.issue_id = sqlc.arg('issue_id')
  AND EXISTS (
      SELECT 1 FROM issues
      WHERE issues.id = items.issue_id AND issues.status = 'draft'
  );

-- name: ItemUpdatePosition :execrows
UPDATE items
SET position = sqlc.arg('position')
WHERE items.id = sqlc.arg('item_id')
  AND items.issue_id = sqlc.arg('issue_id')
  AND EXISTS (
      SELECT 1 FROM issues
      WHERE issues.id = items.issue_id AND issues.status = 'draft'
  );

-- name: ItemIDsByIssue :many
SELECT id FROM items WHERE issue_id = ? ORDER BY position ASC;

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
