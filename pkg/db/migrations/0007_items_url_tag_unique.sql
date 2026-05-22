-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE INDEX items_url_tag_unique ON items (url, tag);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS items_url_tag_unique;
-- +goose StatementEnd
