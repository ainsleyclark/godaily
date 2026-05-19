-- +goose Up
-- +goose StatementBegin
ALTER TABLE items ADD COLUMN published TIMESTAMP;
CREATE INDEX items_published_idx ON items (published);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS items_published_idx;
ALTER TABLE items DROP COLUMN published;
-- +goose StatementEnd
