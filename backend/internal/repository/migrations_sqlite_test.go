package repository

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestApplyMigrationsFreshSQLiteDatabase(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(1)&_time_format=sqlite", t.Name())
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	require.NoError(t, ApplyMigrations(context.Background(), db))

	for _, table := range []string{"users", "accounts", "api_keys", "usage_logs", "schema_migrations"} {
		var exists int
		err := db.QueryRowContext(context.Background(), `
			SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = $1
		`, table).Scan(&exists)
		require.NoError(t, err)
		require.Equalf(t, 1, exists, "expected table %s", table)
	}

	for _, column := range []string{
		"source_order_id",
		"balance_after",
		"aff_quota_after",
		"aff_frozen_quota_after",
		"aff_history_quota_after",
	} {
		var exists int
		err := db.QueryRowContext(context.Background(), `
			SELECT COUNT(*) FROM pragma_table_info('user_affiliate_ledger') WHERE name = $1
		`, column).Scan(&exists)
		require.NoError(t, err)
		require.Equalf(t, 1, exists, "expected user_affiliate_ledger.%s", column)
	}
}

// 复现「用旧版构建迁移过的 SQLite 库升级到新版」：把被改写过的 seed 迁移的 checksum
// 改回改写前的值，再跑一次迁移，必须仍然通过校验，否则线上升级会卡在启动阶段。
func TestApplyMigrationsAcceptsLegacySeedChecksumsOnSQLite(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(1)&_time_format=sqlite", t.Name())
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	require.NoError(t, ApplyMigrations(ctx, db))

	for name, rewrite := range sqliteSeedTimestampRewrites {
		res, execErr := db.ExecContext(ctx,
			`UPDATE schema_migrations SET checksum = $1 WHERE filename = $2`,
			rewrite.dbChecksum, name)
		require.NoError(t, execErr)
		affected, rowsErr := res.RowsAffected()
		require.NoError(t, rowsErr)
		require.Equalf(t, int64(1), affected, "expected schema_migrations row for %s", name)
	}

	require.NoError(t, ApplyMigrations(ctx, db), "旧 checksum 的库必须能继续启动")
}

// modernc.org/sqlite only converts TEXT values to time.Time when the declared
// column type is DATE/DATETIME/TIMESTAMP. TEXT stays a string, so Ent/database/sql
// fails with: unsupported Scan, storing driver.Value type string into type *time.Time.
func TestSQLiteTextTimestampCannotScanIntoTime(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_time_format=sqlite", t.Name())
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE accounts_scan (
		id INTEGER PRIMARY KEY,
		temp_unschedulable_until TEXT
	)`)
	require.NoError(t, err)

	until := time.Date(2026, 8, 14, 17, 23, 43, 380000000, time.FixedZone("CST", 8*3600))
	_, err = db.Exec(`INSERT INTO accounts_scan (temp_unschedulable_until) VALUES ($1)`, until)
	require.NoError(t, err)

	var got *time.Time
	err = db.QueryRow(`SELECT temp_unschedulable_until FROM accounts_scan`).Scan(&got)
	require.Error(t, err)
	require.Contains(t, err.Error(), `unsupported Scan, storing driver.Value type string into type *time.Time`)
}

func TestApplyMigrationsConvertsTimestampTextColumnsForTimeScan(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(1)&_time_format=sqlite", t.Name())
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	require.NoError(t, ApplyMigrations(ctx, db))

	for _, column := range []struct {
		table string
		name  string
	}{
		{"accounts", "temp_unschedulable_until"},
		{"accounts", "overload_until"},
		{"accounts", "session_window_start"},
		{"accounts", "session_window_end"},
		{"api_keys", "window_5h_start"},
		{"api_keys", "window_1d_start"},
		{"api_keys", "window_7d_start"},
	} {
		require.Equal(t, "DATETIME", sqliteColumnDeclType(t, db, column.table, column.name),
			"%s.%s must be DATETIME so modernc can scan into *time.Time", column.table, column.name)
	}

	until := time.Date(2026, 8, 14, 17, 23, 43, 380000000, time.FixedZone("CST", 8*3600))
	_, err = db.ExecContext(ctx, `
		INSERT INTO accounts (name, platform, type, temp_unschedulable_until, overload_until, session_window_start, session_window_end)
		VALUES ('scan-bug', 'openai', 'oauth', $1, $1, $1, $1)
	`, until)
	require.NoError(t, err)

	var gotTemp, gotOverload, gotSessionStart, gotSessionEnd *time.Time
	err = db.QueryRowContext(ctx, `
		SELECT temp_unschedulable_until, overload_until, session_window_start, session_window_end
		FROM accounts WHERE name = 'scan-bug'
	`).Scan(&gotTemp, &gotOverload, &gotSessionStart, &gotSessionEnd)
	require.NoError(t, err)
	require.NotNil(t, gotTemp)
	require.WithinDuration(t, until.UTC(), gotTemp.UTC(), time.Second)
	require.NotNil(t, gotOverload)
	require.NotNil(t, gotSessionStart)
	require.NotNil(t, gotSessionEnd)
}

func TestApplyMigrationsUpgradesExistingTextTimestampsForTimeScan(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(1)&_time_format=sqlite", t.Name())
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	require.NoError(t, applyMigrationsFS(ctx, db, migrationsWithout(t, "222_timestamp_text_to_datetime.sql")))
	require.Equal(t, "TEXT", sqliteColumnDeclType(t, db, "accounts", "temp_unschedulable_until"))

	until := time.Date(2026, 8, 14, 17, 23, 43, 380000000, time.FixedZone("CST", 8*3600))
	_, err = db.ExecContext(ctx, `
		INSERT INTO accounts (name, platform, type, temp_unschedulable_until)
		VALUES ('prod-text', 'openai', 'oauth', $1)
	`, until)
	require.NoError(t, err)

	var before *time.Time
	err = db.QueryRowContext(ctx, `SELECT temp_unschedulable_until FROM accounts WHERE name = 'prod-text'`).Scan(&before)
	require.Error(t, err)
	require.Contains(t, err.Error(), `unsupported Scan, storing driver.Value type string into type *time.Time`)

	require.NoError(t, ApplyMigrations(ctx, db))
	require.Equal(t, "DATETIME", sqliteColumnDeclType(t, db, "accounts", "temp_unschedulable_until"))

	var after *time.Time
	err = db.QueryRowContext(ctx, `SELECT temp_unschedulable_until FROM accounts WHERE name = 'prod-text'`).Scan(&after)
	require.NoError(t, err)
	require.NotNil(t, after)
	require.WithinDuration(t, until.UTC(), after.UTC(), time.Second)
}

func TestApplyMigrationsRemovesDuplicateSQLiteIndexesAndOptimizesPlanner(t *testing.T) {
	ctx := context.Background()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(1)&_time_format=sqlite", t.Name())
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	require.NoError(t, applyMigrationsFS(ctx, db, migrationsWithout(t, "224_sqlite_redundant_index_cleanup.sql")))

	duplicateIndexes := map[string]string{
		"usagelog_created_at":                     "CREATE INDEX usagelog_created_at ON usage_logs(created_at)",
		"usagelog_api_key_id_created_at":          "CREATE INDEX usagelog_api_key_id_created_at ON usage_logs(api_key_id, created_at)",
		"usagelog_subscription_id":                "CREATE INDEX usagelog_subscription_id ON usage_logs(subscription_id)",
		"usagelog_group_id":                       "CREATE INDEX usagelog_group_id ON usage_logs(group_id)",
		"usagelog_user_id_created_at":             "CREATE INDEX usagelog_user_id_created_at ON usage_logs(user_id, created_at)",
		"usagelog_model":                          "CREATE INDEX usagelog_model ON usage_logs(model)",
		"usagelog_account_id":                     "CREATE INDEX usagelog_account_id ON usage_logs(account_id)",
		"usagelog_api_key_id":                     "CREATE INDEX usagelog_api_key_id ON usage_logs(api_key_id)",
		"usagelog_user_id":                        "CREATE INDEX usagelog_user_id ON usage_logs(user_id)",
		"idx_usage_logs_billing_dedup_created_at": "CREATE INDEX IF NOT EXISTS idx_usage_logs_billing_dedup_created_at ON usage_logs(created_at)",
		"account_rate_limit_reset_at":             "CREATE INDEX account_rate_limit_reset_at ON accounts(rate_limit_reset_at)",
		"account_rate_limited_at":                 "CREATE INDEX account_rate_limited_at ON accounts(rate_limited_at)",
		"account_schedulable":                     "CREATE INDEX account_schedulable ON accounts(schedulable)",
		"account_deleted_at":                      "CREATE INDEX account_deleted_at ON accounts(deleted_at)",
		"account_last_used_at":                    "CREATE INDEX account_last_used_at ON accounts(last_used_at)",
		"account_priority":                        "CREATE INDEX account_priority ON accounts(priority)",
		"account_proxy_id":                        "CREATE INDEX account_proxy_id ON accounts(proxy_id)",
		"account_status":                          "CREATE INDEX account_status ON accounts(status)",
		"account_type":                            "CREATE INDEX account_type ON accounts(type)",
		"account_platform":                        "CREATE INDEX account_platform ON accounts(platform)",
		"accountgroup_priority":                   "CREATE INDEX accountgroup_priority ON account_groups(priority)",
		"accountgroup_group_id":                   "CREATE INDEX accountgroup_group_id ON account_groups(group_id)",
	}
	for name, statement := range duplicateIndexes {
		_, err = db.ExecContext(ctx, statement)
		require.NoErrorf(t, err, "create historical duplicate index %s", name)
	}

	_, err = db.ExecContext(ctx, `CREATE TABLE planner_probe (
		id INTEGER PRIMARY KEY,
		category TEXT NOT NULL
	)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `CREATE INDEX idx_planner_probe_category ON planner_probe(category)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		WITH RECURSIVE seq(n) AS (
			VALUES(1)
			UNION ALL
			SELECT n + 1 FROM seq WHERE n < 100
		)
		INSERT INTO planner_probe (id, category)
		SELECT n, CASE WHEN n <= 90 THEN 'common' ELSE 'rare' END FROM seq
	`)
	require.NoError(t, err)

	require.NoError(t, ApplyMigrations(ctx, db))

	for name := range duplicateIndexes {
		var exists int
		err = db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = $1
		`, name).Scan(&exists)
		require.NoError(t, err)
		require.Zerof(t, exists, "historical duplicate index %s must be removed", name)
	}

	for _, name := range []string{
		"idx_usage_logs_user_id",
		"idx_accounts_platform",
		"idx_account_groups_group_id",
	} {
		var exists int
		err = db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = $1
		`, name).Scan(&exists)
		require.NoError(t, err)
		require.Equalf(t, 1, exists, "canonical index %s must remain", name)
	}

	var plannerStats int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sqlite_stat1
		WHERE tbl = 'planner_probe' AND idx = 'idx_planner_probe_category'
	`).Scan(&plannerStats)
	require.NoError(t, err)
	require.Equal(t, 1, plannerStats)

	require.NoError(t, ApplyMigrations(ctx, db), "cleanup migration must be safe on repeated startup")
}

func migrationsWithout(t *testing.T, skip string) fs.FS {
	t.Helper()
	files, err := fs.Glob(migrations.FS, "*.sql")
	require.NoError(t, err)
	mapped := fstest.MapFS{}
	for _, name := range files {
		if name == skip {
			continue
		}
		data, readErr := fs.ReadFile(migrations.FS, name)
		require.NoError(t, readErr)
		mapped[name] = &fstest.MapFile{Data: data}
	}
	return mapped
}

func sqliteColumnDeclType(t *testing.T, db *sql.DB, table, column string) string {
	t.Helper()
	var declType string
	err := db.QueryRow(`
		SELECT type FROM pragma_table_info($1) WHERE name = $2
	`, table, column).Scan(&declType)
	require.NoError(t, err)
	return declType
}
