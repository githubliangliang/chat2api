package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// EnsureSQLiteAuxTables creates tables that are not covered by Ent Schema.Create
// but are required by repository code that runs raw SQL (ported from PG migrations).
// Safe to call on every startup (IF NOT EXISTS).
func EnsureSQLiteAuxTables(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("nil sql db")
	}
	stmts := []string{
		// Required by login/user lookup (loadAllowedGroups). Migration 007 was
		// historically converted to a SQLite no-op; keep a safety net here.
		`CREATE TABLE IF NOT EXISTS user_allowed_groups (
			user_id INTEGER NOT NULL,
			group_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			PRIMARY KEY (user_id, group_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_allowed_groups_group_id
			ON user_allowed_groups (group_id)`,
		// Required by userRepository.GetUserAvatar on every authenticated request.
		`CREATE TABLE IF NOT EXISTS user_avatars (
			user_id INTEGER NOT NULL PRIMARY KEY,
			storage_provider TEXT NOT NULL DEFAULT '',
			storage_key TEXT NOT NULL DEFAULT '',
			url TEXT NOT NULL DEFAULT '',
			content_type TEXT NOT NULL DEFAULT '',
			byte_size INTEGER NOT NULL DEFAULT 0,
			sha256 TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL DEFAULT ''
		)`,
		// Optional but referenced by identity/bootstrap flows.
		`CREATE TABLE IF NOT EXISTS user_provider_default_grants (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			provider_type TEXT NOT NULL,
			grant_reason TEXT NOT NULL DEFAULT 'first_bind',
			granted_at TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS user_provider_default_grants_user_provider_reason_key
			ON user_provider_default_grants (user_id, provider_type, grant_reason)`,
		// Admin user list / per-group rate UI.
		`CREATE TABLE IF NOT EXISTS user_group_rate_multipliers (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			group_id INTEGER NOT NULL,
			rate_multiplier REAL NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS user_group_rate_multipliers_user_group_key
			ON user_group_rate_multipliers (user_id, group_id)`,
		// Scheduler outbox: the snapshot poller consumes these on SQLite too.
		// Existing DBs may carry created_at TEXT from older builds; readers must
		// stay column-type agnostic (see scanSchedulerOutboxTime).
		`CREATE TABLE IF NOT EXISTS scheduler_outbox (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			event_type TEXT NOT NULL,
			account_id INTEGER,
			group_id INTEGER,
			payload TEXT,
			dedup_key TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_scheduler_outbox_pending_dedup_key
			ON scheduler_outbox (dedup_key) WHERE dedup_key IS NOT NULL`,
		// Audit log page (best-effort; bulk COPY still unsupported on SQLite).
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			actor_user_id INTEGER,
			actor_email TEXT NOT NULL DEFAULT '',
			actor_role TEXT NOT NULL DEFAULT '',
			auth_method TEXT NOT NULL DEFAULT '',
			credential_masked TEXT NOT NULL DEFAULT '',
			action TEXT NOT NULL DEFAULT '',
			method TEXT NOT NULL DEFAULT '',
			path TEXT NOT NULL DEFAULT '',
			request_id TEXT NOT NULL DEFAULT '',
			client_ip TEXT NOT NULL DEFAULT '',
			user_agent TEXT NOT NULL DEFAULT '',
			request_body TEXT NOT NULL DEFAULT '',
			status_code INTEGER NOT NULL DEFAULT 0,
			latency_ms INTEGER NOT NULL DEFAULT 0,
			extra TEXT NOT NULL DEFAULT '{}'
		)`,
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec aux ddl: %w", err)
		}
	}
	// Ent Schema.Create lags behind SQL migrations; patch critical usage/billing objects
	// that raw repository SQL depends on (especially ON CONFLICT targets).
	if err := ensureSQLiteUsageLogColumns(ctx, db); err != nil {
		return err
	}
	if err := ensureSQLiteUsageBillingObjects(ctx, db); err != nil {
		return err
	}
	return nil
}

// ensureSQLiteUsageBillingObjects creates the idempotency index and billing dedup
// tables required by usage_log_repo_insert / usage_billing_repo.
//
// Without idx_usage_logs_request_id_api_key_unique, SQLite rejects:
//
//	INSERT ... ON CONFLICT (request_id, api_key_id) DO NOTHING
//
// with "ON CONFLICT clause does not match any PRIMARY KEY or UNIQUE constraint",
// so API calls succeed but usage_logs stay empty.
func ensureSQLiteUsageBillingObjects(ctx context.Context, db *sql.DB) error {
	stmts := []string{
		// Normalize empty request_id before unique index (matches migration 027 intent).
		// On Ent schemas request_id is NOT NULL, so only '' rows are an issue.
		`UPDATE usage_logs SET request_id = NULL WHERE request_id = ''`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_usage_logs_request_id_api_key_unique
			ON usage_logs (request_id, api_key_id)`,
		`CREATE TABLE IF NOT EXISTS usage_billing_dedup (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			request_id VARCHAR(255) NOT NULL,
			api_key_id BIGINT NOT NULL,
			request_fingerprint VARCHAR(64) NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_usage_billing_dedup_request_api_key
			ON usage_billing_dedup (request_id, api_key_id)`,
		`CREATE TABLE IF NOT EXISTS usage_billing_dedup_archive (
			request_id VARCHAR(255) NOT NULL,
			api_key_id BIGINT NOT NULL,
			request_fingerprint VARCHAR(64) NOT NULL,
			created_at TEXT NOT NULL,
			archived_at TEXT NOT NULL DEFAULT (datetime('now')),
			PRIMARY KEY (request_id, api_key_id)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			msg := strings.ToLower(err.Error())
			// usage_logs may not exist yet on brand-new installs before migrations.
			if strings.Contains(msg, "no such table") {
				continue
			}
			// Ent schemas mark request_id NOT NULL — skipping NULL normalize is fine.
			if strings.Contains(msg, "not null") && strings.Contains(stmt, "request_id = NULL") {
				continue
			}
			return fmt.Errorf("ensure sqlite usage/billing objects: %w\nstmt: %s", err, stmt)
		}
	}
	return nil
}

func ensureSQLiteUsageLogColumns(ctx context.Context, db *sql.DB) error {
	cols := []struct {
		name string
		ddl  string
	}{
		{"image_output_tokens", "INTEGER NOT NULL DEFAULT 0"},
		{"image_output_cost", "REAL NOT NULL DEFAULT 0"},
		{"image_input_tokens", "INTEGER NOT NULL DEFAULT 0"},
		{"image_input_cost", "REAL NOT NULL DEFAULT 0"},
		{"request_type", "INTEGER NOT NULL DEFAULT 0"},
		{"openai_ws_mode", "INTEGER NOT NULL DEFAULT 0"},
		{"service_tier", "TEXT"},
		{"reasoning_effort", "TEXT"},
		{"inbound_endpoint", "TEXT"},
		{"upstream_endpoint", "TEXT"},
		{"account_stats_cost", "REAL"},
		{"session_id", "TEXT"},
	}
	for _, col := range cols {
		// SQLite: ADD COLUMN IF NOT EXISTS supported in recent versions; ignore duplicate errors.
		_, err := db.ExecContext(ctx, "ALTER TABLE usage_logs ADD COLUMN "+col.name+" "+col.ddl)
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			// Table might not exist yet in brand-new DBs before Schema.Create — ignore.
			if strings.Contains(strings.ToLower(err.Error()), "no such table") {
				continue
			}
			return fmt.Errorf("add usage_logs.%s: %w", col.name, err)
		}
	}
	return nil
}

func isMissingTableOrRelationError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	// Postgres: relation "x" does not exist
	// SQLite: no such table: x
	return (strings.Contains(s, "does not exist") && strings.Contains(s, "relation")) ||
		strings.Contains(s, "no such table")
}
