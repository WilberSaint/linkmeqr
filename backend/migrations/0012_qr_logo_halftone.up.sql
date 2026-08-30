-- When true and a custom logo image is set, the logo is rendered as a
-- halftone dot pattern (variable-radius circles sampled from the image's
-- luminance) instead of a smoothly scaled photo, so it visually blends with
-- the QR's own dot/module aesthetic. Has no effect on preset icons, which
-- are already simple procedural glyphs.
ALTER TABLE qr_codes
    ADD COLUMN logo_halftone TINYINT(1) NOT NULL DEFAULT 0 AFTER logo_media_id;
