-- name: MetricsSummary :one
SELECT
    COUNT(DISTINCT CASE WHEN e.event_type = 'delivered'           THEN e.issue_id      END) AS issues_sent,
    COUNT(CASE          WHEN e.event_type = 'delivered'           THEN 1               END) AS delivered,
    COUNT(DISTINCT CASE WHEN e.event_type = 'opened'              THEN e.subscriber_id END) AS unique_opens,
    COUNT(CASE          WHEN e.event_type = 'opened'              THEN 1               END) AS total_opens,
    COUNT(DISTINCT CASE WHEN e.event_type = 'clicked'             THEN e.subscriber_id END) AS unique_clicks,
    COUNT(CASE          WHEN e.event_type = 'clicked'             THEN 1               END) AS total_clicks,
    COUNT(CASE          WHEN e.event_type = 'bounced'             THEN 1               END) AS bounced,
    COUNT(CASE          WHEN e.event_type = 'complained'          THEN 1               END) AS complained,
    COUNT(DISTINCT CASE WHEN e.event_type IN ('opened','clicked') THEN e.subscriber_id END) AS unique_engaged
FROM email_events e
WHERE e.issue_id IS NOT NULL
  AND (sqlc.narg('from') IS NULL OR e.occurred_at >= sqlc.narg('from'))
  AND (sqlc.narg('to')   IS NULL OR e.occurred_at <  sqlc.narg('to'));

-- name: MetricsItemList :many
SELECT
    it.id,
    it.title,
    it.url,
    it.tag,
    it.source,
    COUNT(*) AS clicks
FROM email_events e
JOIN items it ON it.id = e.item_id
WHERE e.event_type = 'clicked'
  AND e.item_id IS NOT NULL
  AND (sqlc.narg('from') IS NULL OR e.occurred_at >= sqlc.narg('from'))
  AND (sqlc.narg('to')   IS NULL OR e.occurred_at <  sqlc.narg('to'))
GROUP BY it.id, it.title, it.url, it.tag, it.source
ORDER BY clicks DESC
LIMIT sqlc.arg('limit');

-- name: MetricsTagList :many
SELECT
    it.tag,
    COUNT(*) AS clicks
FROM email_events e
JOIN items it ON it.id = e.item_id
WHERE e.event_type = 'clicked'
  AND e.item_id IS NOT NULL
  AND (sqlc.narg('from') IS NULL OR e.occurred_at >= sqlc.narg('from'))
  AND (sqlc.narg('to')   IS NULL OR e.occurred_at <  sqlc.narg('to'))
GROUP BY it.tag
ORDER BY clicks DESC
LIMIT sqlc.arg('limit');

-- name: MetricsSourceList :many
SELECT
    it.source,
    COUNT(*) AS clicks
FROM email_events e
JOIN items it ON it.id = e.item_id
WHERE e.event_type = 'clicked'
  AND e.item_id IS NOT NULL
  AND (sqlc.narg('from') IS NULL OR e.occurred_at >= sqlc.narg('from'))
  AND (sqlc.narg('to')   IS NULL OR e.occurred_at <  sqlc.narg('to'))
GROUP BY it.source
ORDER BY clicks DESC
LIMIT sqlc.arg('limit');
