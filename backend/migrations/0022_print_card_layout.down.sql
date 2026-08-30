DROP TABLE IF EXISTS print_card_layout_versions;

ALTER TABLE print_cards
    DROP COLUMN layout_version,
    DROP COLUMN layout;
