ALTER TABLE profile_themes DROP FOREIGN KEY fk_theme_background_media;
ALTER TABLE profile_themes DROP COLUMN background_media_id;
ALTER TABLE profile_themes MODIFY COLUMN background_type ENUM('color','gradient','image') NOT NULL DEFAULT 'color';
