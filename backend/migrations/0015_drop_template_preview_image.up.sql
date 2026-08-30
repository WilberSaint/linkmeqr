-- preview_image was never rendered anywhere in the app — both the admin
-- catalog and the client's template picker always synthesize a live
-- preview from the theme's own colors/logo instead, which can't go stale
-- the way a pasted screenshot URL could. Dropping the dead field rather
-- than leaving a form input that silently does nothing.
ALTER TABLE templates DROP COLUMN preview_image;
