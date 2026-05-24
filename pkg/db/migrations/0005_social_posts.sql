-- +goose Up
-- +goose StatementBegin
CREATE TABLE social_posts (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    issue_id    INTEGER REFERENCES issues(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL DEFAULT 'featured',
    subject     TEXT,
    platform    TEXT NOT NULL,
    text        TEXT NOT NULL,
    post_url    TEXT,
    posted_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_social_posts_posted_at ON social_posts (posted_at);
CREATE INDEX idx_social_posts_kind_platform ON social_posts (kind, platform, posted_at);
CREATE INDEX idx_social_posts_subject_platform ON social_posts (subject, platform)
    WHERE subject IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_social_posts_subject_platform;
DROP INDEX IF EXISTS idx_social_posts_kind_platform;
DROP INDEX IF EXISTS idx_social_posts_posted_at;
DROP TABLE IF EXISTS social_posts;
-- +goose StatementEnd
