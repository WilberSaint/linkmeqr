-- Loyalty/stamp-card system. loyalty_customers are the business's own
-- walk-in patrons — a lightweight identity separate from the platform's
-- users table (they never log in), recognized across visits via a token
-- stored in a browser cookie.

CREATE TABLE loyalty_programs (
    id                  CHAR(36)     NOT NULL PRIMARY KEY,
    user_id             CHAR(36)     NOT NULL,
    stamps_required     INT          NOT NULL DEFAULT 10,
    reward_description  VARCHAR(255) NULL,
    loyalty_token       VARCHAR(40)  NOT NULL,
    is_active           TINYINT(1)   NOT NULL DEFAULT 1,
    created_at          DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_loyalty_programs_user (user_id),
    UNIQUE KEY uq_loyalty_programs_token (loyalty_token),
    CONSTRAINT fk_loyalty_programs_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE loyalty_customers (
    id              CHAR(36)     NOT NULL PRIMARY KEY,
    user_id         CHAR(36)     NOT NULL,
    full_name       VARCHAR(150) NOT NULL,
    phone           VARCHAR(30)  NULL,
    identity_token  CHAR(32)     NOT NULL,
    stamps_count    INT          NOT NULL DEFAULT 0,
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_loyalty_customers_identity (identity_token),
    KEY idx_loyalty_customers_user (user_id),
    CONSTRAINT fk_loyalty_customers_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE loyalty_stamps (
    id                  CHAR(36)   NOT NULL PRIMARY KEY,
    loyalty_customer_id CHAR(36)   NOT NULL,
    source              ENUM('nfc','manual') NOT NULL DEFAULT 'nfc',
    created_by_admin_id CHAR(36)   NULL,
    created_at          DATETIME   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_loyalty_stamps_customer (loyalty_customer_id, created_at),
    CONSTRAINT fk_loyalty_stamps_customer FOREIGN KEY (loyalty_customer_id) REFERENCES loyalty_customers(id) ON DELETE CASCADE,
    CONSTRAINT fk_loyalty_stamps_admin FOREIGN KEY (created_by_admin_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
