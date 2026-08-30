-- When true and the logo is set to 'color' style, the three finder-pattern
-- eyes are tinted with a dominant color sampled from the logo (e.g. a green
-- logo gets green eyes) instead of the QR's own foreground color. Falls
-- back silently to the foreground color if the sampled color doesn't have
-- enough contrast against the background to stay reliably scannable.
ALTER TABLE qr_codes
    ADD COLUMN eye_color_from_logo TINYINT(1) NOT NULL DEFAULT 0 AFTER logo_style;
