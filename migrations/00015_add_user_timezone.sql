-- +goose Up
INSERT OR IGNORE INTO settings (key, value, description) VALUES
('user_timezone', 'America/New_York', 'Default timezone for displaying dates and times');

-- +goose Down
DELETE FROM settings WHERE key = 'user_timezone';
