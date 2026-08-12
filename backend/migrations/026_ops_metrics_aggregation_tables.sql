-- [sqlite-converted] from PostgreSQL migration: 026_ops_metrics_aggregation_tables.sql
-- Ops monitoring: pre-aggregation tables for dashboard queries (schema only).

CREATE TABLE IF NOT EXISTS ops_metrics_hourly (
    bucket_start DATETIME NOT NULL,
    platform VARCHAR(50) NOT NULL,
    request_count BIGINT NOT NULL DEFAULT 0,
    success_count BIGINT NOT NULL DEFAULT 0,
    error_count BIGINT NOT NULL DEFAULT 0,
    error_4xx_count BIGINT NOT NULL DEFAULT 0,
    error_5xx_count BIGINT NOT NULL DEFAULT 0,
    timeout_count BIGINT NOT NULL DEFAULT 0,
    avg_latency_ms REAL,
    p99_latency_ms REAL,
    error_rate REAL NOT NULL DEFAULT 0,
    computed_at DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (bucket_start, platform)
);

CREATE INDEX IF NOT EXISTS idx_ops_metrics_hourly_platform_bucket_start
    ON ops_metrics_hourly (platform, bucket_start DESC);

CREATE TABLE IF NOT EXISTS ops_metrics_daily (
    bucket_date DATE NOT NULL,
    platform VARCHAR(50) NOT NULL,
    request_count BIGINT NOT NULL DEFAULT 0,
    success_count BIGINT NOT NULL DEFAULT 0,
    error_count BIGINT NOT NULL DEFAULT 0,
    error_4xx_count BIGINT NOT NULL DEFAULT 0,
    error_5xx_count BIGINT NOT NULL DEFAULT 0,
    timeout_count BIGINT NOT NULL DEFAULT 0,
    avg_latency_ms REAL,
    p99_latency_ms REAL,
    error_rate REAL NOT NULL DEFAULT 0,
    computed_at DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (bucket_date, platform)
);

CREATE INDEX IF NOT EXISTS idx_ops_metrics_daily_platform_bucket_date
    ON ops_metrics_daily (platform, bucket_date DESC);
