-- Let the profile owner choose between showing an uploaded logo image or a
-- generated initial letter, and control the shape the logo/avatar renders as.
ALTER TABLE profile_themes
    ADD COLUMN logo_display_mode ENUM('image','initial') NOT NULL DEFAULT 'initial' AFTER logo_background_color,
    ADD COLUMN logo_shape ENUM('circle','rounded','square') NOT NULL DEFAULT 'circle' AFTER logo_display_mode;
