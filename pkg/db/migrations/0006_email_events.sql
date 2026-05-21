-- +goose Up
-- +goose StatementBegin
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

ALTER TABLE subscribers ADD COLUMN bounced_at TIMESTAMP;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE subscribers DROP COLUMN bounced_at;

DROP INDEX IF EXISTS idx_email_events_type;
DROP INDEX IF EXISTS idx_email_events_issue_id;
DROP INDEX IF EXISTS idx_email_events_event_id;
DROP TABLE IF EXISTS email_events;
-- +goose StatementEnd
