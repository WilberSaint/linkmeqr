-- The print-card designer moves from "pick one of six hardcoded layouts and
-- fill in its fields" to a free-form element tree: every card now stores its
-- own {canvas, background, elements[]} document, and the exporter renders
-- that tree instead of running layout math in Go.
--
-- layout_key / content / color_overrides are deliberately LEFT IN PLACE and
-- still NOT NULL. They are no longer read for rendering, but a card whose
-- tree has not been backfilled yet (see cmd/migratelayouts) still needs them
-- to be seeded from, and dropping them here would make the backfill
-- unrunnable and the migration irreversible.

ALTER TABLE print_cards
    ADD COLUMN layout         JSON NULL           AFTER content,
    ADD COLUMN layout_version INT  NOT NULL DEFAULT 0 AFTER layout;

-- One row per saved revision of a card's tree. The card's own
-- print_cards.layout always mirrors the highest version here; this table
-- exists so a design can be restored after a bad edit, which matters more
-- than usual because the artifact being designed is physically printed and
-- sold.
CREATE TABLE print_card_layout_versions (
    id            CHAR(36) NOT NULL PRIMARY KEY,
    print_card_id CHAR(36) NOT NULL,
    version       INT      NOT NULL,
    layout        JSON     NOT NULL,
    -- Nullable because the backfill writes version 1 with no admin behind
    -- it; every interactive save records who made it.
    created_by    CHAR(36) NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uq_print_card_layout_version (print_card_id, version),
    KEY idx_print_card_layout_versions_card (print_card_id),
    CONSTRAINT fk_layout_versions_card FOREIGN KEY (print_card_id) REFERENCES print_cards(id) ON DELETE CASCADE,
    CONSTRAINT fk_layout_versions_user FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
