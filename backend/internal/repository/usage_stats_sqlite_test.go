package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
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

func TestBatchAPIKeyUsageStatsRunsOnSQLiteWithMultipleIDs(t *testing.T) {
	db := openUsageStatsSQLite(t)
	today := timezone.Today()
	start := today.Add(-48 * time.Hour)
	end := today.Add(2 * time.Hour)

	_, err := db.Exec(`INSERT INTO usage_logs (
		id, user_id, api_key_id, account_id, request_id, model, actual_cost, created_at
	) VALUES
		(101, 10, 20, 30, 'batch-key-before', 'gpt-5', 8.0, ?),
		(102, 10, 20, 30, 'batch-key-range', 'gpt-5', 1.0, ?),
		(103, 10, 21, 30, 'batch-key-today', 'gpt-5', 2.0, ?)`,
		start.Add(-time.Hour), start.Add(time.Hour), today.Add(time.Hour))
	require.NoError(t, err)

	repo := newUsageLogRepositoryWithSQL(nil, db)
	stats, err := repo.GetBatchAPIKeyUsageStats(
		context.Background(),
		[]int64{20, 21, 22, 21, 0, -1},
		start,
		end,
	)
	require.NoError(t, err)
	require.Len(t, stats, 3)
	require.Equal(t, 1.0, stats[20].TotalActualCost)
	require.Zero(t, stats[20].TodayActualCost)
	require.Equal(t, 2.0, stats[21].TotalActualCost)
	require.Equal(t, 2.0, stats[21].TodayActualCost)
	require.Zero(t, stats[22].TotalActualCost)
	require.Zero(t, stats[22].TodayActualCost)
}

func TestBatchUserUsageStatsRunsOnSQLiteWithMultipleIDs(t *testing.T) {
	db := openUsageStatsSQLite(t)
	today := timezone.Today()
	start := today.Add(-48 * time.Hour)
	end := today.Add(2 * time.Hour)

	_, err := db.Exec(`
		INSERT INTO groups (id, name, platform) VALUES (40, 'batch-user-openai', 'openai');
		INSERT INTO accounts (id, name, platform, type, credentials)
		VALUES (30, 'batch-user-account', 'anthropic', 'apikey', '{}');
		INSERT INTO usage_logs (
			id, user_id, api_key_id, account_id, group_id, request_id, model, actual_cost, created_at
		) VALUES
			(201, 10, 20, 30, 40, 'batch-user-range', 'gpt-5', 1.0, ?),
			(202, 10, 20, 30, 40, 'batch-user-failed', 'gpt-5', 0.0, ?),
			(203, 10, 20, 30, 40, 'batch-user-today', 'gpt-5', 2.0, ?)`,
		start.Add(time.Hour), start.Add(2*time.Hour), today.Add(time.Hour))
	require.NoError(t, err)

	repo := newUsageLogRepositoryWithSQL(nil, db)
	stats, err := repo.GetBatchUserUsageStats(
		context.Background(),
		[]int64{10, 11, 10, 0, -1},
		start,
		end,
	)
	require.NoError(t, err)
	require.Len(t, stats, 2)
	require.Equal(t, 3.0, stats[10].TotalActualCost)
	require.Equal(t, 2.0, stats[10].TodayActualCost)
	require.Equal(t, []PlatformUsage{{
		Platform:        "openai",
		TotalActualCost: 3.0,
		TodayActualCost: 2.0,
	}}, stats[10].ByPlatform)
	require.Zero(t, stats[11].TotalActualCost)
	require.Zero(t, stats[11].TodayActualCost)
	require.Empty(t, stats[11].ByPlatform)
}

func TestGetAccountWindowStatsBatchIgnoresHistoryOutsideWindow(t *testing.T) {
	db := openUsageStatsSQLite(t)
	windowStart := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	_, err := db.Exec(`INSERT INTO usage_logs (
		id, user_id, api_key_id, account_id, request_id, model, total_cost, actual_cost, created_at
	) VALUES
		(301, 10, 20, 30, 'acc-a-history', 'claude-opus-4-6', 40.0, 40.0, ?),
		(302, 10, 20, 30, 'acc-a-window', 'claude-opus-4-6', 5.0, 5.0, ?),
		(303, 10, 21, 31, 'acc-b-history', 'claude-opus-4-6', 40.0, 40.0, ?)`,
		windowStart.Add(-time.Hour), windowStart.Add(time.Hour), windowStart.Add(-time.Hour))
	require.NoError(t, err)

	repo := newUsageLogRepositoryWithSQL(nil, db)
	stats, err := repo.GetAccountWindowStatsBatch(context.Background(), []int64{30, 31}, windowStart)
	require.NoError(t, err)
	require.Equal(t, 5.0, stats[30].StandardCost)
	require.Zero(t, stats[31].StandardCost)
}

func TestGetGeminiUsageTotalsBatchIgnoresHistoryOutsideWindow(t *testing.T) {
	db := openUsageStatsSQLite(t)
	start := time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC)
	end := start.Add(5 * time.Hour)
	_, err := db.Exec(`INSERT INTO usage_logs (
		id, user_id, api_key_id, account_id, request_id, model, input_tokens, output_tokens,
		cache_creation_tokens, cache_read_tokens, actual_cost, created_at
	) VALUES
		(401, 10, 20, 30, 'gemini-a-history', 'gemini-2.5-pro', 100, 50, 0, 0, 8.0, ?),
		(402, 10, 20, 30, 'gemini-a-window', 'gemini-2.5-flash', 10, 5, 0, 0, 1.0, ?),
		(403, 10, 21, 31, 'gemini-b-history', 'gemini-2.5-pro', 100, 50, 0, 0, 8.0, ?)`,
		start.Add(-time.Hour), start.Add(time.Hour), start.Add(-time.Hour))
	require.NoError(t, err)

	repo := newUsageLogRepositoryWithSQL(nil, db)
	stats, err := repo.GetGeminiUsageTotalsBatch(context.Background(), []int64{30, 31}, start, end)
	require.NoError(t, err)
	require.Equal(t, int64(1), stats[30].FlashRequests)
	require.Zero(t, stats[30].ProRequests)
	require.Zero(t, stats[31].ProRequests)
	require.Zero(t, stats[31].FlashRequests)
}

func openUsageStatsSQLite(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_time_format=sqlite", t.Name()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	require.NoError(t, ApplyMigrations(context.Background(), db))
	return db
}
