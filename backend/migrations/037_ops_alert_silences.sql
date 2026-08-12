-- [sqlite-converted] from PostgreSQL migration: 037_ops_alert_silences.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- -- +goose Up
-- -- +goose StatementBegin
-- Ops alert silences: scoped (rule_id + platform + group_id + region)

CREATE TABLE IF NOT EXISTS ops_alert_silences (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    rule_id BIGINT NOT NULL,
    platform VARCHAR(64) NOT NULL,
    group_id BIGINT,
    region VARCHAR(64),

    until TEXT NOT NULL,
    reason TEXT,

    created_by BIGINT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_ops_alert_silences_lookup
    ON ops_alert_silences (rule_id, platform, group_id, region, until);

-- -- +goose StatementEnd

-- -- [sqlite] skipped +goose Down section
