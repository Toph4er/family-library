-- +goose Up
CREATE INDEX IF NOT EXISTS idx_books_title ON books(title);
CREATE INDEX IF NOT EXISTS idx_books_authors ON books(authors);
CREATE INDEX IF NOT EXISTS idx_books_isbn ON books(isbn);
CREATE INDEX IF NOT EXISTS idx_books_book_type ON books(book_type);
CREATE INDEX IF NOT EXISTS idx_books_reading_levels ON books(reading_levels);
CREATE INDEX IF NOT EXISTS idx_books_genres ON books(genres);
CREATE INDEX IF NOT EXISTS idx_books_themes ON books(themes);
CREATE INDEX IF NOT EXISTS idx_books_awards ON books(awards);
CREATE INDEX IF NOT EXISTS idx_books_gift_from ON books(gift_from);
CREATE INDEX IF NOT EXISTS idx_wishlist_priority ON wishlist(priority);

-- +goose Down
DROP INDEX IF EXISTS idx_books_title;
DROP INDEX IF EXISTS idx_books_authors;
DROP INDEX IF EXISTS idx_books_isbn;
DROP INDEX IF EXISTS idx_books_book_type;
DROP INDEX IF EXISTS idx_books_reading_levels;
DROP INDEX IF EXISTS idx_books_genres;
DROP INDEX IF EXISTS idx_books_themes;
DROP INDEX IF EXISTS idx_books_awards;
DROP INDEX IF EXISTS idx_books_gift_from;
DROP INDEX IF EXISTS idx_wishlist_priority;
