-- name: SubscriberCreate :one
INSERT INTO subscribers (
    email, confirm_token, unsubscribe_token
) VALUES (
    ?, ?, ?
)
RETURNING *;

-- name: SubscriberByID :one
SELECT * FROM subscribers WHERE id = ? LIMIT 1;

-- name: SubscriberByEmail :one
SELECT * FROM subscribers WHERE email = ? LIMIT 1;

-- name: SubscriberByConfirmToken :one
SELECT * FROM subscribers WHERE confirm_token = ? LIMIT 1;

-- name: SubscriberByUnsubscribeToken :one
SELECT * FROM subscribers WHERE unsubscribe_token = ? LIMIT 1;

-- name: SubscriberConfirm :exec
UPDATE subscribers
SET confirmed_at = CURRENT_TIMESTAMP
WHERE confirm_token = ? AND confirmed_at IS NULL;

-- name: SubscriberUnsubscribe :exec
UPDATE subscribers
SET unsubscribed_at = CURRENT_TIMESTAMP
WHERE unsubscribe_token = ? AND unsubscribed_at IS NULL;

-- name: SubscriberListActive :many
SELECT * FROM subscribers
WHERE confirmed_at IS NOT NULL AND unsubscribed_at IS NULL
ORDER BY id ASC;
