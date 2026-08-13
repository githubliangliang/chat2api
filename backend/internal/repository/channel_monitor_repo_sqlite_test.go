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

func TestChannelMonitorHistoryQueriesRunOnSQLite(t *testing.T) {
	db := openChannelMonitorSQLite(t)
	createChannelMonitorSQLiteTables(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	insertHistory := func(monitorID int64, model, status string, latency any, checkedAt time.Time) {
		t.Helper()
		_, err := db.ExecContext(ctx, `INSERT INTO channel_monitor_histories
			(monitor_id, model, status, latency_ms, ping_latency_ms, message, checked_at)
			VALUES ($1,$2,$3,$4,NULL,'',$5)`, monitorID, model, status, latency, checkedAt)
		require.NoError(t, err)
	}
	insertHistory(1, "primary", "failed", nil, now.Add(-2*time.Hour))
	insertHistory(1, "primary", "operational", 100, now.Add(-time.Hour))
	insertHistory(1, "extra", "degraded", 300, now.Add(-30*time.Minute))
	insertHistory(2, "primary", "error", 500, now.Add(-time.Hour))
	insertHistory(2, "other", "operational", 10, now.Add(-time.Minute))

	repo := &channelMonitorRepository{db: db}
	latest, err := repo.ListLatestPerModel(ctx, 1)
	require.NoError(t, err)
	require.Len(t, latest, 2)
	require.Equal(t, "degraded", latest[0].Status)
	require.Equal(t, "operational", latest[1].Status)

	batch, err := repo.ListLatestForMonitorIDs(ctx, []int64{1, 2})
	require.NoError(t, err)
	require.Len(t, batch[1], 2)
	require.Len(t, batch[2], 2)

	recent, err := repo.ListRecentHistoryForMonitors(ctx, []int64{1, 2}, map[int64]string{1: "primary", 2: "primary"}, 1)
	require.NoError(t, err)
	require.Len(t, recent[1], 1)
	require.Equal(t, "operational", recent[1][0].Status)
	require.Len(t, recent[2], 1)
	require.Equal(t, "error", recent[2][0].Status)

	availability, err := repo.ComputeAvailabilityForMonitors(ctx, []int64{1, 2}, 7)
	require.NoError(t, err)
	require.Len(t, availability[1], 2)
	require.Len(t, availability[2], 2)

	single, err := repo.ComputeAvailability(ctx, 1, 7)
	require.NoError(t, err)
	require.Len(t, single, 2)
}

func TestChannelMonitorRollupsWatermarkAndPruningRunOnSQLite(t *testing.T) {
	db := openChannelMonitorSQLite(t)
	createChannelMonitorSQLiteTables(t, db)
	ctx := context.Background()
	day := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	for _, status := range []string{"operational", "failed"} {
		_, err := db.ExecContext(ctx, `INSERT INTO channel_monitor_histories
			(monitor_id,model,status,message,checked_at) VALUES (1,'primary',$1,'',$2)`, status, day.Add(time.Hour))
		require.NoError(t, err)
	}
	repo := &channelMonitorRepository{db: db}
	affected, err := repo.UpsertDailyRollupsFor(ctx, day.Add(12*time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)
	var total, okCount int
	require.NoError(t, db.QueryRow(`SELECT total_checks, ok_count FROM channel_monitor_daily_rollups`).Scan(&total, &okCount))
	require.Equal(t, 2, total)
	require.Equal(t, 1, okCount)

	require.NoError(t, repo.UpdateAggregationWatermark(ctx, day))
	watermark, err := repo.LoadAggregationWatermark(ctx)
	require.NoError(t, err)
	require.NotNil(t, watermark)
	require.Equal(t, day.Format("2006-01-02"), watermark.UTC().Format("2006-01-02"))

	deleted, err := repo.DeleteRollupsBefore(ctx, day.AddDate(0, 0, 1))
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)
	deleted, err = repo.DeleteHistoryBefore(ctx, day.AddDate(0, 0, 1))
	require.NoError(t, err)
	require.Equal(t, int64(2), deleted)
}

func openChannelMonitorSQLite(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_time_format=sqlite", t.Name()))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}

func createChannelMonitorSQLiteTables(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`CREATE TABLE channel_monitor_histories (
		id INTEGER PRIMARY KEY AUTOINCREMENT, monitor_id INTEGER NOT NULL, model TEXT NOT NULL,
		status TEXT NOT NULL, latency_ms INTEGER, ping_latency_ms INTEGER, message TEXT NOT NULL,
		checked_at DATETIME NOT NULL);
	CREATE TABLE channel_monitor_daily_rollups (
		id INTEGER PRIMARY KEY AUTOINCREMENT, monitor_id INTEGER NOT NULL, model TEXT NOT NULL,
		bucket_date DATE NOT NULL, total_checks INTEGER NOT NULL, ok_count INTEGER NOT NULL,
		operational_count INTEGER NOT NULL, degraded_count INTEGER NOT NULL,
		failed_count INTEGER NOT NULL, error_count INTEGER NOT NULL,
		sum_latency_ms INTEGER NOT NULL, count_latency INTEGER NOT NULL,
		sum_ping_latency_ms INTEGER NOT NULL, count_ping_latency INTEGER NOT NULL,
		computed_at DATETIME NOT NULL, UNIQUE(monitor_id, model, bucket_date));
	CREATE TABLE channel_monitor_aggregation_watermark (
		id INTEGER PRIMARY KEY, last_aggregated_date DATE, updated_at DATETIME NOT NULL)`)
	require.NoError(t, err)
}
