-- +goose Up
ALTER TABLE books ADD COLUMN subject_places TEXT;
ALTER TABLE books ADD COLUMN subject_people TEXT;
ALTER TABLE books ADD COLUMN subject_times TEXT;

-- +goose Down
ALTER TABLE books DROP COLUMN subject_places;
ALTER TABLE books DROP COLUMN subject_people;
ALTER TABLE books DROP COLUMN subject_times;
