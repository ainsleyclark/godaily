-- +goose Up
-- +goose StatementBegin
ALTER TABLE issues DROP COLUMN html_body;
ALTER TABLE issues DROP COLUMN text_body;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE issues ADD COLUMN html_body TEXT NOT NULL DEFAULT '';
ALTER TABLE issues ADD COLUMN text_body TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd
