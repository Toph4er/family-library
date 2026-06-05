-- +goose Up
INSERT OR IGNORE INTO settings (key, value, description) VALUES
('active_theme', 'woodland', 'Active visual theme for the library');

-- +goose Down
DELETE FROM settings WHERE key = 'active_theme';
