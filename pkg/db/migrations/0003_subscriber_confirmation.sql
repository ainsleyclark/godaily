-- +goose Up
-- +goose StatementBegin
ALTER TABLE subscribers ADD COLUMN confirm_token TEXT;
ALTER TABLE subscribers ADD COLUMN confirmed_at TIMESTAMP;

CREATE UNIQUE INDEX idx_subscribers_confirm_token ON subscribers(confirm_token) WHERE confirm_token IS NOT NULL;

-- Grandfather existing subscribers in as already confirmed.
UPDATE subscribers SET confirmed_at = created_at WHERE confirmed_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_subscribers_confirm_token;
ALTER TABLE subscribers DROP COLUMN confirm_token;
ALTER TABLE subscribers DROP COLUMN confirmed_at;
-- +goose StatementEnd
