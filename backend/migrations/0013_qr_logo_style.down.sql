ALTER TABLE qr_codes
    ADD COLUMN logo_halftone TINYINT(1) NOT NULL DEFAULT 0 AFTER logo_style;

UPDATE qr_codes SET logo_halftone = 1 WHERE logo_style = 'dots';

ALTER TABLE qr_codes DROP COLUMN logo_style;
