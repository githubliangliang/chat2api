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
}
