-- +goose Up
ALTER TABLE books ADD COLUMN age_range TEXT;
ALTER TABLE books ADD COLUMN series TEXT;

-- +goose Down
ALTER TABLE books DROP COLUMN age_range;
ALTER TABLE books DROP COLUMN series;
