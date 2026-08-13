package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *opsRepository) GetRealtimeTrafficSummary(ctx context.Context, filter *service.OpsDashboardFilter) (*service.OpsRealtimeTrafficSummary, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if filter == nil {
		return nil, fmt.Errorf("nil filter")
	}
	if filter.StartTime.IsZero() || filter.EndTime.IsZero() {
		return nil, fmt.Errorf("start_time/end_time required")
	}

	start := filter.StartTime.UTC()
	end := filter.EndTime.UTC()
	if start.After(end) {
		return nil, fmt.Errorf("start_time must be <= end_time")
	}

	window := end.Sub(start)
	if window <= 0 {
		return nil, fmt.Errorf("invalid time window")
	}
	if window > time.Hour {
		return nil, fmt.Errorf("window too large")
	}

	usageJoin, usageWhere, usageArgs, next := buildUsageWhere(filter, start, end, 1)
	errorWhere, errorArgs, _ := buildErrorWhere(filter, start, end, next)

	q := `
WITH usage_buckets AS (
  SELECT
    datetime(strftime('%Y-%m-%d %H:%M:00', ul.created_at)) AS bucket,
    COALESCE(COUNT(*), 0) AS success_count,
    COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) AS token_sum
  FROM usage_logs ul
  ` + usageJoin + `
  ` + usageWhere + `
  GROUP BY 1
),
error_buckets AS (
  SELECT
    datetime(strftime('%Y-%m-%d %H:%M:00', created_at)) AS bucket,
    COALESCE(COUNT(*), 0) AS error_count
  FROM ops_error_logs
  ` + errorWhere + `
    AND COALESCE(status_code, 0) >= 400
  GROUP BY 1
),
combined AS (
  SELECT
    bucket,
    SUM(success_count) AS success_count,
    SUM(token_sum) AS token_sum,
    SUM(error_count) AS error_count,
    SUM(success_count) + SUM(error_count) AS request_total
  FROM (
    SELECT bucket, success_count, token_sum, 0 AS error_count FROM usage_buckets
    UNION ALL
    SELECT bucket, 0, 0, error_count FROM error_buckets
  ) buckets
  GROUP BY bucket
)
SELECT
  COALESCE(SUM(success_count), 0) AS success_total,
  COALESCE(SUM(error_count), 0) AS error_total,
  COALESCE(SUM(token_sum), 0) AS token_total,
  COALESCE(MAX(request_total), 0) AS peak_requests_per_min,
  COALESCE(MAX(token_sum), 0) AS peak_tokens_per_min
FROM combined`

	args := append(usageArgs, errorArgs...)
	var successCount int64
	var errorTotal int64
	var tokenConsumed int64
	var peakRequestsPerMin int64
	var peakTokensPerMin int64
	if err := r.db.QueryRowContext(ctx, q, args...).Scan(
		&successCount,
		&errorTotal,
		&tokenConsumed,
		&peakRequestsPerMin,
		&peakTokensPerMin,
	); err != nil {
		return nil, err
	}

	windowSeconds := window.Seconds()
	if windowSeconds <= 0 {
		windowSeconds = 1
	}

	requestCountTotal := successCount + errorTotal
	qpsAvg := roundTo1DP(float64(requestCountTotal) / windowSeconds)
	tpsAvg := roundTo1DP(float64(tokenConsumed) / windowSeconds)

	// Keep "current" consistent with the dashboard overview semantics: last 1 minute.
	// This remains "within the selected window" since end=start+window.
	qpsCurrent, tpsCurrent, err := r.queryCurrentRates(ctx, filter, end)
	if err != nil {
		return nil, err
	}

	qpsPeak := roundTo1DP(float64(peakRequestsPerMin) / 60.0)
	tpsPeak := roundTo1DP(float64(peakTokensPerMin) / 60.0)

	return &service.OpsRealtimeTrafficSummary{
		StartTime: start,
		EndTime:   end,
		Platform:  strings.TrimSpace(filter.Platform),
		GroupID:   filter.GroupID,
		QPS: service.OpsRateSummary{
			Current: qpsCurrent,
			Peak:    qpsPeak,
			Avg:     qpsAvg,
		},
		TPS: service.OpsRateSummary{
			Current: tpsCurrent,
			Peak:    tpsPeak,
			Avg:     tpsAvg,
		},
	}, nil
}
