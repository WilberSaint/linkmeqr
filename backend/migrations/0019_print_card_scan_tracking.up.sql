ALTER TABLE analytics_events
    MODIFY COLUMN event_type ENUM('VIEW','BLOCK_CLICK','QR_SCAN') NOT NULL,
    ADD COLUMN print_card_id CHAR(36)    NULL AFTER block_id,
    ADD COLUMN qr_slot       VARCHAR(10) NULL AFTER print_card_id,
    ADD KEY idx_events_print_card (print_card_id),
    ADD CONSTRAINT fk_events_print_card FOREIGN KEY (print_card_id) REFERENCES print_cards(id) ON DELETE SET NULL;
