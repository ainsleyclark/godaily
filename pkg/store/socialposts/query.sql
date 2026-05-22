-- name: SocialPostCreate :one
INSERT INTO social_posts (
    issue_id, platform, text, post_url, posted_at
) VALUES (
    ?, ?, ?, ?, ?
)
RETURNING *;

-- name: SocialPostExists :one
SELECT EXISTS (
    SELECT 1 FROM social_posts
    WHERE issue_id = ? AND platform = ?
) AS exists_flag;

-- name: SocialPostListByIssue :many
SELECT * FROM social_posts
WHERE issue_id = ?
ORDER BY posted_at ASC;

-- name: SocialPostListSince :many
SELECT * FROM social_posts
WHERE posted_at >= ?
ORDER BY posted_at DESC;
