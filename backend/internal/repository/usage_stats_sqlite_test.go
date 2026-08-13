package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestUsageLogDailyAndAccountStatsRunOnSQLite(t *testing.T) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_time_format=sqlite", t.Name()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	require.NoError(t, ApplyMigrations(context.Background(), db))

	start := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	end := start.Add(48 * time.Hour)
	_, err = db.Exec(`INSERT INTO usage_logs (
		id, user_id, api_key_id, account_id, group_id, request_id, requested_model, model,
		input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens,
		total_cost, actual_cost, duration_ms, created_at
	) VALUES
		(1, 10, 20, 30, 40, 'req-1', 'gpt-5', 'gpt-5', 10, 5, 2, 1, 1.25, 1.0, 100, ?),
		(2, 10, 20, 30, 40, 'req-2', 'gpt-5', 'gpt-5', 20, 8, 3, 2, 2.50, 2.0, 200, ?)`,
		start.Add(time.Hour), start.Add(25*time.Hour))
	require.NoError(t, err)

	repo := newUsageLogRepositoryWithSQL(nil, db)
	daily, err := repo.GetDailyStatsAggregated(context.Background(), 10, start, end)
	require.NoError(t, err)
	require.Len(t, daily, 2)
	require.Equal(t, "2026-08-12", daily[0]["date"])
	require.Equal(t, "2026-08-13", daily[1]["date"])

	account, err := repo.GetAccountUsageStats(context.Background(), 30, start, end)
	require.NoError(t, err)
	require.Len(t, account.History, 2)
	require.Equal(t, "2026-08-12", account.History[0].Date)
	require.Equal(t, "2026-08-13", account.History[1].Date)
}
