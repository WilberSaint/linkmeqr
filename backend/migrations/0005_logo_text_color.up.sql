-- Let the initial-letter logo have its own text color instead of a hardcoded white.
ALTER TABLE profile_themes
    ADD COLUMN logo_text_color VARCHAR(20) NOT NULL DEFAULT '#ffffff' AFTER logo_background_color;
