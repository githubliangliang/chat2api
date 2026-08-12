-- [sqlite-converted] from PostgreSQL migration: 066_add_scheduled_test_tables.sql
-- Auto-converted for SQLite dialect. Review complex logic if needed.
-- 066_add_scheduled_test_tables.sql
-- Scheduled account test plans and results

CREATE TABLE IF NOT EXISTS scheduled_test_plans (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id      BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    model_id        VARCHAR(100) NOT NULL DEFAULT '',
    cron_expression VARCHAR(100) NOT NULL DEFAULT '*/30 * * * *',
    enabled         BOOLEAN NOT NULL DEFAULT true,
    max_results     INT NOT NULL DEFAULT 50,
    last_run_at DATETIME,
    next_run_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_stp_account_id ON scheduled_test_plans(account_id);
CREATE INDEX IF NOT EXISTS idx_stp_enabled_next_run ON scheduled_test_plans(enabled, next_run_at) WHERE enabled = true;

CREATE TABLE IF NOT EXISTS scheduled_test_results (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_id       BIGINT NOT NULL REFERENCES scheduled_test_plans(id) ON DELETE CASCADE,
    status        VARCHAR(20) NOT NULL DEFAULT 'success',
    response_text TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    latency_ms    BIGINT NOT NULL DEFAULT 0,
    started_at DATETIME NOT NULL DEFAULT (datetime('now')),
    finished_at DATETIME NOT NULL DEFAULT (datetime('now')),
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_str_plan_created ON scheduled_test_results(plan_id, created_at DESC);
