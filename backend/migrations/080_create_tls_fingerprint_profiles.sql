-- [sqlite-converted] from PostgreSQL migration: 080_create_tls_fingerprint_profiles.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- Create tls_fingerprint_profiles table for managing TLS fingerprint templates.
-- Each profile contains ClientHello parameters to simulate specific client TLS handshake characteristics.

-- [sqlite] skipped SET LOCAL

-- [sqlite] skipped SET LOCAL


CREATE TABLE IF NOT EXISTS tls_fingerprint_profiles (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    name         VARCHAR(100) NOT NULL UNIQUE,
    description  TEXT,
    enable_grease BOOLEAN     NOT NULL DEFAULT false,
    cipher_suites        TEXT,
    curves               TEXT,
    point_formats        TEXT,
    signature_algorithms TEXT,
    alpn_protocols       TEXT,
    supported_versions   TEXT,
    key_share_groups     TEXT,
    psk_modes            TEXT,
    extensions           TEXT,
    created_at DATETIME  NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME  NOT NULL DEFAULT (datetime('now'))
);

-- [sqlite] skipped COMMENT ON table

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

-- [sqlite] skipped COMMENT ON column

