package main

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestCleanupSQLiteDeletesMatchedRows(t *testing.T) {
	db, err := sql.Open("sqlite", "file:cleanup_sqlite?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE ops_error_logs (
		id INTEGER PRIMARY KEY,
		status_code INTEGER,
		error_message TEXT,
		error_body TEXT,
		created_at DATETIME,
		error_phase TEXT,
		account_id INTEGER,
		upstream_status_code INTEGER,
		upstream_error_message TEXT,
		upstream_error_detail TEXT
	)`)
	require.NoError(t, err)
	created := time.Now().UTC().Add(-time.Hour)
	_, err = db.Exec(`INSERT INTO ops_error_logs
		(id, status_code, error_message, error_body, created_at, error_phase)
		VALUES (1, 401, 'API key is required', '', ?, 'auth'), (2, 500, 'upstream', '', ?, 'auth')`, created, created)
	require.NoError(t, err)

	counts, scanned, matched, deleted, err := cleanup(context.Background(), db, time.Now().UTC(), 10, true)
	require.NoError(t, err)
	require.EqualValues(t, 2, scanned)
	require.EqualValues(t, 1, matched)
	require.EqualValues(t, 1, deleted)
	require.EqualValues(t, int64(1), counts["missing_key"])

	var remaining int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM ops_error_logs`).Scan(&remaining))
	require.Equal(t, 1, remaining)
}
