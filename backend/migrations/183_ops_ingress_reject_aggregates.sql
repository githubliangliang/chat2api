-- [sqlite-converted] from PostgreSQL migration: 183_ops_ingress_reject_aggregates.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- [sqlite] skipped SET LOCAL

-- [sqlite] skipped SET LOCAL


CREATE TABLE IF NOT EXISTS ops_ingress_reject_aggregates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    bucket_start TEXT NOT NULL,
    reject_reason VARCHAR(64) NOT NULL,
    route_family VARCHAR(64) NOT NULL,
    protocol VARCHAR(32) NOT NULL,
    client_ip TEXT NOT NULL,
    user_id BIGINT NOT NULL DEFAULT 0,
    api_key_id BIGINT NOT NULL DEFAULT 0,
    request_count BIGINT NOT NULL DEFAULT 0,
    first_seen TEXT NOT NULL,
    last_seen TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    CONSTRAINT ops_ingress_reject_aggregates_dimensions_unique UNIQUE
        (bucket_start, reject_reason, route_family, protocol, client_ip, user_id, api_key_id)
);

CREATE INDEX IF NOT EXISTS idx_ops_ingress_reject_aggregates_bucket
    ON ops_ingress_reject_aggregates (bucket_start DESC);
CREATE INDEX IF NOT EXISTS idx_ops_ingress_reject_aggregates_reason_bucket
    ON ops_ingress_reject_aggregates (reject_reason, bucket_start DESC);
CREATE INDEX IF NOT EXISTS idx_ops_ingress_reject_aggregates_ip_bucket
    ON ops_ingress_reject_aggregates (client_ip, bucket_start DESC);
