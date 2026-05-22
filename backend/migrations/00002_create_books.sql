-- +goose Up
CREATE TABLE IF NOT EXISTS books (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    isbn TEXT UNIQUE,
    title TEXT NOT NULL,
    subtitle TEXT,
    authors TEXT,
    illustrators TEXT,
    publisher TEXT,
    publication_year INTEGER,
    page_count INTEGER,
    book_type TEXT CHECK(book_type IN ('hardback', 'paperback', 'board_book', 'ebook', 'audiobook', 'other')),
    reading_levels TEXT,
    genres TEXT,
    themes TEXT,
    awards TEXT,
    gift_from TEXT,
    gift_relationship TEXT,
    date_received DATE,
    condition TEXT CHECK(condition IN ('new', 'like_new', 'good', 'fair', 'poor', 'damaged')),
    location TEXT,
    notes TEXT,
    child_rating INTEGER CHECK(child_rating BETWEEN 0 AND 5),
    read_count INTEGER DEFAULT 0,
    last_read_date DATE,
    cover_image_url TEXT,
    cover_source TEXT CHECK(cover_source IN ('open_library', 'uploaded', 'manual', 'none')),
    guest_visible_fields TEXT DEFAULT '{"title":true,"authors":true,"book_type":true,"reading_levels":true,"genres":true,"themes":true,"awards":true,"cover_image_url":true,"child_rating":true,"read_count":true,"gift_from":true,"gift_relationship":true,"publisher":true,"publication_year":true,"page_count":true,"illustrators":true,"subtitle":true,"isbn":false,"condition":false,"location":false,"notes":false,"date_received":false,"last_read_date":false}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS books;
