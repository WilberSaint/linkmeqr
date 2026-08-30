-- The name/description/save-contact/share header and the text-heavy content
-- blocks (Google review, hours, testimonials) each sit in a translucent
-- panel so they stay legible over any background — previously a fixed
-- rgba(0,0,0,0.04) baked into each block's own markup, invisible over
-- anything busier than a flat color and impossible to turn up. Same idea as
-- background_fit: freeform image backgrounds made the fixed value wrong
-- often enough that it needed to become a real setting.
ALTER TABLE profile_themes
    ADD COLUMN card_color   VARCHAR(20) NOT NULL DEFAULT '#000000' AFTER background_fit,
    ADD COLUMN card_opacity DECIMAL(3,2) NOT NULL DEFAULT 0.04 AFTER card_color;
