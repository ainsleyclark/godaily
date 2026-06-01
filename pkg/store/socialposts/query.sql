-- name: SocialPostCreate :one
INSERT INTO social_posts (
    issue_id, kind, subject, platform, text, post_url, posted_at, status, published_at, mention_source
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: SocialPostExists :one
SELECT EXISTS (
    SELECT 1 FROM social_posts
    WHERE issue_id = ? AND platform = ? AND kind = 'featured' AND status = 'published'
) AS exists_flag;

-- name: SocialPostExistsBySubject :one
SELECT EXISTS (
    SELECT 1 FROM social_posts
    WHERE subject = ? AND platform = ? AND status = 'published'
) AS exists_flag;

-- name: SocialPostExistsOrCancelledBySubject :one
SELECT EXISTS (
    SELECT 1 FROM social_posts
    WHERE subject = ? AND platform = ? AND status IN ('published', 'cancelled')
) AS exists_flag;

-- name: SocialPostExistsKindSince :one
SELECT EXISTS (
    SELECT 1 FROM social_posts
    WHERE kind = ? AND platform = ? AND posted_at >= ? AND status = 'published'
) AS exists_flag;

-- name: SocialPostList :many
SELECT * FROM social_posts
WHERE (sqlc.narg('issue_id') IS NULL OR issue_id = sqlc.narg('issue_id'))
  AND (sqlc.narg('since')    IS NULL OR posted_at >= sqlc.narg('since'))
  AND (sqlc.narg('status')   IS NULL OR status    = sqlc.narg('status'))
  AND (sqlc.narg('platform') IS NULL OR platform  = sqlc.narg('platform'))
ORDER BY posted_at DESC, id DESC;

-- name: SocialPostGet :one
SELECT * FROM social_posts WHERE id = ?;

-- name: SocialPostUpdate :one
UPDATE social_posts
SET text         = COALESCE(sqlc.narg('text'),         text),
    status       = COALESCE(sqlc.narg('status'),       status),
    published_at = COALESCE(sqlc.narg('published_at'), published_at),
    post_url     = COALESCE(sqlc.narg('post_url'),     post_url)
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: SocialPostDeleteDraftsByIssue :exec
DELETE FROM social_posts WHERE issue_id = ? AND status = 'draft';

-- name: SocialPostDeleteDraftsByKind :exec
DELETE FROM social_posts WHERE issue_id IS NULL AND kind = ? AND status = 'draft';

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
  AND sp.status = 'published'
ORDER BY sp.posted_at DESC;
