-- +goose Up
-- +goose StatementBegin
CREATE TABLE issues (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    slug            TEXT NOT NULL UNIQUE,
    sent_at         TIMESTAMP NOT NULL,
    subject         TEXT NOT NULL,
    summary         TEXT,
    html_body       TEXT NOT NULL,
    text_body       TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'sent'
);

CREATE TABLE items (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    issue_id           INTEGER NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    source             TEXT NOT NULL,
    title              TEXT NOT NULL,
    url                TEXT NOT NULL,
    author_name        TEXT,
    author_username    TEXT,
    author_avatar_url  TEXT,
    author_profile_url TEXT,
    score              REAL,
    summary            TEXT,
    position           INTEGER NOT NULL
);
CREATE INDEX idx_items_issue ON items(issue_id);

CREATE TABLE subscribers (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    email             TEXT NOT NULL UNIQUE,
    confirm_token     TEXT NOT NULL UNIQUE,
    unsubscribe_token TEXT NOT NULL UNIQUE,
    confirmed_at      TIMESTAMP,
    unsubscribed_at   TIMESTAMP,
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_subscribers_active
    ON subscribers(confirmed_at) WHERE unsubscribed_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_subscribers_active;
DROP TABLE IF EXISTS subscribers;
DROP INDEX IF EXISTS idx_items_issue;
DROP TABLE IF EXISTS items;
DROP TABLE IF EXISTS issues;
-- +goose StatementEnd
