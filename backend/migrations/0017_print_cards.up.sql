-- Printable marketing cards (review-us-on-Google, follow-us, scan-the-menu,
-- loyalty-card, etc.) a business can design and export, reusing their own
-- QR styling and brand theme. qr_target_type/value say what URL the card's
-- QR encodes; layout_key selects which built-in layout renders the rest.

CREATE TABLE print_cards (
    id                CHAR(36)      NOT NULL PRIMARY KEY,
    user_id           CHAR(36)      NOT NULL,
    layout_key        VARCHAR(64)   NOT NULL,
    title             VARCHAR(150)  NULL,
    size_preset       VARCHAR(32)   NOT NULL,
    qr_target_type    VARCHAR(20)   NOT NULL,
    qr_target_value   VARCHAR(2048) NULL,
    color_overrides   JSON          NULL,
    content           JSON          NOT NULL,
    created_at        DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_print_cards_user (user_id),
    CONSTRAINT fk_print_cards_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
