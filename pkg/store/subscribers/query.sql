-- name: SubscriberCreate :one
INSERT INTO subscribers (
    email, unsubscribe_token
) VALUES (
    ?, ?
)
RETURNING *;

-- name: SubscriberByID :one
SELECT * FROM subscribers WHERE id = ? LIMIT 1;

-- name: SubscriberByEmail :one
SELECT * FROM subscribers WHERE email = ? LIMIT 1;

-- name: SubscriberByUnsubscribeToken :one
SELECT * FROM subscribers WHERE unsubscribe_token = ? LIMIT 1;

-- name: SubscriberUnsubscribe :exec
UPDATE subscribers
SET unsubscribed_at = CURRENT_TIMESTAMP
WHERE unsubscribe_token = ? AND unsubscribed_at IS NULL;

-- name: SubscriberListActive :many
SELECT * FROM subscribers
WHERE unsubscribed_at IS NULL
ORDER BY id ASC;
