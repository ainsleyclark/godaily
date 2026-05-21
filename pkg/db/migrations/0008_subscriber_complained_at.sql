-- +goose Up
-- +goose StatementBegin
ALTER TABLE subscribers ADD COLUMN complained_at TIMESTAMP;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE subscribers DROP COLUMN complained_at;
-- +goose StatementEnd
