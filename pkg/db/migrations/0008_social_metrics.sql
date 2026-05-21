-- +goose Up
-- +goose StatementBegin
CREATE TABLE social_metrics (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    social_post_id INTEGER NOT NULL REFERENCES social_posts(id) ON DELETE CASCADE,
    platform       TEXT NOT NULL,
    likes          INTEGER NOT NULL DEFAULT 0,
    reposts        INTEGER NOT NULL DEFAULT 0,
    comments       INTEGER NOT NULL DEFAULT 0,
    impressions    INTEGER NOT NULL DEFAULT 0,
    fetched_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_social_metrics_post_platform ON social_metrics (social_post_id, platform);
CREATE INDEX idx_social_metrics_social_post_id ON social_metrics (social_post_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_social_metrics_social_post_id;
DROP INDEX IF EXISTS idx_social_metrics_post_platform;
DROP TABLE IF EXISTS social_metrics;
-- +goose StatementEnd
