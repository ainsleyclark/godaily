-- +goose Up
-- +goose StatementBegin
ALTER TABLE subscribers ADD COLUMN suppressed_at TIMESTAMP;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE subscribers DROP COLUMN suppressed_at;
-- +goose StatementEnd
