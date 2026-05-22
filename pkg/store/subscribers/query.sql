-- name: SubscriberCreate :one
INSERT INTO subscribers (
    email, unsubscribe_token, confirm_token
) VALUES (
    ?, ?, ?
)
RETURNING *;

-- name: SubscriberByID :one
SELECT * FROM subscribers WHERE id = ? LIMIT 1;

-- name: SubscriberByEmail :one
SELECT * FROM subscribers WHERE email = ? LIMIT 1;

-- name: SubscriberByUnsubscribeToken :one
SELECT * FROM subscribers WHERE unsubscribe_token = ? LIMIT 1;

-- name: SubscriberByConfirmToken :one
SELECT * FROM subscribers WHERE confirm_token = ? LIMIT 1;

-- name: SubscriberConfirm :one
UPDATE subscribers
SET confirmed_at = CURRENT_TIMESTAMP,
    confirm_token = NULL
WHERE confirm_token = ? AND confirmed_at IS NULL
RETURNING *;

-- name: SubscriberUnsubscribe :exec
UPDATE subscribers
SET unsubscribed_at = CURRENT_TIMESTAMP
WHERE unsubscribe_token = ? AND unsubscribed_at IS NULL;

-- name: SubscriberReactivate :one
UPDATE subscribers
SET unsubscribed_at = NULL,
    confirmed_at = NULL,
    confirm_token = ?,
    unsubscribe_token = ?
WHERE email = ? AND unsubscribed_at IS NOT NULL
RETURNING *;

-- name: SubscriberMarkBounced :exec
UPDATE subscribers
SET bounced_at = CURRENT_TIMESTAMP
WHERE email = ? AND bounced_at IS NULL;

-- name: SubscriberMarkComplained :exec
UPDATE subscribers
SET unsubscribed_at = CURRENT_TIMESTAMP
WHERE email = ? AND unsubscribed_at IS NULL;

-- name: SubscriberMarkSuppressed :exec
UPDATE subscribers
SET suppressed_at = CURRENT_TIMESTAMP
WHERE email = ? AND suppressed_at IS NULL;

-- name: SubscriberListActive :many
SELECT * FROM subscribers
WHERE unsubscribed_at IS NULL
  AND confirmed_at IS NOT NULL
  AND bounced_at IS NULL
  AND suppressed_at IS NULL
ORDER BY id ASC;

-- name: SubscriberCountActive :one
SELECT COUNT(*) FROM subscribers
WHERE unsubscribed_at IS NULL
  AND bounced_at IS NULL
  AND suppressed_at IS NULL;
