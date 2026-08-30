-- When true, the frame_shape silhouette isn't just decorative background —
-- the QR's own data modules are masked to approximate it (module "erosion"),
-- relying on error correction to reconstruct the missing modules.
ALTER TABLE qr_codes
    ADD COLUMN shape_fill TINYINT(1) NOT NULL DEFAULT 0 AFTER frame_shape;
