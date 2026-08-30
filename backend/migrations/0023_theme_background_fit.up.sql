-- How an 'image' background is sized against the viewport: 'cover' fills
-- edge to edge and crops (right for a full-bleed photo), 'contain' shows the
-- whole image with the theme's own background_value as letterbox fill
-- (right for a framed/poster-style illustration with its own border), and
-- 'repeat' tiles it at natural size (right for a genuinely seamless
-- pattern). Meaningless for color/gradient/pattern types, but always set so
-- there's never a null branch to handle.
ALTER TABLE profile_themes
    ADD COLUMN background_fit ENUM('cover','contain','repeat') NOT NULL DEFAULT 'cover' AFTER background_media_id;
