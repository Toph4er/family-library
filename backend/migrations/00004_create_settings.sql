-- +goose Up
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO settings (key, value, description) VALUES
('cover_image_provider', 'open_library', 'Primary API for fetching book covers'),
('cover_image_fallback', 'open_library', 'Fallback API if primary fails'),
('guest_password_hash', '', 'Hash of the shared guest password'),
('site_name', 'Our Library', 'Display name for the site'),
('site_tagline', 'A woodland fairy tale collection', 'Tagline shown on landing page'),
('theme_colors', '{"primary":"#2d5016","secondary":"#8b4513","accent":"#d4a574","background":"#f5f0e8","surface":"#faf8f5","text":"#1a2f0a","textLight":"#4a5d23","success":"#4a7c2e","warning":"#c75b39","error":"#8b2500"}', 'Theme color palette'),
('default_guest_visibility', '{"title":true,"authors":true,"book_type":true,"reading_levels":true,"genres":true,"themes":true,"awards":true,"cover_image_url":true,"child_rating":true,"read_count":true,"gift_from":true,"gift_relationship":true,"publisher":true,"publication_year":true,"page_count":true,"illustrators":true,"subtitle":true,"isbn":false,"condition":false,"location":false,"notes":false,"date_received":false,"last_read_date":false}', 'Default field visibility for guests');

-- +goose Down
DROP TABLE IF EXISTS settings;
