package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestOpsCleanupExecutorRunsOnSQLite(t *testing.T) {
	db, err := sql.Open("sqlite", "file:ops-cleanup-executor?mode=memory&cache=shared&_time_format=sqlite")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`CREATE TABLE ops_metrics_daily (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		bucket_date DATE NOT NULL
	)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO ops_metrics_daily(bucket_date) VALUES (?), (?), (?)`,
		"2026-08-01", "2026-08-02", "2026-08-12")
	require.NoError(t, err)

	deleted, err := deleteOldRowsByID(context.Background(), db, "ops_metrics_daily", "bucket_date",
		time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC), 1, true)
	require.NoError(t, err)
	require.EqualValues(t, 2, deleted)

	deleted, err = truncateOpsTable(context.Background(), db, "ops_metrics_daily")
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)
	var remaining int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM ops_metrics_daily`).Scan(&remaining))
	require.Zero(t, remaining)
}
