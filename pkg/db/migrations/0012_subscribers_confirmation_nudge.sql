-- +goose Up
-- +goose StatementBegin
ALTER TABLE subscribers ADD COLUMN confirmation_nudge_sent_at TIMESTAMP;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE subscribers DROP COLUMN confirmation_nudge_sent_at;
-- +goose StatementEnd
