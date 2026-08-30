-- Replaces the halftone-only boolean with a proper 3-way choice for how an
-- uploaded logo is rendered: 'color' (the original image, unmodified),
-- 'monochrome' (recolored to the QR's own ink color as a solid silhouette
-- — the recommended default, since it reads as part of the code rather
-- than a sticker placed on top of it), or 'dots' (the existing halftone
-- texture). Has no effect on preset icons.
ALTER TABLE qr_codes
    ADD COLUMN logo_style VARCHAR(20) NOT NULL DEFAULT 'color' AFTER logo_halftone;

UPDATE qr_codes SET logo_style = 'dots' WHERE logo_halftone = 1;

ALTER TABLE qr_codes DROP COLUMN logo_halftone;
