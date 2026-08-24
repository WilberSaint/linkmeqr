-- Separate button text color from general body text color, and give the
-- logo/avatar circle its own background color instead of reusing background_value.
ALTER TABLE profile_themes
    ADD COLUMN button_text_color VARCHAR(20) NOT NULL DEFAULT '#ffffff' AFTER text_color,
    ADD COLUMN logo_background_color VARCHAR(20) NOT NULL DEFAULT '#111827' AFTER button_text_color;
