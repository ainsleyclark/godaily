-- +goose Up
-- +goose StatementBegin
CREATE TABLE social_posts (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    issue_id    INTEGER NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    platform    TEXT NOT NULL,
    text        TEXT NOT NULL,
    post_url    TEXT,
    posted_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (issue_id, platform)
);
CREATE INDEX idx_social_posts_posted_at ON social_posts (posted_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_social_posts_posted_at;
DROP TABLE IF EXISTS social_posts;
-- +goose StatementEnd
