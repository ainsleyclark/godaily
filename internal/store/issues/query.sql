-- name: IssueCreate :one
INSERT INTO issues (
    slug, sent_at, subject, summary, html_body, text_body, status
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: IssueBySlug :many
SELECT sqlc.embed(issues), sqlc.embed(items)
FROM issues
LEFT JOIN items ON items.issue_id = issues.id
WHERE issues.slug = ?
ORDER BY items.position ASC;

-- name: IssueByID :many
SELECT sqlc.embed(issues), sqlc.embed(items)
FROM issues
LEFT JOIN items ON items.issue_id = issues.id
WHERE issues.id = ?
ORDER BY items.position ASC;

-- name: IssueList :many
SELECT * FROM issues
WHERE status = 'sent'
ORDER BY sent_at DESC
LIMIT ? OFFSET ?;

-- name: IssueUpdateStatus :one
UPDATE issues SET status = ?, sent_at = ? WHERE id = ? RETURNING *;

-- name: IssueCount :one
SELECT COUNT(*) FROM issues WHERE status = 'sent';
