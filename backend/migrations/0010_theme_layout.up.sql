ALTER TABLE profile_themes
    ADD COLUMN layout ENUM('list','grid') NOT NULL DEFAULT 'list' AFTER button_shadow;
