-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_email_events_type;
DROP INDEX IF EXISTS idx_email_events_issue_id;
DROP INDEX IF EXISTS idx_email_events_event_id;
DROP TABLE IF EXISTS email_events;

CREATE TABLE email_events (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    issue_id      INTEGER REFERENCES issues(id) ON DELETE SET NULL,
    subscriber_id INTEGER REFERENCES subscribers(id) ON DELETE SET NULL,
    item_id       INTEGER REFERENCES items(id) ON DELETE SET NULL,
    email         TEXT NOT NULL,
    event_type    TEXT NOT NULL,
    url           TEXT,
    provider_id   TEXT,
    event_id      TEXT NOT NULL,
    occurred_at   TIMESTAMP NOT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX idx_email_events_event_id ON email_events (event_id);
CREATE INDEX idx_email_events_issue_id ON email_events (issue_id);
CREATE INDEX idx_email_events_item_id  ON email_events (item_id);
CREATE INDEX idx_email_events_type     ON email_events (event_type);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_email_events_type;
DROP INDEX IF EXISTS idx_email_events_item_id;
DROP INDEX IF EXISTS idx_email_events_issue_id;
DROP INDEX IF EXISTS idx_email_events_event_id;
DROP TABLE IF EXISTS email_events;

CREATE TABLE email_events (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    issue_id      INTEGER REFERENCES issues(id) ON DELETE CASCADE,
    subscriber_id INTEGER REFERENCES subscribers(id) ON DELETE SET NULL,
    email         TEXT NOT NULL,
    event_type    TEXT NOT NULL,
    url           TEXT,
    provider_id   TEXT,
    event_id      TEXT NOT NULL,
    occurred_at   TIMESTAMP NOT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX idx_email_events_event_id ON email_events (event_id);
CREATE INDEX idx_email_events_issue_id ON email_events (issue_id);
CREATE INDEX idx_email_events_type ON email_events (event_type);
-- +goose StatementEnd
