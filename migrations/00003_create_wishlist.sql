-- +goose Up
CREATE TABLE IF NOT EXISTS wishlist (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    author TEXT,
    isbn TEXT,
    reason TEXT,
    priority INTEGER DEFAULT 5 CHECK(priority BETWEEN 1 AND 10),
    amazon_url TEXT,
    thriftbooks_url TEXT,
    other_urls TEXT,
    cover_image_url TEXT,
    requested_by TEXT,
    requested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    fulfilled BOOLEAN DEFAULT FALSE,
    fulfilled_at TIMESTAMP,
    notes TEXT
);

-- +goose Down
DROP TABLE IF EXISTS wishlist;
