package service

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestWriteOpenAIFastPolicyBlockedResponseMarksBusinessLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	writeOpenAIFastPolicyBlockedResponse(c, &OpenAIFastBlockedError{Message: "custom fast policy block"})

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.True(t, HasOpsClientBusinessLimited(c))
	reason, ok := c.Get(OpsClientBusinessLimitedReasonKey)
	require.True(t, ok)
	require.Equal(t, OpsClientBusinessLimitedReasonLocalPolicyDenied, reason)
}

func TestOpsMetricsCollectorQueryUsageLatencySQLite(t *testing.T) {
	db, err := sql.Open("sqlite", "file:ops-metrics-latency?mode=memory&cache=shared")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE usage_logs (created_at DATETIME, duration_ms INTEGER, first_token_ms INTEGER)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO usage_logs(created_at, duration_ms, first_token_ms) VALUES
		('2026-05-26 10:10:00', 100, 10),
		('2026-05-26 10:20:00', 200, 20),
		('2026-05-26 10:30:00', 300, 30),
		('2026-05-26 10:40:00', 400, 40)`)
	require.NoError(t, err)

	collector := &OpsMetricsCollector{db: db}
	start := time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	duration, ttft, err := collector.queryUsageLatency(context.Background(), start, end)
	require.NoError(t, err)
	require.Equal(t, 250, *duration.p50)
	require.Equal(t, 397, *duration.p99)
	require.Equal(t, 25, *ttft.p50)
	require.Equal(t, 40, *ttft.max)
}

func TestOpsMetricsCollectorQueryErrorCountsExcludesCountTokens(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	collector := &OpsMetricsCollector{db: db}
	start := time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	mock.ExpectQuery(`(?s)FROM ops_error_logs\s+WHERE created_at >= \$1 AND created_at < \$2\s+AND is_count_tokens = FALSE`).
		WithArgs(start, end).
		WillReturnRows(sqlmock.NewRows([]string{
			"error_total",
			"business_limited",
			"error_sla",
			"upstream_excl",
			"upstream_429",
			"upstream_529",
		}).AddRow(int64(5), int64(2), int64(3), int64(1), int64(1), int64(1)))

	errorTotal, businessLimited, errorSLA, upstreamExcl429529, upstream429, upstream529, err := collector.queryErrorCounts(context.Background(), start, end)
	require.NoError(t, err)
	require.Equal(t, int64(5), errorTotal)
	require.Equal(t, int64(2), businessLimited)
	require.Equal(t, int64(3), errorSLA)
	require.Equal(t, int64(1), upstreamExcl429529)
	require.Equal(t, int64(1), upstream429)
	require.Equal(t, int64(1), upstream529)
	require.NoError(t, mock.ExpectationsWereMet())
	mock.ExpectClose()
	require.NoError(t, db.Close())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOpsMetricsCollectorQueryAccountSwitchCountSQLite(t *testing.T) {
	db, err := sql.Open("sqlite", "file:ops-metrics-switch-count?mode=memory&cache=shared")
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, func() error {
		_, err := db.Exec(`CREATE TABLE ops_error_logs (
			created_at DATETIME, is_count_tokens BOOLEAN, upstream_errors TEXT
		)`)
		return err
	}())
	require.NoError(t, func() error {
		_, err := db.Exec(`INSERT INTO ops_error_logs VALUES
			('2026-05-26 10:10:00', 0, '[{"kind":"failover:account"},{"kind":"retry"}]'),
			('2026-05-26 10:20:00', 0, '[{"kind":"retry_exhausted_failover:model"}]'),
			('2026-05-26 10:30:00', 1, '[{"kind":"failover:ignored"}]'),
			('2026-05-26 10:40:00', 0, NULL)`)
		return err
	}())

	collector := &OpsMetricsCollector{db: db}
	start := time.Date(2026, 5, 26, 10, 0, 0, 0, time.UTC)
	count, err := collector.queryAccountSwitchCount(context.Background(), start, start.Add(time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(2), count)
}
