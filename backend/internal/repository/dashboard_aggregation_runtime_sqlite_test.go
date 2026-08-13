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

func openDashboardAggregationRuntimeSQLite(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_time_format=sqlite", t.Name()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	require.NoError(t, ApplyMigrations(context.Background(), db))
	return db
}

func TestDashboardAggregationAcceptsLegacyGoTimestampOnSQLite(t *testing.T) {
	db := openDashboardAggregationRuntimeSQLite(t)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `INSERT INTO usage_logs (
		id, user_id, api_key_id, account_id, model, input_tokens, output_tokens,
		total_cost, actual_cost, created_at
	) VALUES (99001, 99001, 99001, 99001, 'gpt-5', 10, 5, 1, 1,
	          '2026-08-12 22:40:01.569488059 +0800 CST m=+2569.755356883')`)
	require.NoError(t, err)

	repo := newDashboardAggregationRepositoryWithSQL(db)
	start := time.Date(2026, 8, 12, 0, 0, 0, 0, time.Local)
	require.NoError(t, repo.AggregateRange(ctx, start, start.Add(24*time.Hour)))

	var bucket time.Time
	var users, requests int64
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT h.bucket_start, h.active_users, h.total_requests
		FROM usage_dashboard_hourly h
		WHERE h.total_requests > 0
	`).Scan(&bucket, &users, &requests))
	require.Equal(t, time.Date(2026, 8, 12, 22, 0, 0, 0, time.UTC), bucket.UTC())
	require.Equal(t, int64(1), users)
	require.Equal(t, int64(1), requests)
}

func TestDashboardAggregationWatermarkScansSQLiteText(t *testing.T) {
	db := openDashboardAggregationRuntimeSQLite(t)
	repo := newDashboardAggregationRepositoryWithSQL(db)

	got, err := repo.GetAggregationWatermark(context.Background())
	require.NoError(t, err)
	require.Equal(t, time.Unix(0, 0).UTC(), got)

	want := time.Date(2026, 8, 14, 1, 30, 0, 0, time.UTC)
	require.NoError(t, repo.UpdateAggregationWatermark(context.Background(), want))
	got, err = repo.GetAggregationWatermark(context.Background())
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestUsageTrendSkipsUnparseableSQLiteDate(t *testing.T) {
	db := openDashboardAggregationRuntimeSQLite(t)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `INSERT INTO usage_logs (
		id, user_id, api_key_id, account_id, model, input_tokens, output_tokens,
		total_cost, actual_cost, created_at
	) VALUES
		(99002, 99002, 99002, 99002, 'gpt-5', 10, 5, 1, 1,
		 '2026-08-12 22:40:01.569488059 +0800 CST m=+2569.755356883'),
		(99003, 99002, 99002, 99002, 'gpt-5', 99, 99, 9, 9, 'not-a-time')`)
	require.NoError(t, err)

	repo := newUsageLogRepositoryWithSQL(nil, db)
	start := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	trend, err := repo.GetUserUsageTrendByUserID(ctx, 99002, start, start.Add(24*time.Hour), "hour")
	require.NoError(t, err)
	require.Len(t, trend, 1)
	require.Equal(t, "2026-08-12 22:00", trend[0].Date)
	require.Equal(t, int64(1), trend[0].Requests)
}
