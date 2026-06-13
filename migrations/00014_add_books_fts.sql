-- +goose Up
-- FTS5 virtual table for full-text search on books.
-- Covers the fields used in the search queries: title, authors, isbn,
-- genres, themes, awards, reading_levels.
CREATE VIRTUAL TABLE books_fts USING fts5(
	title, authors, isbn, genres, themes, awards, reading_levels,
	content='books', content_rowid='id'
);

-- Populate the FTS index from existing data.
INSERT INTO books_fts(rowid, title, authors, isbn, genres, themes, awards, reading_levels)
	SELECT id, title, authors, isbn, genres, themes, awards, reading_levels
	FROM books;

-- Triggers to keep the FTS index in sync with the books table.
CREATE TRIGGER books_ai AFTER INSERT ON books BEGIN
	INSERT INTO books_fts(rowid, title, authors, isbn, genres, themes, awards, reading_levels)
		VALUES (new.id, new.title, new.authors, new.isbn, new.genres, new.themes, new.awards, new.reading_levels);
END;

CREATE TRIGGER books_ad AFTER DELETE ON books BEGIN
	INSERT INTO books_fts(books_fts, rowid, title, authors, isbn, genres, themes, awards, reading_levels)
		VALUES ('delete', old.id, old.title, old.authors, old.isbn, old.genres, old.themes, old.awards, old.reading_levels);
END;

CREATE TRIGGER books_au AFTER UPDATE ON books BEGIN
	INSERT INTO books_fts(books_fts, rowid, title, authors, isbn, genres, themes, awards, reading_levels)
		VALUES ('delete', old.id, old.title, old.authors, old.isbn, old.genres, old.themes, old.awards, old.reading_levels);
	INSERT INTO books_fts(rowid, title, authors, isbn, genres, themes, awards, reading_levels)
		VALUES (new.id, new.title, new.authors, new.isbn, new.genres, new.themes, new.awards, new.reading_levels);
END;

-- +goose Down
DROP TRIGGER IF EXISTS books_ai;
DROP TRIGGER IF EXISTS books_ad;
DROP TRIGGER IF EXISTS books_au;
DROP TABLE IF EXISTS books_fts;
