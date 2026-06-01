-- +goose Up
CREATE TABLE IF NOT EXISTS reading_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id INTEGER NOT NULL,
    start_page INTEGER,
    end_page INTEGER,
    total_pages INTEGER,
    entire_book INTEGER NOT NULL DEFAULT 0,
    read_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reader_name TEXT NOT NULL,
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS reading_logs;
