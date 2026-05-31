-- +goose Up
ALTER TABLE books ADD COLUMN quantity INTEGER DEFAULT 1 CHECK(quantity >= 1);

-- +goose Down
ALTER TABLE books DROP COLUMN quantity;
