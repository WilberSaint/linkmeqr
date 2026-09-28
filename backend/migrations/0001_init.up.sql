-- LinkMeQR schema (SQLite).
--
-- Consolidates what used to be 24 incremental MySQL migrations into the
-- schema they had arrived at. Conventions:
--   * UUIDs as TEXT (generated in application code).
--   * MySQL ENUMs become TEXT with a CHECK constraint.
--   * JSON columns are TEXT; the application reads/writes them as strings.
--   * DATETIME columns keep that declared type so the driver scans them into
--     time.Time. Everything is stored in UTC.
--   * MySQL's ON UPDATE CURRENT_TIMESTAMP is emulated with AFTER UPDATE
--     triggers at the end of this file.
-- Foreign keys are enforced per connection (PRAGMA foreign_keys=ON in the DSN).

-- ============================================================
-- users: both ADMIN and CLIENT accounts
-- ============================================================
CREATE TABLE users (
    id              TEXT     NOT NULL PRIMARY KEY,
    email           TEXT     NOT NULL,
    password_hash   TEXT     NOT NULL,
    role            TEXT     NOT NULL DEFAULT 'CLIENT' CHECK (role IN ('ADMIN','CLIENT')),
    full_name       TEXT     NOT NULL,
    phone           TEXT     NULL,
    is_active       INTEGER  NOT NULL DEFAULT 1,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX uq_users_email ON users (email);
CREATE INDEX idx_users_role ON users (role);

-- ============================================================
-- refresh_tokens: JWT refresh token store (allows revocation)
-- ============================================================
CREATE TABLE refresh_tokens (
    id              TEXT     NOT NULL PRIMARY KEY,
    user_id         TEXT     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash      TEXT     NOT NULL,
    expires_at      DATETIME NOT NULL,
    revoked_at      DATETIME NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_refresh_user ON refresh_tokens (user_id);
CREATE UNIQUE INDEX uq_refresh_token_hash ON refresh_tokens (token_hash);

-- ============================================================
-- templates: predefined visual templates (Minimal, Business, ...)
-- ============================================================
CREATE TABLE templates (
    id              TEXT     NOT NULL PRIMARY KEY,
    slug            TEXT     NOT NULL,
    name            TEXT     NOT NULL,
    description     TEXT     NULL,
    default_theme   TEXT     NOT NULL,
    is_active       INTEGER  NOT NULL DEFAULT 1,
    sort_order      INTEGER  NOT NULL DEFAULT 0,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX uq_templates_slug ON templates (slug);

-- ============================================================
-- media: uploaded images/files (logos, backgrounds, block images)
-- ============================================================
CREATE TABLE media (
    id              TEXT     NOT NULL PRIMARY KEY,
    owner_user_id   TEXT     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    file_name       TEXT     NOT NULL,
    file_path       TEXT     NOT NULL,
    mime_type       TEXT     NOT NULL,
    size_bytes      INTEGER  NOT NULL DEFAULT 0,
    width           INTEGER  NULL,
    height          INTEGER  NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_media_owner ON media (owner_user_id);

-- ============================================================
-- profiles: one public digital profile per client
-- ============================================================
CREATE TABLE profiles (
    id              TEXT     NOT NULL PRIMARY KEY,
    user_id         TEXT     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    slug            TEXT     NOT NULL,
    business_name   TEXT     NOT NULL,
    description     TEXT     NULL,
    logo_media_id   TEXT     NULL REFERENCES media(id) ON DELETE SET NULL,
    cover_media_id  TEXT     NULL REFERENCES media(id) ON DELETE SET NULL,
    template_id     TEXT     NULL REFERENCES templates(id) ON DELETE SET NULL,
    is_published    INTEGER  NOT NULL DEFAULT 1,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX uq_profiles_slug ON profiles (slug);
CREATE INDEX idx_profiles_user ON profiles (user_id);

-- ============================================================
-- profile_themes: visual customization for a profile (1:1)
-- ============================================================
CREATE TABLE profile_themes (
    id                    TEXT     NOT NULL PRIMARY KEY,
    profile_id            TEXT     NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    background_type       TEXT     NOT NULL DEFAULT 'color'
                                   CHECK (background_type IN ('color','gradient','pattern','image')),
    background_value      TEXT     NOT NULL DEFAULT '#ffffff',
    -- For 'image' backgrounds; background_value keeps the CSS value otherwise.
    background_media_id   TEXT     NULL REFERENCES media(id) ON DELETE SET NULL,
    -- How an 'image' background is sized against the viewport.
    background_fit        TEXT     NOT NULL DEFAULT 'cover'
                                   CHECK (background_fit IN ('cover','contain','repeat')),
    -- Translucent panel behind the header and text-heavy blocks.
    card_color            TEXT     NOT NULL DEFAULT '#000000',
    card_opacity          REAL     NOT NULL DEFAULT 0.04,
    primary_color         TEXT     NOT NULL DEFAULT '#111827',
    secondary_color       TEXT     NOT NULL DEFAULT '#6366f1',
    text_color            TEXT     NOT NULL DEFAULT '#111827',
    button_text_color     TEXT     NOT NULL DEFAULT '#ffffff',
    logo_background_color TEXT     NOT NULL DEFAULT '#111827',
    logo_text_color       TEXT     NOT NULL DEFAULT '#ffffff',
    logo_display_mode     TEXT     NOT NULL DEFAULT 'initial'
                                   CHECK (logo_display_mode IN ('image','initial')),
    logo_shape            TEXT     NOT NULL DEFAULT 'circle'
                                   CHECK (logo_shape IN ('circle','rounded','square')),
    font_family           TEXT     NOT NULL DEFAULT 'Inter',
    button_style          TEXT     NOT NULL DEFAULT 'rounded'
                                   CHECK (button_style IN ('rounded','square','pill','outline')),
    button_shadow         INTEGER  NOT NULL DEFAULT 0,
    layout                TEXT     NOT NULL DEFAULT 'list' CHECK (layout IN ('list','grid')),
    extra_css_vars        TEXT     NULL,
    updated_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX uq_theme_profile ON profile_themes (profile_id);

-- ============================================================
-- profile_blocks: ordered content blocks on a profile page
-- ============================================================
CREATE TABLE profile_blocks (
    id              TEXT     NOT NULL PRIMARY KEY,
    profile_id      TEXT     NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    block_type      TEXT     NOT NULL CHECK (block_type IN (
                        'instagram','facebook','tiktok','youtube','whatsapp','phone','email',
                        'location','website','menu','catalog','image','video','text','link',
                        'google_review','gallery','hours','testimonials','map')),
    title           TEXT     NULL,
    description     TEXT     NULL,
    url             TEXT     NULL,
    icon            TEXT     NULL,
    media_id        TEXT     NULL REFERENCES media(id) ON DELETE SET NULL,
    style_overrides TEXT     NULL,
    content         TEXT     NULL,
    is_visible      INTEGER  NOT NULL DEFAULT 1,
    sort_order      INTEGER  NOT NULL DEFAULT 0,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_blocks_profile_order ON profile_blocks (profile_id, sort_order);

-- ============================================================
-- licenses: current license/subscription state per client (1:1 with user)
-- ============================================================
CREATE TABLE licenses (
    id              TEXT     NOT NULL PRIMARY KEY,
    user_id         TEXT     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status          TEXT     NOT NULL DEFAULT 'INACTIVE'
                             CHECK (status IN ('INACTIVE','ACTIVE','EXPIRED')),
    activated_at    DATETIME NULL,
    expires_at      DATETIME NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX uq_license_user ON licenses (user_id);
CREATE INDEX idx_license_status_expiry ON licenses (status, expires_at);

-- ============================================================
-- activation_codes: generated codes (individual or batch)
-- ============================================================
CREATE TABLE activation_codes (
    id                  TEXT     NOT NULL PRIMARY KEY,
    code                TEXT     NOT NULL,
    duration_type       TEXT     NOT NULL
                                 CHECK (duration_type IN ('1_MONTH','3_MONTHS','6_MONTHS','1_YEAR','CUSTOM')),
    duration_days       INTEGER  NOT NULL,
    status              TEXT     NOT NULL DEFAULT 'UNUSED' CHECK (status IN ('UNUSED','USED','REVOKED')),
    batch_id            TEXT     NULL,
    assigned_user_id    TEXT     NULL REFERENCES users(id) ON DELETE SET NULL,
    used_by_user_id     TEXT     NULL REFERENCES users(id) ON DELETE SET NULL,
    created_by_admin_id TEXT     NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    activated_at        DATETIME NULL,
    -- Expiration resulting from this activation, informational.
    expires_at          DATETIME NULL,
    revoked_at          DATETIME NULL
);
CREATE UNIQUE INDEX uq_code ON activation_codes (code);
CREATE INDEX idx_codes_status ON activation_codes (status);
CREATE INDEX idx_codes_batch ON activation_codes (batch_id);
CREATE INDEX idx_codes_assigned ON activation_codes (assigned_user_id);

-- ============================================================
-- license_activations: full audit history of activations/renewals
-- ============================================================
CREATE TABLE license_activations (
    id                  TEXT     NOT NULL PRIMARY KEY,
    license_id          TEXT     NOT NULL REFERENCES licenses(id) ON DELETE CASCADE,
    activation_code_id  TEXT     NOT NULL REFERENCES activation_codes(id) ON DELETE RESTRICT,
    user_id             TEXT     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    duration_days_added INTEGER  NOT NULL,
    previous_expires_at DATETIME NULL,
    new_expires_at      DATETIME NOT NULL,
    activated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_activations_license ON license_activations (license_id);
CREATE INDEX idx_activations_user ON license_activations (user_id);

-- ============================================================
-- audit_logs: administrative action history
-- ============================================================
CREATE TABLE audit_logs (
    id              TEXT     NOT NULL PRIMARY KEY,
    actor_user_id   TEXT     NULL REFERENCES users(id) ON DELETE SET NULL,
    action          TEXT     NOT NULL,
    entity_type     TEXT     NOT NULL,
    entity_id       TEXT     NULL,
    metadata        TEXT     NULL,
    ip_address      TEXT     NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_audit_actor ON audit_logs (actor_user_id);
CREATE INDEX idx_audit_entity ON audit_logs (entity_type, entity_id);

-- ============================================================
-- Loyalty / stamp cards. loyalty_customers are the business's own walk-in
-- patrons — never platform users — recognized via a browser-cookie token.
-- ============================================================
CREATE TABLE loyalty_programs (
    id                     TEXT     NOT NULL PRIMARY KEY,
    user_id                TEXT     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    stamps_required        INTEGER  NOT NULL DEFAULT 10,
    -- Optional intermediate reward; non-null mid_reward_stamps enables it.
    mid_reward_stamps      INTEGER  NULL,
    mid_reward_description TEXT     NULL,
    reward_description     TEXT     NULL,
    loyalty_token          TEXT     NOT NULL,
    is_active              INTEGER  NOT NULL DEFAULT 1,
    created_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX uq_loyalty_programs_user ON loyalty_programs (user_id);
CREATE UNIQUE INDEX uq_loyalty_programs_token ON loyalty_programs (loyalty_token);

CREATE TABLE loyalty_customers (
    id              TEXT     NOT NULL PRIMARY KEY,
    user_id         TEXT     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    full_name       TEXT     NOT NULL,
    phone           TEXT     NULL,
    identity_token  TEXT     NOT NULL,
    stamps_count    INTEGER  NOT NULL DEFAULT 0,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX uq_loyalty_customers_identity ON loyalty_customers (identity_token);
CREATE INDEX idx_loyalty_customers_user ON loyalty_customers (user_id);

CREATE TABLE loyalty_stamps (
    id                  TEXT     NOT NULL PRIMARY KEY,
    loyalty_customer_id TEXT     NOT NULL REFERENCES loyalty_customers(id) ON DELETE CASCADE,
    source              TEXT     NOT NULL DEFAULT 'nfc' CHECK (source IN ('nfc','manual')),
    created_by_admin_id TEXT     NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_loyalty_stamps_customer ON loyalty_stamps (loyalty_customer_id, created_at);

-- ============================================================
-- analytics_events: profile visits and link clicks
-- ============================================================
CREATE TABLE analytics_events (
    id              TEXT     NOT NULL PRIMARY KEY,
    profile_id      TEXT     NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    event_type      TEXT     NOT NULL CHECK (event_type IN ('VIEW','BLOCK_CLICK')),
    block_id        TEXT     NULL REFERENCES profile_blocks(id) ON DELETE SET NULL,
    device_type     TEXT     NULL, -- mobile/tablet/desktop
    os_name         TEXT     NULL,
    browser_name    TEXT     NULL,
    referrer        TEXT     NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_events_profile_date ON analytics_events (profile_id, created_at);
CREATE INDEX idx_events_type ON analytics_events (event_type);

-- ============================================================
-- updated_at maintenance (MySQL's ON UPDATE CURRENT_TIMESTAMP).
-- The WHEN clause skips updates that set updated_at themselves; the inner
-- UPDATE doesn't re-fire the trigger since recursive_triggers is off.
-- ============================================================
CREATE TRIGGER trg_users_updated_at AFTER UPDATE ON users
WHEN NEW.updated_at IS OLD.updated_at
BEGIN UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;

CREATE TRIGGER trg_profiles_updated_at AFTER UPDATE ON profiles
WHEN NEW.updated_at IS OLD.updated_at
BEGIN UPDATE profiles SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;

CREATE TRIGGER trg_profile_themes_updated_at AFTER UPDATE ON profile_themes
WHEN NEW.updated_at IS OLD.updated_at
BEGIN UPDATE profile_themes SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;

CREATE TRIGGER trg_profile_blocks_updated_at AFTER UPDATE ON profile_blocks
WHEN NEW.updated_at IS OLD.updated_at
BEGIN UPDATE profile_blocks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;

CREATE TRIGGER trg_licenses_updated_at AFTER UPDATE ON licenses
WHEN NEW.updated_at IS OLD.updated_at
BEGIN UPDATE licenses SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;

CREATE TRIGGER trg_loyalty_programs_updated_at AFTER UPDATE ON loyalty_programs
WHEN NEW.updated_at IS OLD.updated_at
BEGIN UPDATE loyalty_programs SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;

CREATE TRIGGER trg_loyalty_customers_updated_at AFTER UPDATE ON loyalty_customers
WHEN NEW.updated_at IS OLD.updated_at
BEGIN UPDATE loyalty_customers SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id; END;
