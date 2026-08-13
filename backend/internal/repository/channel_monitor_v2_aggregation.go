package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Platform is derived from group/account (usage_logs has no provider column on upstream schema).
const channelMonitorV2PlatformSQL = `lower(` + usageLogEffectivePlatformExpr + `)`
const channelMonitorV2ModelSQL = `COALESCE(NULLIF(TRIM(ul.requested_model), ''), NULLIF(TRIM(ul.model), ''), 'unknown')`

// Tiered retention balances UI windows against storage:
//
//	1m facts  → short (late writes + rebuild rollups)
//	5m/1h/12h/1d rollups → longer, aligned to 90m / 24h / 7d / 30d(+audit)
//
// Backfill may still write short-lived 1m rows for old windows so rollups can be
// built; prune at end of each recompute drops them past their TTL while rollups remain.
const (
	channelMonitorV2RetentionUser1m      = 3 * 24 * time.Hour
	channelMonitorV2RetentionMetrics1m   = 7 * 24 * time.Hour
	channelMonitorV2RetentionError1m     = 7 * 24 * time.Hour
	channelMonitorV2RetentionHistogram1m = 7 * 24 * time.Hour
	channelMonitorV2RetentionRollup5m    = 7 * 24 * time.Hour  // bucket_seconds=300
	channelMonitorV2RetentionRollup1h    = 30 * 24 * time.Hour // 3600
	channelMonitorV2RetentionRollup12h   = 45 * 24 * time.Hour // 43200
	channelMonitorV2RetentionRollup1d    = 90 * 24 * time.Hour // 86400
	channelMonitorV2RetentionMax         = channelMonitorV2RetentionRollup1d
)

// channelMonitorV2MaxRetention is the longest stored window (1d rollup). Used to
// clamp recompute/backfill so we never scan older than product history needs.
func channelMonitorV2MaxRetention() time.Duration {
	return channelMonitorV2RetentionMax
}

func channelMonitorV2RetentionCutoff(now time.Time, retention time.Duration) time.Time {
	return now.UTC().Truncate(time.Minute).Add(-retention)
}

type channelMonitorV2RetentionRule struct {
	table         string
	retention     time.Duration
	bucketSeconds int // 0 = fact table (no bucket_seconds column)
}

// channelMonitorV2RetentionRules is ordered coarse→fine for predictable prune plans.
var channelMonitorV2RetentionRules = []channelMonitorV2RetentionRule{
	{table: "channel_monitor_v2_user_metrics_1m", retention: channelMonitorV2RetentionUser1m},
	{table: "channel_monitor_v2_metrics_1m", retention: channelMonitorV2RetentionMetrics1m},
	{table: "channel_monitor_v2_error_metrics_1m", retention: channelMonitorV2RetentionError1m},
	{table: "channel_monitor_v2_latency_histograms_1m", retention: channelMonitorV2RetentionHistogram1m},
	{table: "channel_monitor_v2_metrics_rollup", retention: channelMonitorV2RetentionRollup5m, bucketSeconds: 300},
	{table: "channel_monitor_v2_user_metrics_rollup", retention: channelMonitorV2RetentionRollup5m, bucketSeconds: 300},
	{table: "channel_monitor_v2_error_metrics_rollup", retention: channelMonitorV2RetentionRollup5m, bucketSeconds: 300},
	{table: "channel_monitor_v2_latency_histograms_rollup", retention: channelMonitorV2RetentionRollup5m, bucketSeconds: 300},
	{table: "channel_monitor_v2_metrics_rollup", retention: channelMonitorV2RetentionRollup1h, bucketSeconds: 3600},
	{table: "channel_monitor_v2_user_metrics_rollup", retention: channelMonitorV2RetentionRollup1h, bucketSeconds: 3600},
	{table: "channel_monitor_v2_error_metrics_rollup", retention: channelMonitorV2RetentionRollup1h, bucketSeconds: 3600},
	{table: "channel_monitor_v2_latency_histograms_rollup", retention: channelMonitorV2RetentionRollup1h, bucketSeconds: 3600},
	{table: "channel_monitor_v2_metrics_rollup", retention: channelMonitorV2RetentionRollup12h, bucketSeconds: 43200},
	{table: "channel_monitor_v2_user_metrics_rollup", retention: channelMonitorV2RetentionRollup12h, bucketSeconds: 43200},
	{table: "channel_monitor_v2_error_metrics_rollup", retention: channelMonitorV2RetentionRollup12h, bucketSeconds: 43200},
	{table: "channel_monitor_v2_latency_histograms_rollup", retention: channelMonitorV2RetentionRollup12h, bucketSeconds: 43200},
	{table: "channel_monitor_v2_metrics_rollup", retention: channelMonitorV2RetentionRollup1d, bucketSeconds: 86400},
	{table: "channel_monitor_v2_user_metrics_rollup", retention: channelMonitorV2RetentionRollup1d, bucketSeconds: 86400},
	{table: "channel_monitor_v2_error_metrics_rollup", retention: channelMonitorV2RetentionRollup1d, bucketSeconds: 86400},
	{table: "channel_monitor_v2_latency_histograms_rollup", retention: channelMonitorV2RetentionRollup1d, bucketSeconds: 86400},
}

func (r *channelMonitorV2Repository) pruneChannelMonitorV2Retention(ctx context.Context, tx *sql.Tx, now time.Time) error {
	// During historical bootstrap, retain all 1m facts until the cursor reaches
	// the oldest rollup boundary. Otherwise adjacent chunks would rebuild the
	// same daily bucket from source rows already pruned by the prior chunk.
	var backfillCursorEpoch sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT unixepoch(backfill_cursor) FROM channel_monitor_v2_watermarks WHERE id = 1`).Scan(&backfillCursorEpoch); err == nil && backfillCursorEpoch.Valid && time.Unix(backfillCursorEpoch.Int64, 0).After(channelMonitorV2RetentionCutoff(now, channelMonitorV2RetentionMax)) {
		return nil
	}
	for _, rule := range channelMonitorV2RetentionRules {
		cutoff := channelMonitorV2RetentionCutoff(now, rule.retention)
		var err error
		if rule.bucketSeconds == 0 {
			_, err = tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE bucket_start < $1`, rule.table), cutoff)
		} else {
			_, err = tx.ExecContext(ctx,
				fmt.Sprintf(`DELETE FROM %s WHERE bucket_seconds = $1 AND bucket_start < $2`, rule.table),
				rule.bucketSeconds, cutoff,
			)
		}
		if err != nil {
			return fmt.Errorf("prune %s (bucket_seconds=%d): %w", rule.table, rule.bucketSeconds, err)
		}
	}
	return nil
}

func (r *channelMonitorV2Repository) RecomputeRange(ctx context.Context, start, end time.Time) (err error) {
	start = start.UTC().Truncate(time.Minute)
	end = end.UTC().Truncate(time.Minute)
	now := time.Now().UTC().Truncate(time.Minute)
	// Clamp to longest rollup TTL so backfill does not scan beyond product history.
	maxCutoff := channelMonitorV2RetentionCutoff(now, channelMonitorV2MaxRetention())
	if start.Before(maxCutoff) {
		start = maxCutoff
	}
	if !start.Before(end) {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Idempotent window rewrite: drop existing facts/rollups in [start,end) then re-insert.
	for _, table := range []string{
		"channel_monitor_v2_latency_histograms_rollup",
		"channel_monitor_v2_error_metrics_rollup",
		"channel_monitor_v2_user_metrics_rollup",
		"channel_monitor_v2_metrics_rollup",
		"channel_monitor_v2_latency_histograms_1m",
		"channel_monitor_v2_error_metrics_1m",
		"channel_monitor_v2_user_metrics_1m",
		"channel_monitor_v2_metrics_1m",
	} {
		if _, err = tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE bucket_start >= $1 AND bucket_start < $2", table), start, end); err != nil {
			return err
		}
	}

	if _, err = tx.ExecContext(ctx, fmt.Sprintf(channelMonitorV2UsageMetricsSQL, channelMonitorV2PlatformSQL, channelMonitorV2ModelSQL), start, end); err != nil {
		return fmt.Errorf("aggregate channel monitor v2 usage: %w", err)
	}
	if _, err = tx.ExecContext(ctx, fmt.Sprintf(channelMonitorV2UserMetricsSQL, channelMonitorV2PlatformSQL, channelMonitorV2ModelSQL), start, end); err != nil {
		return fmt.Errorf("aggregate channel monitor v2 users: %w", err)
	}
	if _, err = tx.ExecContext(ctx, fmt.Sprintf(channelMonitorV2HistogramSQL, channelMonitorV2PlatformSQL, channelMonitorV2ModelSQL, channelMonitorV2HistogramBoundSQL("samples.value_ms")), start, end); err != nil {
		return fmt.Errorf("aggregate channel monitor v2 histograms: %w", err)
	}
	if err = r.aggregateChannelMonitorV2Errors(ctx, tx, start, end); err != nil {
		return fmt.Errorf("aggregate channel monitor v2 errors: %w", err)
	}
	if err = r.recomputeFixedRollups(ctx, tx, start, end); err != nil {
		return err
	}
	// Drop rows past per-tier TTL (1m short, coarse rollups long). Safe after rollup
	// so a backfill chunk can build 1d rollups from temporary 1m rows then discard 1m.
	if err = r.pruneChannelMonitorV2Retention(ctx, tx, now); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, channelMonitorV2WatermarkSQL, start, end); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

const channelMonitorV2UsageMetricsSQL = `
INSERT INTO channel_monitor_v2_metrics_1m (
  bucket_start, platform, group_id, model, success_requests,
  input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens,
  ttft_sum_ms, ttft_count, duration_sum_ms, duration_count, computed_at
)
SELECT strftime('%%Y-%%m-%%d %%H:%%M:00', ul.created_at), %s, COALESCE(ul.group_id, 0), %s,
       COUNT(DISTINCT CASE
         WHEN COALESCE(ul.request_type, 0) NOT IN (4, 6) AND ` + usageLogSuccessFilterUL + `
         THEN COALESCE(NULLIF(ul.request_id, ''), 'usage:' || CAST(ul.id AS TEXT)) END),
       COALESCE(SUM(CASE WHEN ` + usageLogSuccessFilterUL + ` THEN ul.input_tokens ELSE 0 END), 0),
       COALESCE(SUM(CASE WHEN ` + usageLogSuccessFilterUL + ` THEN ul.output_tokens ELSE 0 END), 0),
       COALESCE(SUM(CASE WHEN ` + usageLogSuccessFilterUL + ` THEN ul.cache_creation_tokens ELSE 0 END), 0),
       COALESCE(SUM(CASE WHEN ` + usageLogSuccessFilterUL + ` THEN ul.cache_read_tokens ELSE 0 END), 0),
       COALESCE(SUM(CASE WHEN ul.first_token_ms IS NOT NULL AND ` + usageLogSuccessFilterUL + ` THEN ul.first_token_ms ELSE 0 END), 0),
       COUNT(CASE WHEN ` + usageLogSuccessFilterUL + ` THEN ul.first_token_ms END),
       COALESCE(SUM(CASE WHEN ul.duration_ms IS NOT NULL AND ` + usageLogSuccessFilterUL + ` THEN ul.duration_ms ELSE 0 END), 0),
       COUNT(CASE WHEN ` + usageLogSuccessFilterUL + ` THEN ul.duration_ms END), datetime('now')
FROM usage_logs ul
LEFT JOIN groups g ON g.id = ul.group_id
LEFT JOIN accounts a ON a.id = ul.account_id
WHERE ul.created_at >= $1 AND ul.created_at < $2
GROUP BY 1, 2, 3, 4`

const channelMonitorV2UserMetricsSQL = `
INSERT INTO channel_monitor_v2_user_metrics_1m (
  bucket_start, platform, group_id, model, user_id, success_requests,
  input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens,
  ttft_sum_ms, ttft_count, duration_sum_ms, duration_count, computed_at
)
SELECT strftime('%%Y-%%m-%%d %%H:%%M:00', ul.created_at), %s, COALESCE(ul.group_id, 0), %s, ul.user_id,
       COUNT(DISTINCT CASE
         WHEN COALESCE(ul.request_type, 0) NOT IN (4, 6) AND ` + usageLogSuccessFilterUL + `
         THEN COALESCE(NULLIF(ul.request_id, ''), 'usage:' || CAST(ul.id AS TEXT)) END),
       COALESCE(SUM(CASE WHEN ` + usageLogSuccessFilterUL + ` THEN ul.input_tokens ELSE 0 END), 0),
       COALESCE(SUM(CASE WHEN ` + usageLogSuccessFilterUL + ` THEN ul.output_tokens ELSE 0 END), 0),
       COALESCE(SUM(CASE WHEN ` + usageLogSuccessFilterUL + ` THEN ul.cache_creation_tokens ELSE 0 END), 0),
       COALESCE(SUM(CASE WHEN ` + usageLogSuccessFilterUL + ` THEN ul.cache_read_tokens ELSE 0 END), 0),
       COALESCE(SUM(CASE WHEN ul.first_token_ms IS NOT NULL AND ` + usageLogSuccessFilterUL + ` THEN ul.first_token_ms ELSE 0 END), 0),
       COUNT(CASE WHEN ` + usageLogSuccessFilterUL + ` THEN ul.first_token_ms END),
       COALESCE(SUM(CASE WHEN ul.duration_ms IS NOT NULL AND ` + usageLogSuccessFilterUL + ` THEN ul.duration_ms ELSE 0 END), 0),
       COUNT(CASE WHEN ` + usageLogSuccessFilterUL + ` THEN ul.duration_ms END), datetime('now')
FROM usage_logs ul
LEFT JOIN groups g ON g.id = ul.group_id
LEFT JOIN accounts a ON a.id = ul.account_id
WHERE ul.created_at >= $1 AND ul.created_at < $2 AND ul.user_id IS NOT NULL
GROUP BY 1, 2, 3, 4, 5`

const channelMonitorV2HistogramSQL = `
INSERT INTO channel_monitor_v2_latency_histograms_1m (
  bucket_start, platform, group_id, model, user_id, metric, upper_bound_ms, sample_count
)
WITH base AS (
  SELECT strftime('%%Y-%%m-%%d %%H:%%M:00', ul.created_at) AS bucket_start,
         %s AS platform, COALESCE(ul.group_id, 0) AS group_id, %s AS model,
         ul.user_id, ul.first_token_ms, ul.duration_ms
  FROM usage_logs ul
  LEFT JOIN groups g ON g.id = ul.group_id
  LEFT JOIN accounts a ON a.id = ul.account_id
  WHERE ul.created_at >= $1 AND ul.created_at < $2 AND ` + usageLogSuccessFilterUL + `
), samples AS (
  SELECT bucket_start, platform, group_id, model, 0 AS user_id, 'ttft' AS metric, first_token_ms AS value_ms
  FROM base WHERE first_token_ms IS NOT NULL AND first_token_ms >= 0
  UNION ALL
  SELECT bucket_start, platform, group_id, model, user_id, 'ttft', first_token_ms
  FROM base WHERE user_id IS NOT NULL AND first_token_ms IS NOT NULL AND first_token_ms >= 0
  UNION ALL
  SELECT bucket_start, platform, group_id, model, 0, 'duration', duration_ms
  FROM base WHERE duration_ms IS NOT NULL AND duration_ms >= 0
  UNION ALL
  SELECT bucket_start, platform, group_id, model, user_id, 'duration', duration_ms
  FROM base WHERE user_id IS NOT NULL AND duration_ms IS NOT NULL AND duration_ms >= 0
)
SELECT samples.bucket_start, samples.platform, samples.group_id, samples.model,
       samples.user_id, samples.metric, %s, COUNT(*)
FROM samples
GROUP BY 1, 2, 3, 4, 5, 6, 7`

func channelMonitorV2HistogramBoundSQL(column string) string {
	return `CASE
WHEN ` + column + ` <= 50 THEN 50 WHEN ` + column + ` <= 100 THEN 100
WHEN ` + column + ` <= 250 THEN 250 WHEN ` + column + ` <= 500 THEN 500
WHEN ` + column + ` <= 1000 THEN 1000 WHEN ` + column + ` <= 2000 THEN 2000
WHEN ` + column + ` <= 3000 THEN 3000 WHEN ` + column + ` <= 5000 THEN 5000
WHEN ` + column + ` <= 8000 THEN 8000 WHEN ` + column + ` <= 10000 THEN 10000
WHEN ` + column + ` <= 15000 THEN 15000 WHEN ` + column + ` <= 30000 THEN 30000
WHEN ` + column + ` <= 60000 THEN 60000 WHEN ` + column + ` <= 120000 THEN 120000
WHEN ` + column + ` <= 300000 THEN 300000 WHEN ` + column + ` <= 600000 THEN 600000
ELSE 2147483647 END`
}

// Error dedup uses a per-connection temporary table because SQLite does not
// support data-modifying CTEs. The request-id branch is bounded to 90 minutes
// before the chunk so retries just outside the chunk can still be deduplicated.
const channelMonitorV2ClassifyErrorsSQL = `
CREATE TEMP TABLE channel_monitor_v2_classified_errors AS
WITH candidate_ids AS (
  SELECT DISTINCT request_id
  FROM ops_error_logs
  WHERE created_at >= $1 AND created_at < $2 AND NULLIF(request_id, '') IS NOT NULL
), ranked AS (
  SELECT
    strftime('%Y-%m-%d %H:%M:00', current_error.created_at) AS bucket_start,
    lower(COALESCE(NULLIF(TRIM(current_error.platform), ''), 'unknown')) AS platform,
    COALESCE(current_error.group_id, 0) AS group_id,
    COALESCE(NULLIF(TRIM(current_error.requested_model), ''), NULLIF(TRIM(current_error.model), ''), 'unknown') AS model,
    current_error.user_id, current_error.error_type, current_error.error_owner,
    COALESCE(current_error.status_code, 0) AS status_code,
    COALESCE(current_error.upstream_status_code, 0) AS upstream_status_code,
    lower(
      COALESCE(current_error.error_type, '') || ' ' || COALESCE(current_error.error_source, '') || ' ' ||
      COALESCE(current_error.error_message, '') || ' ' || COALESCE(current_error.upstream_error_message, '') || ' ' ||
      COALESCE(current_error.upstream_error_detail, '') || ' ' || COALESCE(current_error.error_body, '')
    ) AS text,
    CASE WHEN (json_valid(current_error.upstream_errors) AND json_type(current_error.upstream_errors) = 'array'
                    AND json_array_length(current_error.upstream_errors) > 0)
                   OR current_error.error_owner = 'provider' OR current_error.upstream_status_code IS NOT NULL
         THEN 1 ELSE 0 END AS upstream_affected,
    CASE WHEN json_valid(current_error.upstream_errors) AND json_type(current_error.upstream_errors) = 'array'
         THEN json_array_length(current_error.upstream_errors) ELSE 0 END AS upstream_attempts,
    ROW_NUMBER() OVER (
      PARTITION BY COALESCE(NULLIF(current_error.request_id, ''), 'error:' || CAST(current_error.id AS TEXT))
      ORDER BY current_error.created_at DESC, current_error.id DESC
    ) AS row_number
  FROM ops_error_logs current_error
  WHERE (
      (NULLIF(current_error.request_id, '') IS NULL AND current_error.created_at >= $1 AND current_error.created_at < $2)
      OR (
        current_error.request_id IN (SELECT request_id FROM candidate_ids)
        AND current_error.created_at >= datetime($1, '-90 minutes')
        AND current_error.created_at < $2
      )
    )
    AND NOT current_error.is_count_tokens
    AND (COALESCE(current_error.status_code, 0) >= 400 OR current_error.error_type = 'cyber_policy')
), dedup AS (
  SELECT * FROM ranked WHERE row_number = 1 AND bucket_start >= $1 AND bucket_start < $2
)
SELECT *, CASE
  -- Keep in lockstep with service.ClassifyChannelMonitorV2Error needles.
  WHEN error_type = 'cyber_policy' OR text LIKE '%content policy%' OR text LIKE '%content_policy%' OR text LIKE '%safety policy%' OR text LIKE '%moderation%' OR text LIKE '%blocked keyword%' THEN 'content_policy'
  WHEN status_code = 401 OR upstream_status_code = 401 OR text LIKE '%unauthorized%' OR text LIKE '%invalid api key%' OR text LIKE '%invalid_api_key%' OR text LIKE '%authentication%' OR text LIKE '%api_key_disabled%' THEN 'authentication'
  WHEN text LIKE '%context window%' OR text LIKE '%context length%' OR text LIKE '%maximum prompt length%' OR text LIKE '%too many tokens%' OR text LIKE '%max_tokens%' THEN 'context_limit'
  WHEN text LIKE '%failed to deserialize%' OR text LIKE '%missing required parameter%' OR text LIKE '%invalid request%' OR text LIKE '%invalid_request%' OR text LIKE '%tool_choice%' THEN 'invalid_request'
  WHEN text LIKE '%does not support the requested model%' OR text LIKE '%not supported by any configured account%' OR text LIKE '%model not supported%' OR text LIKE '%unsupported model%' THEN 'model_unsupported'
  WHEN text LIKE '%group not allowed%' OR text LIKE '%group_not_allowed%' OR text LIKE '%group access%' THEN 'group_access'
  WHEN text LIKE '%run out of credits%' OR text LIKE '%insufficient balance%' OR text LIKE '%insufficient quota%' OR text LIKE '%subscription%' OR text LIKE '%quota exceeded%' OR text LIKE '%billing hard limit%' THEN 'quota_or_balance'
  WHEN text LIKE '%no available accounts%' OR text LIKE '%no healthy account%' OR text LIKE '%no healthy upstream account%' OR text LIKE '%failover budget exhausted%' OR text LIKE '%account pool%' THEN 'account_pool_unavailable'
  WHEN status_code = 429 OR upstream_status_code = 429 OR text LIKE '%rate limit%' OR text LIKE '%rate_limit%' OR text LIKE '%high demand%' OR text LIKE '%overloaded%' OR text LIKE '%concurrency limit%' OR text LIKE '%capacity%' THEN 'rate_or_capacity'
  WHEN status_code IN (408,504) OR text LIKE '%timeout%' OR text LIKE '%deadline exceeded%' OR text LIKE '%error code: 524%' OR text LIKE '%gateway time-out%' OR text LIKE '%gateway timeout%' THEN 'timeout'
  WHEN text LIKE '%transport%' OR text LIKE '%stream_read_error%' OR text LIKE '%connection reset%' OR text LIKE '%connection refused%' OR text LIKE '%tls%' OR text LIKE '%http2%' OR text LIKE '%missing terminal event%' OR text LIKE '%unexpected eof%' THEN 'transport_or_stream'
  WHEN status_code = 403 OR upstream_status_code = 403 THEN 'upstream_forbidden'
  WHEN status_code = 404 OR upstream_status_code = 404 THEN 'not_found'
  WHEN status_code = 499 OR text LIKE '%client cancelled%' OR text LIKE '%client canceled%' OR text LIKE '%context canceled%' THEN 'client_cancelled'
  WHEN upstream_status_code >= 500 OR (error_owner = 'provider' AND status_code >= 500) THEN 'upstream_5xx'
  WHEN status_code >= 500 OR error_type = 'internal' OR error_owner = 'system' THEN 'internal'
  ELSE 'other' END AS category
FROM dedup`

const channelMonitorV2MetricErrorsSQL = `
INSERT INTO channel_monitor_v2_metrics_1m (
  bucket_start, platform, group_id, model, error_requests,
  upstream_affected_requests, upstream_attempt_count, computed_at
)
SELECT bucket_start, platform, group_id, model, COUNT(*), SUM(upstream_affected),
       SUM(upstream_attempts), datetime('now')
FROM channel_monitor_v2_classified_errors
GROUP BY bucket_start, platform, group_id, model
ON CONFLICT (bucket_start, platform, group_id, model) DO UPDATE SET
  error_requests = excluded.error_requests,
  upstream_affected_requests = excluded.upstream_affected_requests,
  upstream_attempt_count = excluded.upstream_attempt_count,
  computed_at = datetime('now')`

const channelMonitorV2UserErrorsSQL = `
INSERT INTO channel_monitor_v2_user_metrics_1m (
  bucket_start, platform, group_id, model, user_id, error_requests, computed_at
)
SELECT bucket_start, platform, group_id, model, user_id, COUNT(*), datetime('now')
FROM channel_monitor_v2_classified_errors
WHERE user_id IS NOT NULL
GROUP BY bucket_start, platform, group_id, model, user_id
ON CONFLICT (bucket_start, platform, group_id, model, user_id) DO UPDATE SET
  error_requests = excluded.error_requests, computed_at = datetime('now')`

const channelMonitorV2CategoryErrorsSQL = `
INSERT INTO channel_monitor_v2_error_metrics_1m (
  bucket_start, platform, group_id, model, error_category, taxonomy_version, error_requests
)
SELECT bucket_start, platform, group_id, model, category, 1, COUNT(*)
FROM channel_monitor_v2_classified_errors
GROUP BY bucket_start, platform, group_id, model, category
ON CONFLICT (bucket_start, platform, group_id, model, error_category, taxonomy_version)
DO UPDATE SET error_requests = excluded.error_requests`

func (r *channelMonitorV2Repository) aggregateChannelMonitorV2Errors(ctx context.Context, tx *sql.Tx, start, end time.Time) error {
	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS temp.channel_monitor_v2_classified_errors`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, channelMonitorV2ClassifyErrorsSQL, start, end); err != nil {
		return err
	}
	for _, query := range []string{
		channelMonitorV2MetricErrorsSQL,
		channelMonitorV2UserErrorsSQL,
		channelMonitorV2CategoryErrorsSQL,
	} {
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return err
		}
	}
	_, err := tx.ExecContext(ctx, `DROP TABLE temp.channel_monitor_v2_classified_errors`)
	return err
}

// Floor matches channelMonitorV2RetentionMax (90d). Keep the datetime modifier
// in sync when changing channelMonitorV2RetentionRollup1d.
//
// Coverage starts track how far back recompute has walked ($1 = chunk start), not
// "min(source_log.created_at)". Using global min(ops_error_logs) pins
// error_coverage_start to the first real error forever and collapses UI windows
// when errors only exist in a recent slice (common on first upgrade).
const channelMonitorV2WatermarkSQL = `
INSERT INTO channel_monitor_v2_watermarks (id, usage_coverage_start, error_coverage_start, data_through, last_successful_at, backfill_cursor, updated_at)
VALUES (
  1,
  $1,
  $1,
  $2, datetime('now'), $1, datetime('now')
)
ON CONFLICT (id) DO UPDATE SET
  usage_coverage_start = max(
    datetime('now', '-90 days', 'start of minute'),
    min(COALESCE(channel_monitor_v2_watermarks.usage_coverage_start, excluded.usage_coverage_start), excluded.usage_coverage_start)
  ),
  error_coverage_start = max(
    datetime('now', '-90 days', 'start of minute'),
    min(COALESCE(channel_monitor_v2_watermarks.error_coverage_start, excluded.error_coverage_start), excluded.error_coverage_start)
  ),
  data_through = max(COALESCE(channel_monitor_v2_watermarks.data_through, excluded.data_through), excluded.data_through),
  last_successful_at = datetime('now'),
  backfill_cursor = min(COALESCE(channel_monitor_v2_watermarks.backfill_cursor, excluded.backfill_cursor), excluded.backfill_cursor),
  updated_at = datetime('now')`

var channelMonitorV2FixedRollupSeconds = []int{300, 3600, 43200, 86400}

func (r *channelMonitorV2Repository) recomputeFixedRollups(ctx context.Context, tx *sql.Tx, start, end time.Time) error {
	for _, seconds := range channelMonitorV2FixedRollupSeconds {
		// Coarse buckets are immutable between boundaries during the normal
		// trailing refresh. Historical backfills and boundary-crossing windows
		// still rebuild them; this avoids repeatedly regrouping the full current
		// day/user table every few minutes.
		if seconds >= 43200 && sameFixedRollupBucket(start, end, seconds) {
			continue
		}
		interval := time.Duration(seconds) * time.Second
		rollupStart := start.Truncate(interval)
		rollupEnd := end.Add(-time.Nanosecond).Truncate(interval).Add(interval)
		for _, table := range []string{
			"channel_monitor_v2_latency_histograms_rollup",
			"channel_monitor_v2_error_metrics_rollup",
			"channel_monitor_v2_user_metrics_rollup",
			"channel_monitor_v2_metrics_rollup",
		} {
			if _, err := tx.ExecContext(ctx, fmt.Sprintf(channelMonitorV2FixedRollupDeleteSQL, table), seconds, rollupStart, rollupEnd); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, channelMonitorV2MetricsRollupSQL, seconds, rollupStart, rollupEnd); err != nil {
			return fmt.Errorf("roll up channel monitor v2 metrics %ds: %w", seconds, err)
		}
		if _, err := tx.ExecContext(ctx, channelMonitorV2UserMetricsRollupSQL, seconds, rollupStart, rollupEnd); err != nil {
			return fmt.Errorf("roll up channel monitor v2 user metrics %ds: %w", seconds, err)
		}
		if _, err := tx.ExecContext(ctx, channelMonitorV2HistogramRollupSQL, seconds, rollupStart, rollupEnd); err != nil {
			return fmt.Errorf("roll up channel monitor v2 histograms %ds: %w", seconds, err)
		}
		if _, err := tx.ExecContext(ctx, channelMonitorV2ErrorRollupSQL, seconds, rollupStart, rollupEnd); err != nil {
			return fmt.Errorf("roll up channel monitor v2 errors %ds: %w", seconds, err)
		}
	}
	return nil
}

func sameFixedRollupBucket(start, end time.Time, seconds int) bool {
	if !end.After(start) {
		return true
	}
	interval := time.Duration(seconds) * time.Second
	return start.Truncate(interval).Equal(end.Add(-time.Nanosecond).Truncate(interval))
}

const channelMonitorV2FixedRollupDeleteSQL = `
DELETE FROM %s
WHERE bucket_seconds = $1
  AND unixepoch(bucket_start) >= unixepoch($2)
  AND unixepoch(bucket_start) < unixepoch($3)`

const channelMonitorV2MetricsRollupSQL = `
INSERT INTO channel_monitor_v2_metrics_rollup (
  bucket_start, bucket_seconds, platform, group_id, model, success_requests, error_requests,
  upstream_affected_requests, upstream_attempt_count, input_tokens, output_tokens,
  cache_creation_tokens, cache_read_tokens, ttft_sum_ms, ttft_count, duration_sum_ms,
  duration_count, computed_at
)
SELECT datetime((unixepoch(m.bucket_start) / $1) * $1, 'unixepoch'), $1,
       platform, group_id, model, SUM(success_requests), SUM(error_requests),
       SUM(upstream_affected_requests), SUM(upstream_attempt_count), SUM(input_tokens),
       SUM(output_tokens), SUM(cache_creation_tokens), SUM(cache_read_tokens),
       SUM(ttft_sum_ms), SUM(ttft_count), SUM(duration_sum_ms), SUM(duration_count), datetime('now')
FROM channel_monitor_v2_metrics_1m m
WHERE unixepoch(m.bucket_start) >= unixepoch($2) AND unixepoch(m.bucket_start) < unixepoch($3)
GROUP BY 1, 2, 3, 4, 5`

const channelMonitorV2UserMetricsRollupSQL = `
INSERT INTO channel_monitor_v2_user_metrics_rollup (
  bucket_start, bucket_seconds, platform, group_id, model, user_id, success_requests,
  error_requests, input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens,
  ttft_sum_ms, ttft_count, duration_sum_ms, duration_count, computed_at
)
SELECT datetime((unixepoch(m.bucket_start) / $1) * $1, 'unixepoch'), $1,
       platform, group_id, model, user_id, SUM(success_requests), SUM(error_requests),
       SUM(input_tokens), SUM(output_tokens), SUM(cache_creation_tokens), SUM(cache_read_tokens),
       SUM(ttft_sum_ms), SUM(ttft_count), SUM(duration_sum_ms), SUM(duration_count), datetime('now')
FROM channel_monitor_v2_user_metrics_1m m
WHERE unixepoch(m.bucket_start) >= unixepoch($2) AND unixepoch(m.bucket_start) < unixepoch($3)
GROUP BY 1, 2, 3, 4, 5, 6`

const channelMonitorV2HistogramRollupSQL = `
INSERT INTO channel_monitor_v2_latency_histograms_rollup (
  bucket_start, bucket_seconds, platform, group_id, model, user_id, metric, upper_bound_ms, sample_count
)
SELECT datetime((unixepoch(h.bucket_start) / $1) * $1, 'unixepoch'), $1,
       platform, group_id, model, user_id, metric, upper_bound_ms, SUM(sample_count)
FROM channel_monitor_v2_latency_histograms_1m h
WHERE unixepoch(h.bucket_start) >= unixepoch($2) AND unixepoch(h.bucket_start) < unixepoch($3)
GROUP BY 1, 2, 3, 4, 5, 6, 7, 8`

const channelMonitorV2ErrorRollupSQL = `
INSERT INTO channel_monitor_v2_error_metrics_rollup (
  bucket_start, bucket_seconds, platform, group_id, model, error_category, taxonomy_version, error_requests
)
SELECT datetime((unixepoch(e.bucket_start) / $1) * $1, 'unixepoch'), $1,
       platform, group_id, model, error_category, taxonomy_version, SUM(error_requests)
FROM channel_monitor_v2_error_metrics_1m e
WHERE unixepoch(e.bucket_start) >= unixepoch($2) AND unixepoch(e.bucket_start) < unixepoch($3)
GROUP BY 1, 2, 3, 4, 5, 6, 7`
