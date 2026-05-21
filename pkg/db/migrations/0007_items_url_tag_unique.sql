-- +goose Up
-- +goose StatementBegin
DELETE FROM items
WHERE id NOT IN (
    SELECT MIN(id)
    FROM items
    GROUP BY issue_id, url, tag
);
CREATE UNIQUE INDEX items_url_tag_unique ON items (issue_id, url, tag);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS items_url_tag_unique;
-- +goose StatementEnd
