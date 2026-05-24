-- name: SocialPostCreate :one
INSERT INTO social_posts (
    issue_id, kind, subject, platform, text, post_url, posted_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: SocialPostExists :one
SELECT EXISTS (
    SELECT 1 FROM social_posts
    WHERE issue_id = ? AND platform = ? AND kind = 'featured'
) AS exists_flag;

-- name: SocialPostExistsBySubject :one
SELECT EXISTS (
    SELECT 1 FROM social_posts
    WHERE subject = ? AND platform = ?
) AS exists_flag;

-- name: SocialPostExistsKindSince :one
SELECT EXISTS (
    SELECT 1 FROM social_posts
    WHERE kind = ? AND platform = ? AND posted_at >= ?
) AS exists_flag;

-- name: SocialPostListByIssue :many
SELECT * FROM social_posts
WHERE issue_id = ?
ORDER BY posted_at ASC;

-- name: SocialPostListSince :many
SELECT * FROM social_posts
WHERE posted_at >= ?
ORDER BY posted_at DESC;
