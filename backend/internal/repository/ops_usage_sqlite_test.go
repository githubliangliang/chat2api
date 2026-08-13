package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func openOpsUsageSQLite(t *testing.T) *sql.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(1)&_time_format=sqlite", t.Name())
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	require.NoError(t, ApplyMigrations(context.Background(), db))
	return db
}

func TestOpsUpsertHourlyMetricsUsesSQLiteAggregation(t *testing.T) {
	db := openOpsUsageSQLite(t)
	ctx := context.Background()
	start := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	seedOpsUsageSQLite(t, db, start)

	repo := &opsRepository{db: db}
	require.NoError(t, repo.UpsertHourlyMetrics(ctx, start, end))

	rows, err := db.QueryContext(ctx, `
		SELECT platform, group_id, success_count, error_count_total, token_consumed,
		       duration_p50_ms, ttft_p50_ms
		FROM ops_metrics_hourly
		ORDER BY platform IS NOT NULL, group_id IS NOT NULL
	`)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()

	type metric struct {
		platform sql.NullString
		groupID  sql.NullInt64
		success  int64
		errors   int64
		tokens   int64
		duration int
		ttft     int
	}
	var got []metric
	for rows.Next() {
		var item metric
		require.NoError(t, rows.Scan(&item.platform, &item.groupID, &item.success, &item.errors, &item.tokens, &item.duration, &item.ttft))
		got = append(got, item)
	}
	require.NoError(t, rows.Err())
	require.Len(t, got, 3)
	for _, item := range got {
		require.Equal(t, int64(1), item.success)
		require.Equal(t, int64(1), item.errors)
		require.Equal(t, int64(35), item.tokens)
		require.Equal(t, 100, item.duration)
		require.Equal(t, 40, item.ttft)
	}
	require.NoError(t, repo.UpsertDailyMetrics(ctx, start, end))
	var dailySuccess, dailyErrors, dailyTokens int64
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT success_count, error_count_total, token_consumed
		FROM ops_metrics_daily
		WHERE platform IS NULL AND group_id IS NULL
	`).Scan(&dailySuccess, &dailyErrors, &dailyTokens))
	require.Equal(t, int64(1), dailySuccess)
	require.Equal(t, int64(1), dailyErrors)
	require.Equal(t, int64(35), dailyTokens)

	latestHourly, found, err := repo.GetLatestHourlyBucketStart(ctx)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, start, latestHourly)
	latestDaily, found, err := repo.GetLatestDailyBucketDate(ctx)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC), latestDaily)
}

func TestOpsAndUsageQueriesRunOnSQLite(t *testing.T) {
	db := openOpsUsageSQLite(t)
	ctx := context.Background()
	start := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	seedOpsUsageSQLite(t, db, start)

	opsRepo := &opsRepository{db: db}
	filter := &service.OpsDashboardFilter{StartTime: start, EndTime: end}
	duration, ttft, count, err := opsRepo.queryUsageLatency(ctx, filter, start, end)
	require.NoError(t, err)
	require.Equal(t, 100, *duration.P50)
	require.Equal(t, 40, *ttft.P50)
	require.Equal(t, int64(1), count)

	qps, tps, err := opsRepo.queryPeakRates(ctx, filter, start, end)
	require.NoError(t, err)
	require.Equal(t, 0.0, qps)
	require.Equal(t, 0.6, tps)

	realtime, err := opsRepo.GetRealtimeTrafficSummary(ctx, filter)
	require.NoError(t, err)
	require.Equal(t, 0.6, realtime.TPS.Peak)

	trend, err := opsRepo.GetThroughputTrend(ctx, filter, 60)
	require.NoError(t, err)
	require.NotEmpty(t, trend.Points)
	var switches int64
	for _, point := range trend.Points {
		switches += point.SwitchCount
	}
	require.Equal(t, int64(1), switches)

	errorTrend, err := opsRepo.GetErrorTrend(ctx, filter, 60)
	require.NoError(t, err)
	require.NotEmpty(t, errorTrend.Points)
	var errors int64
	for _, point := range errorTrend.Points {
		errors += point.ErrorCountTotal
	}
	require.Equal(t, int64(1), errors)

	startCopy, endCopy := start, end
	details, total, err := opsRepo.ListRequestDetails(ctx, &service.OpsRequestDetailFilter{
		StartTime: &startCopy, EndTime: &endCopy, Page: 1, PageSize: 10,
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, details, 2)

	openAI, err := opsRepo.GetOpenAITokenStats(ctx, &service.OpsOpenAITokenStatsFilter{
		StartTime: start, EndTime: end, Page: 1, PageSize: 10,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), openAI.Total)
	require.Equal(t, "gpt-5", openAI.Items[0].Model)

	usageRepo := newUsageLogRepositoryWithSQL(nil, db)
	usageTrend, err := usageRepo.GetUserUsageTrendByUserID(ctx, 9001, start, end, "hour")
	require.NoError(t, err)
	require.Len(t, usageTrend, 1)
	require.Equal(t, "2026-08-13 10:00", usageTrend[0].Date)

	dashboard, err := usageRepo.GetDashboardStatsWithRange(ctx, start, end)
	require.NoError(t, err)
	require.Equal(t, int64(1), dashboard.TotalRequests)
}

func seedOpsUsageSQLite(t *testing.T, db *sql.DB, start time.Time) {
	t.Helper()
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `
		INSERT INTO users (id, email, password_hash) VALUES (9001, 'ops@example.com', 'hash');
		INSERT INTO groups (id, name, platform) VALUES (9001, 'Ops SQLite OpenAI', 'openai');
		INSERT INTO accounts (id, name, platform, type, credentials) VALUES (9001, 'Ops SQLite OpenAI', 'openai', 'apikey', '{}');
		INSERT INTO api_keys (id, user_id, key, name, group_id) VALUES (9001, 9001, 'sk-ops-sqlite', 'ops', 9001);
	`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO usage_logs (
			user_id, api_key_id, account_id, group_id, model,
			input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens,
			duration_ms, first_token_ms, total_cost, actual_cost, created_at
		) VALUES (9001, 9001, 9001, 9001, 'gpt-5', 10, 20, 3, 2, 100, 40, 0.1, 0.08, $1)
	`, start.Add(5*time.Minute))
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO ops_error_logs (
			group_id, platform, error_phase, error_type, status_code,
			is_business_limited, is_count_tokens, error_owner, upstream_status_code,
			upstream_errors, created_at
		) VALUES (9001, 'openai', 'upstream', 'status', 500, FALSE, FALSE, 'provider', 500,
		          '[{"kind":"failover:account"}]', $1)
	`, start.Add(6*time.Minute))
	require.NoError(t, err)
}
