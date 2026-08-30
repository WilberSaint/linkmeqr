-- Decorative outer frame silhouette (heart, coffee cup, etc.) drawn around the
-- actual scannable square QR — independent of logo_media_id/preset_icon,
-- which stay centered inside the QR's own window.
ALTER TABLE qr_codes
    ADD COLUMN frame_shape VARCHAR(50) NULL AFTER preset_icon;
