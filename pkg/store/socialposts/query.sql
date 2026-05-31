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

-- name: SocialPostsWithMetrics :many
SELECT
    sp.id,
    sp.issue_id,
    sp.kind,
    sp.subject,
    sp.platform,
    sp.text,
    sp.post_url,
    sp.posted_at,
    COALESCE(sm.likes, 0)       AS likes,
    COALESCE(sm.reposts, 0)     AS reposts,
    COALESCE(sm.comments, 0)    AS comments,
    COALESCE(sm.impressions, 0) AS impressions
FROM social_posts sp
LEFT JOIN social_metrics sm ON sm.social_post_id = sp.id
WHERE (sqlc.narg('from') IS NULL OR sp.posted_at >= sqlc.narg('from'))
  AND (sqlc.narg('to')   IS NULL OR sp.posted_at <  sqlc.narg('to'))
ORDER BY sp.posted_at DESC;
