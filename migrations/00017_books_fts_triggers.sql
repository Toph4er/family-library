-- +goose Up
-- Keep books_fts in sync with books.
--
-- books_fts is an EXTERNAL-CONTENT FTS5 table (content='books',
-- content_rowid='id'). Such tables do NOT auto-sync with the content table;
-- the "No triggers needed — FTS5 handles INSERT/UPDATE/DELETE automatically"
-- note in 00014_add_books_fts.sql is incorrect. Without these triggers, any
-- book created/updated/deleted after the initial migration never reaches the
-- FTS index, so /books?q= misses it (only the books present when 00014 ran
-- are searchable).
--
-- Standard FTS5 external-content pattern: three triggers maintain the index,
-- and a final 'rebuild' command back-fills rows that were written before the
-- triggers existed (the existing production gap).
--
-- NOTE: goose's SQL parser is line-based and splits on ';' — it cannot see
-- CREATE TRIGGER … BEGIN … END bodies. Without the StatementBegin/StatementEnd
-- annotations below, the first statement goose executes is an unclosed
-- 'CREATE TRIGGER … BEGIN … VALUES (…);' which SQLite rejects with
-- "incomplete input" and aborts the migration (and the server boot).
-- Do not remove the annotations.

-- +goose StatementBegin
CREATE TRIGGER books_fts_ai AFTER INSERT ON books
BEGIN
    INSERT INTO books_fts(rowid, title, authors, isbn, genres, themes, awards, reading_levels)
    VALUES (new.id, new.title, new.authors, new.isbn, new.genres, new.themes, new.awards, new.reading_levels);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER books_fts_ad AFTER DELETE ON books
BEGIN
    INSERT INTO books_fts(books_fts, rowid, title, authors, isbn, genres, themes, awards, reading_levels)
    VALUES ('delete', old.id, old.title, old.authors, old.isbn, old.genres, old.themes, old.awards, old.reading_levels);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER books_fts_au AFTER UPDATE ON books
BEGIN
    INSERT INTO books_fts(books_fts, rowid, title, authors, isbn, genres, themes, awards, reading_levels)
    VALUES ('delete', old.id, old.title, old.authors, old.isbn, old.genres, old.themes, old.awards, old.reading_levels);
    INSERT INTO books_fts(rowid, title, authors, isbn, genres, themes, awards, reading_levels)
    VALUES (new.id, new.title, new.authors, new.isbn, new.genres, new.themes, new.awards, new.reading_levels);
END;
-- +goose StatementEnd

-- Back-fill the FTS index from the content table so books written before
-- these triggers existed become searchable.
INSERT INTO books_fts(books_fts) VALUES ('rebuild');

-- +goose Down
DROP TRIGGER IF EXISTS books_fts_au;
DROP TRIGGER IF EXISTS books_fts_ad;
DROP TRIGGER IF EXISTS books_fts_ai;
