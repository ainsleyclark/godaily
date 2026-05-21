-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE INDEX idx_items_url_tag ON items(url, tag);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_items_url_tag;
-- +goose StatementEnd
