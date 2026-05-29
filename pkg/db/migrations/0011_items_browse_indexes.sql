-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS items_source_idx ON items (source);
CREATE INDEX IF NOT EXISTS items_tag_idx ON items (tag);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS items_source_idx;
DROP INDEX IF EXISTS items_tag_idx;
-- +goose StatementEnd
