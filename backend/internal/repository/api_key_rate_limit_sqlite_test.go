package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestAPIKeyRateLimitWindowsSQLite(t *testing.T) {
	db, err := sql.Open("sqlite", "file:api_key_rate_limit?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`CREATE TABLE api_keys (
		id INTEGER PRIMARY KEY, usage_5h REAL NOT NULL DEFAULT 0, usage_1d REAL NOT NULL DEFAULT 0, usage_7d REAL NOT NULL DEFAULT 0,
		window_5h_start DATETIME, window_1d_start DATETIME, window_7d_start DATETIME, updated_at DATETIME, deleted_at DATETIME
	); INSERT INTO api_keys (id) VALUES (1)`)
	require.NoError(t, err)
	repo := &apiKeyRepository{sql: db}
	require.NoError(t, repo.IncrementRateLimitUsage(context.Background(), 1, 2))
	var usage float64
	require.NoError(t, db.QueryRow(`SELECT usage_5h FROM api_keys WHERE id=1`).Scan(&usage))
	require.Equal(t, 2.0, usage)
	require.NoError(t, repo.ResetRateLimitWindows(context.Background(), 1))
}
