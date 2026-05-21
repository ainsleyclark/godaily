-- name: SocialMetricUpsert :exec
INSERT INTO social_metrics (social_post_id, platform, likes, reposts, comments, impressions, fetched_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(social_post_id, platform) DO UPDATE SET
    likes        = excluded.likes,
    reposts      = excluded.reposts,
    comments     = excluded.comments,
    impressions  = excluded.impressions,
    fetched_at   = excluded.fetched_at;

-- name: SocialMetricListBySocialPostID :many
SELECT * FROM social_metrics
WHERE social_post_id = ?
ORDER BY fetched_at DESC;
