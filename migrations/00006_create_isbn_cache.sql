-- +goose Up
CREATE TABLE IF NOT EXISTS isbn_cache (
    isbn TEXT PRIMARY KEY,
    data TEXT NOT NULL,
    fetched_at TEXT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS isbn_cache;
