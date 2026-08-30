ALTER TABLE analytics_events
    DROP FOREIGN KEY fk_events_print_card,
    DROP KEY idx_events_print_card,
    DROP COLUMN qr_slot,
    DROP COLUMN print_card_id,
    MODIFY COLUMN event_type ENUM('VIEW','BLOCK_CLICK') NOT NULL;
