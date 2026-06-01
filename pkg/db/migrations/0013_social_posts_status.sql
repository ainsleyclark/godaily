-- +goose Up
-- +goose StatementBegin
ALTER TABLE social_posts ADD COLUMN status TEXT NOT NULL DEFAULT 'published';
ALTER TABLE social_posts ADD COLUMN published_at TIMESTAMP;
ALTER TABLE social_posts ADD COLUMN mention_source TEXT;

UPDATE social_posts SET published_at = posted_at WHERE published_at IS NULL;

CREATE INDEX idx_social_posts_status_issue ON social_posts (status, issue_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_social_posts_status_issue;
ALTER TABLE social_posts DROP COLUMN mention_source;
ALTER TABLE social_posts DROP COLUMN published_at;
ALTER TABLE social_posts DROP COLUMN status;
-- +goose StatementEnd
