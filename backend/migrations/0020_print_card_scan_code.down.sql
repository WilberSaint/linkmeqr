ALTER TABLE print_cards
    DROP KEY uq_print_cards_scan_code,
    DROP COLUMN scan_code;
