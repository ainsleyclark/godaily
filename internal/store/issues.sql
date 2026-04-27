-- name: CreateIssue :one
INSERT INTO issues (
    slug, sent_at, subject, summary, html_body, text_body, status
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: GetIssueBySlug :one
SELECT * FROM issues WHERE slug = ? LIMIT 1;

-- name: GetIssueByID :one
SELECT * FROM issues WHERE id = ? LIMIT 1;

-- name: ListIssues :many
SELECT * FROM issues
WHERE status = 'sent'
ORDER BY sent_at DESC
LIMIT ? OFFSET ?;

-- name: CountIssues :one
SELECT COUNT(*) FROM issues WHERE status = 'sent';
