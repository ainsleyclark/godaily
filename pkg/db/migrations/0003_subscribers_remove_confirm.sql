-- +goose Up
-- SQLite doesn't support DROP COLUMN on columns with constraints, so recreate the table.
CREATE TABLE subscribers_new (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    email             TEXT NOT NULL UNIQUE,
    unsubscribe_token TEXT NOT NULL UNIQUE,
    unsubscribed_at   TIMESTAMP,
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO subscribers_new (id, email, unsubscribe_token, unsubscribed_at, created_at)
    SELECT id, email, unsubscribe_token, unsubscribed_at, created_at FROM subscribers;
DROP TABLE subscribers;
ALTER TABLE subscribers_new RENAME TO subscribers;
DROP INDEX IF EXISTS idx_subscribers_active;
CREATE INDEX idx_subscribers_active ON subscribers(id) WHERE unsubscribed_at IS NULL;

-- +goose Down
CREATE TABLE subscribers_old (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    email             TEXT NOT NULL UNIQUE,
    confirm_token     TEXT NOT NULL DEFAULT '',
    unsubscribe_token TEXT NOT NULL UNIQUE,
    confirmed_at      TIMESTAMP,
    unsubscribed_at   TIMESTAMP,
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO subscribers_old (id, email, unsubscribe_token, unsubscribed_at, created_at)
    SELECT id, email, unsubscribe_token, unsubscribed_at, created_at FROM subscribers;
DROP TABLE subscribers;
ALTER TABLE subscribers_old RENAME TO subscribers;
DROP INDEX IF EXISTS idx_subscribers_active;
CREATE INDEX idx_subscribers_active ON subscribers(confirmed_at) WHERE unsubscribed_at IS NULL;
