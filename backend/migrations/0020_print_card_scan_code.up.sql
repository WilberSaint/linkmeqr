ALTER TABLE print_cards ADD COLUMN scan_code VARCHAR(12) NULL AFTER id;

UPDATE print_cards SET scan_code = SUBSTRING(REPLACE(UUID(), '-', ''), 1, 10) WHERE scan_code IS NULL;

ALTER TABLE print_cards
    MODIFY COLUMN scan_code VARCHAR(12) NOT NULL,
    ADD UNIQUE KEY uq_print_cards_scan_code (scan_code);
