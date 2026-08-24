-- Add 'pattern' as a background type option (tiled themed SVG motif over a
-- solid base color, e.g. coffee beans, matcha leaves, hearts) alongside the
-- existing color/gradient/image, and let the background be an uploaded image
-- via profile_themes.background_media_id (background_value keeps the CSS
-- value for color/gradient/pattern; for 'image' it's resolved from the media
-- table, same pattern as profile logos).
ALTER TABLE profile_themes
    MODIFY COLUMN background_type ENUM('color','gradient','pattern','image') NOT NULL DEFAULT 'color',
    ADD COLUMN background_media_id CHAR(36) NULL AFTER background_value;

ALTER TABLE profile_themes
    ADD CONSTRAINT fk_theme_background_media FOREIGN KEY (background_media_id) REFERENCES media(id) ON DELETE SET NULL;
