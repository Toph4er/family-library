-- +goose Up
-- FTS5 virtual table for full-text search on books.
-- Covers the fields used in the search queries: title, authors, isbn,
-- genres, themes, awards, reading_levels.
--
-- Uses content='books' so FTS5 auto-syncs with the books table.
-- No triggers needed — FTS5 handles INSERT/UPDATE/DELETE automatically.
CREATE VIRTUAL TABLE books_fts USING fts5(
	title, authors, isbn, genres, themes, awards, reading_levels,
	content='books', content_rowid='id'
);

-- Populate the FTS index from existing data.
INSERT INTO books_fts(rowid, title, authors, isbn, genres, themes, awards, reading_levels)
	SELECT id, title, authors, isbn, genres, themes, awards, reading_levels
	FROM books;

-- +goose Down
DROP TABLE IF EXISTS books_fts;
