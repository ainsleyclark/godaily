-- name: IssueCreate :one
INSERT INTO issues (
    slug, sent_at, subject, summary, status
) VALUES (
    ?, ?, ?, ?, ?
)
RETURNING *;

-- name: IssueBySlug :one
SELECT * FROM issues WHERE slug = ? LIMIT 1;

-- name: IssueByID :one
SELECT * FROM issues WHERE id = ? LIMIT 1;

-- name: IssueList :many
SELECT * FROM issues
WHERE status = 'sent'
ORDER BY sent_at DESC
LIMIT ? OFFSET ?;

-- name: IssueUpdateStatus :one
UPDATE issues SET status = ?, sent_at = ? WHERE id = ? RETURNING *;

-- name: IssueUpdate :one
UPDATE issues SET subject = ?, summary = ? WHERE id = ? RETURNING *;

-- name: IssueCount :one
SELECT COUNT(*) FROM issues WHERE status = 'sent';
