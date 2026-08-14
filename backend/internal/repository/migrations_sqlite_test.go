package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

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
