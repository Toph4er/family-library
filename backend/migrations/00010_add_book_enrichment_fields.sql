-- +goose Up
ALTER TABLE books ADD COLUMN dewey_decimal_class TEXT;
ALTER TABLE books ADD COLUMN description TEXT;
ALTER TABLE books ADD COLUMN language TEXT;

-- +goose Down
ALTER TABLE books DROP COLUMN dewey_decimal_class;
ALTER TABLE books DROP COLUMN description;
ALTER TABLE books DROP COLUMN language;
