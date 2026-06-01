-- name: EmailEventCreate :one
INSERT INTO email_events (
    issue_id, subscriber_id, item_id, email, event_type, url, provider_id, event_id, occurred_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: EmailEventExistsByEventID :one
SELECT EXISTS (
    SELECT 1 FROM email_events
    WHERE event_id = ?
) AS exists_flag;

-- name: EmailEventIssueStats :one
SELECT
    COUNT(CASE WHEN event_type = 'delivered' THEN 1 END)                      AS delivered,
    COUNT(DISTINCT CASE WHEN event_type = 'opened' THEN subscriber_id END)    AS unique_opens,
    COUNT(CASE WHEN event_type = 'opened' THEN 1 END)                         AS total_opens,
    COUNT(DISTINCT CASE WHEN event_type = 'clicked' THEN subscriber_id END)   AS unique_clicks,
    COUNT(CASE WHEN event_type = 'clicked' THEN 1 END)                        AS total_clicks,
    COUNT(CASE WHEN event_type = 'bounced' THEN 1 END)                        AS bounced,
    COUNT(CASE WHEN event_type = 'complained' THEN 1 END)                     AS complained,
    COUNT(CASE WHEN event_type = 'delivery_delayed' THEN 1 END)               AS delayed,
    COUNT(CASE WHEN event_type = 'failed' THEN 1 END)                         AS failed,
    COUNT(CASE WHEN event_type = 'suppressed' THEN 1 END)                     AS suppressed
FROM email_events
WHERE issue_id = ?;

-- name: EmailEventTopLinks :many
SELECT
    e.url        AS url,
    it.title     AS title,
    it.tag       AS tag,
    it.source    AS source,
    COUNT(*)     AS clicks
FROM email_events e
LEFT JOIN items it ON it.id = e.item_id
WHERE e.issue_id = ?
  AND e.event_type = 'clicked'
  AND e.url IS NOT NULL
  AND e.url != ''
GROUP BY e.url, it.title, it.tag, it.source
ORDER BY clicks DESC
LIMIT ?;
