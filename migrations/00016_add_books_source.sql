-- Add source column to distinguish collection books from external books
ALTER TABLE books
  ADD COLUMN source TEXT
    DEFAULT 'collection'
    CHECK(source IN ('collection', 'external'));
