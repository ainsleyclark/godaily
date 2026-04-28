-- name: CreateSubscriber :one
INSERT INTO subscribers (
    email, confirm_token, unsubscribe_token
) VALUES (
    ?, ?, ?
)
RETURNING *;

-- name: GetSubscriberByEmail :one
SELECT * FROM subscribers WHERE email = ? LIMIT 1;

-- name: GetSubscriberByConfirmToken :one
SELECT * FROM subscribers WHERE confirm_token = ? LIMIT 1;

-- name: GetSubscriberByUnsubscribeToken :one
SELECT * FROM subscribers WHERE unsubscribe_token = ? LIMIT 1;

-- name: ConfirmSubscriber :exec
UPDATE subscribers
SET confirmed_at = CURRENT_TIMESTAMP
WHERE confirm_token = ? AND confirmed_at IS NULL;

-- name: UnsubscribeByToken :exec
UPDATE subscribers
SET unsubscribed_at = CURRENT_TIMESTAMP
WHERE unsubscribe_token = ? AND unsubscribed_at IS NULL;

-- name: ListActiveSubscribers :many
SELECT * FROM subscribers
WHERE confirmed_at IS NOT NULL AND unsubscribed_at IS NULL
ORDER BY id ASC;
