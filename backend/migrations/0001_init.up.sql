-- LinkMeQR initial schema (MySQL 8)
-- UUIDs stored as CHAR(36) for readability/portability (generated in application code).
-- Charset/timezone are set on the connection (DSN: charset=utf8mb4&loc=UTC), not here,
-- since golang-migrate's mysql driver runs each migration file as a single statement.

-- ============================================================
-- users: both ADMIN and CLIENT accounts
-- ============================================================
CREATE TABLE users (
    id              CHAR(36)     NOT NULL PRIMARY KEY,
    email           VARCHAR(190) NOT NULL,
    password_hash   VARCHAR(255) NOT NULL,
    role            ENUM('ADMIN','CLIENT') NOT NULL DEFAULT 'CLIENT',
    full_name       VARCHAR(150) NOT NULL,
    phone           VARCHAR(30)  NULL,
    is_active       TINYINT(1)   NOT NULL DEFAULT 1,
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_users_email (email),
    KEY idx_users_role (role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================
-- refresh_tokens: JWT refresh token store (allows revocation)
-- ============================================================
CREATE TABLE refresh_tokens (
    id              CHAR(36)     NOT NULL PRIMARY KEY,
    user_id         CHAR(36)     NOT NULL,
    token_hash      VARCHAR(255) NOT NULL,
    expires_at      DATETIME     NOT NULL,
    revoked_at      DATETIME     NULL,
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_refresh_user (user_id),
    UNIQUE KEY uq_refresh_token_hash (token_hash),
    CONSTRAINT fk_refresh_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================
-- templates: predefined visual templates (Minimal, Business, ...)
-- ============================================================
CREATE TABLE templates (
    id              CHAR(36)     NOT NULL PRIMARY KEY,
    slug            VARCHAR(60)  NOT NULL,
    name            VARCHAR(100) NOT NULL,
    description     VARCHAR(255) NULL,
    preview_image   VARCHAR(255) NULL,
    default_theme   JSON         NOT NULL,
    is_active       TINYINT(1)   NOT NULL DEFAULT 1,
    sort_order      INT          NOT NULL DEFAULT 0,
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uq_templates_slug (slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================
-- profiles: one public digital profile per client (1:1 with user, extensible to many later)
-- ============================================================
CREATE TABLE profiles (
    id              CHAR(36)     NOT NULL PRIMARY KEY,
    user_id         CHAR(36)     NOT NULL,
    slug            VARCHAR(80)  NOT NULL,
    business_name   VARCHAR(150) NOT NULL,
    description     VARCHAR(500) NULL,
    logo_media_id   CHAR(36)     NULL,
    cover_media_id  CHAR(36)     NULL,
    template_id     CHAR(36)     NULL,
    is_published    TINYINT(1)   NOT NULL DEFAULT 1,
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_profiles_slug (slug),
    KEY idx_profiles_user (user_id),
    CONSTRAINT fk_profiles_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_profiles_template FOREIGN KEY (template_id) REFERENCES templates(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================
-- profile_themes: visual customization for a profile (1:1)
-- ============================================================
CREATE TABLE profile_themes (
    id                  CHAR(36)     NOT NULL PRIMARY KEY,
    profile_id          CHAR(36)     NOT NULL,
    background_type     ENUM('color','gradient','image') NOT NULL DEFAULT 'color',
    background_value    VARCHAR(500) NOT NULL DEFAULT '#ffffff',
    primary_color       VARCHAR(20)  NOT NULL DEFAULT '#111827',
    secondary_color     VARCHAR(20)  NOT NULL DEFAULT '#6366f1',
    text_color          VARCHAR(20)  NOT NULL DEFAULT '#111827',
    font_family         VARCHAR(80)  NOT NULL DEFAULT 'Inter',
    button_style        ENUM('rounded','square','pill','outline') NOT NULL DEFAULT 'rounded',
    button_shadow       TINYINT(1)   NOT NULL DEFAULT 0,
    extra_css_vars      JSON         NULL,
    updated_at          DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_theme_profile (profile_id),
    CONSTRAINT fk_theme_profile FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================
-- profile_blocks: ordered content blocks on a profile page
-- ============================================================
CREATE TABLE profile_blocks (
    id              CHAR(36)     NOT NULL PRIMARY KEY,
    profile_id      CHAR(36)     NOT NULL,
    block_type      ENUM('instagram','facebook','tiktok','youtube','whatsapp','phone','email',
                         'location','website','menu','catalog','image','video','text','link')
                         NOT NULL,
    title           VARCHAR(150) NULL,
    description     VARCHAR(500) NULL,
    url             VARCHAR(500) NULL,
    icon            VARCHAR(60)  NULL,
    media_id        CHAR(36)     NULL,
    style_overrides JSON         NULL,
    content         JSON         NULL,
    is_visible      TINYINT(1)   NOT NULL DEFAULT 1,
    sort_order      INT          NOT NULL DEFAULT 0,
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    KEY idx_blocks_profile_order (profile_id, sort_order),
    CONSTRAINT fk_blocks_profile FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================
-- media: uploaded images/files (logos, backgrounds, block images)
-- ============================================================
CREATE TABLE media (
    id              CHAR(36)     NOT NULL PRIMARY KEY,
    owner_user_id   CHAR(36)     NOT NULL,
    file_name       VARCHAR(255) NOT NULL,
    file_path       VARCHAR(500) NOT NULL,
    mime_type       VARCHAR(100) NOT NULL,
    size_bytes      BIGINT       NOT NULL DEFAULT 0,
    width           INT          NULL,
    height          INT          NULL,
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_media_owner (owner_user_id),
    CONSTRAINT fk_media_owner FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Add FKs from profiles to media now that media exists
ALTER TABLE profiles
    ADD CONSTRAINT fk_profiles_logo_media FOREIGN KEY (logo_media_id) REFERENCES media(id) ON DELETE SET NULL,
    ADD CONSTRAINT fk_profiles_cover_media FOREIGN KEY (cover_media_id) REFERENCES media(id) ON DELETE SET NULL;

ALTER TABLE profile_blocks
    ADD CONSTRAINT fk_blocks_media FOREIGN KEY (media_id) REFERENCES media(id) ON DELETE SET NULL;

-- ============================================================
-- licenses: current license/subscription state per client (1:1 with user)
-- ============================================================
CREATE TABLE licenses (
    id              CHAR(36)     NOT NULL PRIMARY KEY,
    user_id         CHAR(36)     NOT NULL,
    status          ENUM('INACTIVE','ACTIVE','EXPIRED') NOT NULL DEFAULT 'INACTIVE',
    activated_at    DATETIME     NULL,
    expires_at      DATETIME     NULL,
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_license_user (user_id),
    KEY idx_license_status_expiry (status, expires_at),
    CONSTRAINT fk_license_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================
-- activation_codes: generated codes (individual or batch)
-- ============================================================
CREATE TABLE activation_codes (
    id              CHAR(36)     NOT NULL PRIMARY KEY,
    code            VARCHAR(32)  NOT NULL,
    duration_type   ENUM('1_MONTH','3_MONTHS','6_MONTHS','1_YEAR','CUSTOM') NOT NULL,
    duration_days   INT          NOT NULL,
    status          ENUM('UNUSED','USED','REVOKED') NOT NULL DEFAULT 'UNUSED',
    batch_id        CHAR(36)     NULL,
    assigned_user_id CHAR(36)    NULL,
    used_by_user_id CHAR(36)    NULL,
    created_by_admin_id CHAR(36) NOT NULL,
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    activated_at    DATETIME     NULL,
    expires_at      DATETIME     NULL COMMENT 'expiration resulting from this activation, informational',
    revoked_at      DATETIME     NULL,
    UNIQUE KEY uq_code (code),
    KEY idx_codes_status (status),
    KEY idx_codes_batch (batch_id),
    KEY idx_codes_assigned (assigned_user_id),
    CONSTRAINT fk_codes_assigned FOREIGN KEY (assigned_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_codes_used_by FOREIGN KEY (used_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_codes_admin FOREIGN KEY (created_by_admin_id) REFERENCES users(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================
-- license_activations: full audit history of activations/renewals
-- ============================================================
CREATE TABLE license_activations (
    id                  CHAR(36)     NOT NULL PRIMARY KEY,
    license_id          CHAR(36)     NOT NULL,
    activation_code_id  CHAR(36)     NOT NULL,
    user_id             CHAR(36)     NOT NULL,
    duration_days_added INT          NOT NULL,
    previous_expires_at DATETIME     NULL,
    new_expires_at      DATETIME     NOT NULL,
    activated_at        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_activations_license (license_id),
    KEY idx_activations_user (user_id),
    CONSTRAINT fk_activations_license FOREIGN KEY (license_id) REFERENCES licenses(id) ON DELETE CASCADE,
    CONSTRAINT fk_activations_code FOREIGN KEY (activation_code_id) REFERENCES activation_codes(id) ON DELETE RESTRICT,
    CONSTRAINT fk_activations_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================
-- qr_codes: QR customization settings tied to a profile
-- ============================================================
CREATE TABLE qr_codes (
    id                  CHAR(36)     NOT NULL PRIMARY KEY,
    profile_id          CHAR(36)     NOT NULL,
    foreground_color    VARCHAR(20)  NOT NULL DEFAULT '#000000',
    background_color    VARCHAR(20)  NOT NULL DEFAULT '#ffffff',
    module_style        ENUM('square','dots','rounded') NOT NULL DEFAULT 'square',
    eye_style           ENUM('square','circular','rounded') NOT NULL DEFAULT 'square',
    logo_media_id       CHAR(36)     NULL,
    error_correction    ENUM('L','M','Q','H') NOT NULL DEFAULT 'M',
    has_scannability_warning TINYINT(1) NOT NULL DEFAULT 0,
    created_at          DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_qr_profile (profile_id),
    CONSTRAINT fk_qr_profile FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE,
    CONSTRAINT fk_qr_logo_media FOREIGN KEY (logo_media_id) REFERENCES media(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================
-- analytics_events: profile visits and link clicks
-- ============================================================
CREATE TABLE analytics_events (
    id              CHAR(36)     NOT NULL PRIMARY KEY,
    profile_id      CHAR(36)     NOT NULL,
    event_type      ENUM('VIEW','BLOCK_CLICK') NOT NULL,
    block_id        CHAR(36)     NULL,
    device_type     VARCHAR(20)  NULL COMMENT 'mobile/tablet/desktop',
    os_name         VARCHAR(40)  NULL,
    browser_name    VARCHAR(40)  NULL,
    referrer        VARCHAR(255) NULL,
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_events_profile_date (profile_id, created_at),
    KEY idx_events_type (event_type),
    CONSTRAINT fk_events_profile FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE,
    CONSTRAINT fk_events_block FOREIGN KEY (block_id) REFERENCES profile_blocks(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================
-- audit_logs: administrative action history
-- ============================================================
CREATE TABLE audit_logs (
    id              CHAR(36)     NOT NULL PRIMARY KEY,
    actor_user_id   CHAR(36)     NULL,
    action          VARCHAR(100) NOT NULL,
    entity_type     VARCHAR(60)  NOT NULL,
    entity_id       VARCHAR(36)  NULL,
    metadata        JSON         NULL,
    ip_address      VARCHAR(45)  NULL,
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    KEY idx_audit_actor (actor_user_id),
    KEY idx_audit_entity (entity_type, entity_id),
    CONSTRAINT fk_audit_actor FOREIGN KEY (actor_user_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
