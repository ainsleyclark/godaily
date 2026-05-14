-- +goose Up
-- +goose StatementBegin
ALTER TABLE items ADD COLUMN original_url TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE items DROP COLUMN original_url;
-- +goose StatementEnd
