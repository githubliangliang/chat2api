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
