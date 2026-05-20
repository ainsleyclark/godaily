-- name: EmailEventCreate :one
INSERT INTO email_events (
    issue_id, subscriber_id, email, event_type, url, provider_id, event_id, occurred_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: EmailEventExistsByEventID :one
SELECT EXISTS (
    SELECT 1 FROM email_events
    WHERE event_id = ?
) AS exists_flag;

-- name: EmailEventIssueStats :one
SELECT
    COUNT(CASE WHEN event_type = 'delivered' THEN 1 END)                    AS delivered,
    COUNT(DISTINCT CASE WHEN event_type = 'opened' THEN subscriber_id END)  AS unique_opens,
    COUNT(CASE WHEN event_type = 'opened' THEN 1 END)                       AS total_opens,
    COUNT(DISTINCT CASE WHEN event_type = 'clicked' THEN subscriber_id END) AS unique_clicks,
    COUNT(CASE WHEN event_type = 'clicked' THEN 1 END)                      AS total_clicks,
    COUNT(CASE WHEN event_type = 'bounced' THEN 1 END)                      AS bounced,
    COUNT(CASE WHEN event_type = 'complained' THEN 1 END)                   AS complained
FROM email_events
WHERE issue_id = ?;

-- name: EmailEventTopLinks :many
SELECT url, COUNT(*) AS clicks
FROM email_events
WHERE issue_id = ?
  AND event_type = 'clicked'
  AND url IS NOT NULL
  AND url != ''
GROUP BY url
ORDER BY clicks DESC
LIMIT ?;
