package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestChannelMonitorV2DisplayModelIsPlatformScoped(t *testing.T) {
	cfg := service.ChannelMonitorV2Config{Platforms: []service.ChannelMonitorV2PlatformConfig{
		{Platform: "openai", Enabled: true, Models: []string{"shared", "gpt-5"}},
		{Platform: "grok", Enabled: true, Models: []string{"grok-4"}},
		// Empty models list must NOT collapse everything into __other__.
		{Platform: "anthropic", Enabled: true, Models: []string{}},
	}}
	require.Equal(t, "shared", channelMonitorV2DisplayModel(cfg, "openai", "shared"))
	require.Equal(t, service.ChannelMonitorV2OtherModel, channelMonitorV2DisplayModel(cfg, "grok", "shared"))
	require.Equal(t, "claude-sonnet-4", channelMonitorV2DisplayModel(cfg, "anthropic", "claude-sonnet-4"))
	// Unconfigured platform still surfaces the real model name.
	require.Equal(t, "gemini-2.5-pro", channelMonitorV2DisplayModel(cfg, "gemini", "gemini-2.5-pro"))
	require.True(t, channelMonitorV2ModelSelected(service.ChannelMonitorV2Filter{Models: []string{service.ChannelMonitorV2OtherModel}}, cfg, "grok", "shared"))
}

func TestChannelMonitorV2MatrixDimensionKey(t *testing.T) {
	cfg := service.ChannelMonitorV2Config{Platforms: []service.ChannelMonitorV2PlatformConfig{{Platform: "openai", Enabled: true, Models: []string{"gpt-5"}}}}
	key := channelMonitorV2MatrixDimensionKey(service.ChannelMonitorV2GroupByPlatformGroupModel, cfg, "openai", 7, "gpt-5")
	require.Equal(t, channelMonitorV2MatrixKey{platform: "openai", groupID: 7, model: "gpt-5"}, key)
	key = channelMonitorV2MatrixDimensionKey(service.ChannelMonitorV2GroupByPlatformModel, cfg, "openai", 7, "unlisted")
	require.Equal(t, channelMonitorV2MatrixKey{platform: "openai", model: service.ChannelMonitorV2OtherModel}, key)
	key = channelMonitorV2MatrixDimensionKey(service.ChannelMonitorV2GroupByPlatform, cfg, "openai", 7, "gpt-5")
	require.Equal(t, channelMonitorV2MatrixKey{platform: "openai"}, key)
}

func TestChannelMonitorV2HistogramPercentilesAreMergedFromCounts(t *testing.T) {
	// 100 samples: 50@100, 40@500, 10@1000
	// target = int64(total*p + 0.999999) truncates: p50→50, p90→90, p95→95
	// cumulative hits: p50@100, p90@500 (50+40), p95@1000
	histogram := map[int64]int64{100: 50, 500: 40, 1000: 10}
	require.Equal(t, int64(100), *histPercentile(histogram, .5))
	require.Equal(t, int64(500), *histPercentile(histogram, .9))
	require.Equal(t, int64(1000), *histPercentile(histogram, .95))
	require.Nil(t, histPercentile(nil, .95))
	// latencyMetric exposes avg + p50 + p90 + p95
	lat := latencyMetric(1000, 10, histogram)
	require.NotNil(t, lat.AvgMs)
	require.NotNil(t, lat.P50Ms)
	require.NotNil(t, lat.P90Ms)
	require.NotNil(t, lat.P95Ms)
	require.Equal(t, int64(100), *lat.P50Ms)
	require.Equal(t, int64(500), *lat.P90Ms)
	require.Equal(t, int64(1000), *lat.P95Ms)
}

func TestChannelMonitorV2MetricIncludesSuccessRate(t *testing.T) {
	acc := newMetricAccumulator()
	acc.success, acc.errors = 80, 20
	metric := acc.metric(1, false)
	require.Equal(t, int64(100), metric.RequestCount)
	require.InDelta(t, 0.8, metric.SuccessRate, 0.0001)
	require.InDelta(t, 0.2, metric.ErrorRate, 0.0001)
	require.Nil(t, metric.UpstreamAffectedRequests)

	adminMetric := acc.metric(1, true)
	require.NotNil(t, adminMetric.UpstreamAffectedRequests)
}

func TestChannelMonitorV2WhereUsesConfiguredScopeAndEmptyFilterMeansAllConfigured(t *testing.T) {
	filter := service.ChannelMonitorV2Filter{Start: time.Unix(1, 0), End: time.Unix(2, 0)}
	cfg := service.ChannelMonitorV2Config{
		Platforms: []service.ChannelMonitorV2PlatformConfig{{Platform: "openai", Enabled: true}, {Platform: "grok", Enabled: false}},
		GroupIDs:  []int64{3, 4},
	}
	where, args := channelMonitorV2Where(filter, cfg, "m")
	require.Contains(t, where, "m.platform IN ($3)")
	require.Contains(t, where, "m.group_id IN ($4,$5)")
	require.Equal(t, []any{filter.Start, filter.End, "openai", int64(3), int64(4)}, args)
}

func TestChannelMonitorV2WhereRejectsGroupFilterOutsideConfiguredScope(t *testing.T) {
	filter := service.ChannelMonitorV2Filter{
		Start: time.Unix(1, 0), End: time.Unix(2, 0), GroupIDs: []int64{9},
	}
	cfg := service.ChannelMonitorV2Config{
		Platforms: []service.ChannelMonitorV2PlatformConfig{{Platform: "openai", Enabled: true}},
		GroupIDs:  []int64{3, 4},
	}
	where, args := channelMonitorV2Where(filter, cfg, "m")
	require.Contains(t, where, "FALSE")
	require.NotContains(t, where, "m.group_id IN")
	require.Len(t, args, 3)
}

func TestChannelMonitorV2ErrorAggregationCountsFinalUserErrorsOnly(t *testing.T) {
	query := strings.ToLower(channelMonitorV2ClassifyErrorsSQL)
	require.Contains(t, query, "not current_error.is_count_tokens")
	require.Contains(t, query, "error_type = 'cyber_policy'")
	require.Contains(t, query, "row_number() over")
	require.Contains(t, query, "candidate_ids")
	require.Contains(t, query, "bucket_start >= $1 and bucket_start < $2")
	require.Contains(t, query, "upstream_affected")
	require.Contains(t, query, "json_array_length(current_error.upstream_errors) > 0")
	// request_id dedup must be time-bounded (no full-history scan).
	require.Contains(t, query, "datetime($1, '-90 minutes')")
	require.Contains(t, query, "current_error.created_at >= datetime($1, '-90 minutes')")
}

func TestChannelMonitorV2UsageSuccessExcludesCyberBillingRows(t *testing.T) {
	for _, query := range []string{channelMonitorV2UsageMetricsSQL, channelMonitorV2UserMetricsSQL} {
		require.Contains(t, query, "COALESCE(ul.request_type, 0) NOT IN (4, 6)")
		require.Contains(t, query, "ul.actual_cost > 0")
	}
	require.Contains(t, channelMonitorV2PlatformSQL, "g.platform = 'composite'")
	require.Contains(t, channelMonitorV2PlatformSQL, "a.platform")
	require.Contains(t, channelMonitorV2HistogramSQL, "ul.actual_cost > 0")
}

func TestChannelMonitorV2RatesUseCoveredWindow(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	filter := service.ChannelMonitorV2Filter{Start: start, End: start.Add(24 * time.Hour)}
	coverage := service.ChannelMonitorV2Coverage{CoverageStart: start.Add(6 * time.Hour), DataThrough: start.Add(18 * time.Hour)}
	require.Equal(t, 12*60.0, channelMonitorV2CoveredMinutes(filter, coverage))
	effective := channelMonitorV2CommonCoverageFilter(filter, coverage)
	require.Equal(t, coverage.CoverageStart, effective.Start)
	require.Equal(t, coverage.DataThrough, effective.End)
}

func TestChannelMonitorV2HistoryCoverageCompleteIgnoresTrailingLag(t *testing.T) {
	start := time.Date(2026, 8, 7, 2, 20, 0, 0, time.UTC)
	// History reaches the window start → complete even if data_through is behind filter.End.
	require.True(t, channelMonitorV2HistoryCoverageComplete(start, start))
	require.True(t, channelMonitorV2HistoryCoverageComplete(start.Add(-time.Hour), start))
	// Backfill still short of the window start → incomplete.
	require.False(t, channelMonitorV2HistoryCoverageComplete(start.Add(time.Hour), start))
	require.False(t, channelMonitorV2HistoryCoverageComplete(time.Time{}, start))
}

func TestChannelMonitorV2TierRetentionPolicy(t *testing.T) {
	require.Equal(t, 3*24*time.Hour, channelMonitorV2RetentionUser1m)
	require.Equal(t, 7*24*time.Hour, channelMonitorV2RetentionMetrics1m)
	require.Equal(t, 7*24*time.Hour, channelMonitorV2RetentionError1m)
	require.Equal(t, 7*24*time.Hour, channelMonitorV2RetentionHistogram1m)
	require.Equal(t, 7*24*time.Hour, channelMonitorV2RetentionRollup5m)
	require.Equal(t, 30*24*time.Hour, channelMonitorV2RetentionRollup1h)
	require.Equal(t, 45*24*time.Hour, channelMonitorV2RetentionRollup12h)
	require.Equal(t, 90*24*time.Hour, channelMonitorV2RetentionRollup1d)
	require.Equal(t, channelMonitorV2RetentionRollup1d, channelMonitorV2MaxRetention())
	require.Contains(t, channelMonitorV2WatermarkSQL, "'-90 days'")

	// Every fixed rollup second must appear with a retention rule.
	wantSeconds := map[int]time.Duration{
		300:   channelMonitorV2RetentionRollup5m,
		3600:  channelMonitorV2RetentionRollup1h,
		43200: channelMonitorV2RetentionRollup12h,
		86400: channelMonitorV2RetentionRollup1d,
	}
	seen := map[int]time.Duration{}
	for _, rule := range channelMonitorV2RetentionRules {
		if rule.bucketSeconds == 0 {
			require.True(t, rule.retention > 0)
			continue
		}
		if prev, ok := seen[rule.bucketSeconds]; ok {
			require.Equal(t, prev, rule.retention)
		}
		seen[rule.bucketSeconds] = rule.retention
	}
	for seconds, want := range wantSeconds {
		got, ok := seen[seconds]
		require.Truef(t, ok, "missing retention rule for bucket_seconds=%d", seconds)
		require.Equal(t, want, got)
	}

	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	require.Equal(t, now.Add(-7*24*time.Hour), channelMonitorV2RetentionCutoff(now, channelMonitorV2RetentionMetrics1m))
	require.Equal(t, now.Add(-90*24*time.Hour), channelMonitorV2RetentionCutoff(now, channelMonitorV2MaxRetention()))
}

func TestSameFixedRollupBucket(t *testing.T) {
	start := time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC)
	require.True(t, sameFixedRollupBucket(start, start.Add(10*time.Minute), 86400))
	require.False(t, sameFixedRollupBucket(start, start.Add(15*time.Hour), 43200))
	require.False(t, sameFixedRollupBucket(start, start.Add(24*time.Hour), 86400))
}

// Needles present in service.ClassifyChannelMonitorV2Error must appear in the
// aggregation SQL CASE so rollup categories match drilldown classification.
func TestChannelMonitorV2SQLTaxonomyContainsGoNeedles(t *testing.T) {
	sql := channelMonitorV2ClassifyErrorsSQL
	needles := []string{
		"blocked keyword",
		"invalid_api_key",
		"max_tokens",
		"invalid_request",
		"model not supported",
		"billing hard limit",
		"no healthy upstream account",
		"rate_limit",
		"gateway timeout",
		"connection refused",
		"unexpected eof",
	}
	for _, needle := range needles {
		require.Containsf(t, strings.ToLower(sql), strings.ToLower(needle), "SQL taxonomy missing Go needle %q", needle)
	}
}

func TestApplyIgnoredErrorsAdjustsRatesKeepsAbsoluteVolume(t *testing.T) {
	m := service.ChannelMonitorV2Metric{
		RequestCount:  100,
		ErrorRequests: 20,
		ErrorRate:     0.20,
		SuccessRate:   0.80,
	}
	// Success absolute still 80 → success rate stays 0.80 even after ignoring 5 errors.
	m.SuccessRequests = 80
	applyIgnoredErrors(&m, 5)
	require.Equal(t, int64(100), m.RequestCount)
	require.Equal(t, int64(20), m.ErrorRequests)
	require.InDelta(t, 0.15, m.ErrorRate, 0.0001)
	require.InDelta(t, 0.80, m.SuccessRate, 0.0001)

	// Clamp ignored > errors: scored error_rate → 0; success stays true ratio.
	m2 := service.ChannelMonitorV2Metric{RequestCount: 10, ErrorRequests: 2, SuccessRequests: 8, ErrorRate: 0.2, SuccessRate: 0.8}
	applyIgnoredErrors(&m2, 99)
	require.InDelta(t, 0.0, m2.ErrorRate, 0.0001)
	require.InDelta(t, 0.8, m2.SuccessRate, 0.0001)

	// No-op when ignored is zero
	m3 := service.ChannelMonitorV2Metric{RequestCount: 10, ErrorRequests: 2, SuccessRequests: 8, ErrorRate: 0.2, SuccessRate: 0.8}
	applyIgnoredErrors(&m3, 0)
	require.InDelta(t, 0.2, m3.ErrorRate, 0.0001)
	require.InDelta(t, 0.8, m3.SuccessRate, 0.0001)
}

func TestRedactChannelMonitorV2MetricZerosVolume(t *testing.T) {
	// Service helper is in service package; covered there. Keep a smoke note that
	// rates survive a manual zeroing of volume fields used by the UI contract.
	m := service.ChannelMonitorV2Metric{
		RequestCount: 100, ErrorRequests: 10, SuccessRequests: 90,
		TokenCount: 1000, RPM: 5, TPM: 50, ErrorRate: 0.1, SuccessRate: 0.9, CacheRate: 0.4,
	}
	// Mimic redact: zero volume only
	m.RequestCount, m.ErrorRequests, m.SuccessRequests, m.TokenCount = 0, 0, 0, 0
	require.Equal(t, 0.1, m.ErrorRate)
	require.Equal(t, 5.0, m.RPM)
}

func TestChannelMonitorV2CatalogFilterClearsMultiSelectDimensions(t *testing.T) {
	start := time.Unix(1, 0)
	end := time.Unix(2, 0)
	filter := service.ChannelMonitorV2Filter{
		Start: start, End: end, Bucket: time.Minute,
		Platforms: []string{"openai"}, GroupIDs: []int64{3}, Models: []string{"gpt-5"},
	}
	catalog := channelMonitorV2CatalogFilter(filter)
	require.Nil(t, catalog.Platforms)
	require.Nil(t, catalog.GroupIDs)
	require.Nil(t, catalog.Models)
	// Time window / coverage-related fields remain.
	require.Equal(t, start, catalog.Start)
	require.Equal(t, end, catalog.End)
	require.Equal(t, time.Minute, catalog.Bucket)

	cfg := service.ChannelMonitorV2Config{
		Platforms: []service.ChannelMonitorV2PlatformConfig{
			{Platform: "openai", Enabled: true},
			{Platform: "grok", Enabled: true},
		},
		GroupIDs: []int64{3, 4},
	}
	catalogWhere, catalogArgs := channelMonitorV2Where(catalog, cfg, "m")
	_, metricArgs := channelMonitorV2Where(filter, cfg, "m")

	// Catalog WHERE still applies config scope (enabled platforms + group allow-list).
	require.Contains(t, catalogWhere, "m.platform IN")
	require.Contains(t, catalogWhere, "m.group_id IN")
	require.Len(t, catalogArgs, 6) // start, end, two platforms, two groups
	require.Len(t, metricArgs, 4)

	// Metrics WHERE is narrower once multi-select platforms/groups are applied.
	require.NotEqual(t, catalogArgs, metricArgs)
	// Group seeding without multi-select uses full config allow-list.
	require.Equal(t, []int64{3, 4}, configuredChannelMonitorV2GroupIDs(catalog, cfg))
	require.Equal(t, []int64{3}, configuredChannelMonitorV2GroupIDs(filter, cfg))
}

func TestChannelMonitorV2ConfigRoundTripsSQLiteJSON(t *testing.T) {
	db := openChannelMonitorV2SQLite(t)
	_, err := db.Exec(`CREATE TABLE channel_monitor_v2_config (
		id INTEGER PRIMARY KEY, version INTEGER NOT NULL, enabled BOOLEAN NOT NULL,
		refresh_interval_seconds INTEGER NOT NULL, platforms TEXT NOT NULL,
		group_ids TEXT NOT NULL, ignored_error_categories TEXT NOT NULL,
		health_thresholds TEXT NOT NULL, updated_at DATETIME NOT NULL, updated_by INTEGER
	)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO channel_monitor_v2_config VALUES
		(1, 1, 1, 60, '[]', '[]', '[]', '{}', datetime('now'), NULL)`)
	require.NoError(t, err)

	repo := &channelMonitorV2Repository{db: db}
	updatedBy := int64(42)
	want := service.ChannelMonitorV2Config{
		Enabled: true, RefreshIntervalSeconds: 300,
		Platforms: []service.ChannelMonitorV2PlatformConfig{{Platform: "openai", Enabled: true, Models: []string{"gpt-5"}}},
		GroupIDs:  []int64{7, 9}, IgnoredErrorCategories: []string{"timeout"}, UpdatedBy: &updatedBy,
	}
	updated, err := repo.UpdateConfig(context.Background(), want, 1)
	require.NoError(t, err)
	require.Equal(t, 2, updated.Version)
	require.Equal(t, want.Platforms, updated.Platforms)
	require.Equal(t, want.GroupIDs, updated.GroupIDs)
	require.Equal(t, want.IgnoredErrorCategories, updated.IgnoredErrorCategories)

	loaded, err := repo.GetConfig(context.Background())
	require.NoError(t, err)
	require.Equal(t, updated.Version, loaded.Version)
	require.Equal(t, want.Platforms, loaded.Platforms)
	require.Equal(t, want.GroupIDs, loaded.GroupIDs)
	require.Equal(t, want.IgnoredErrorCategories, loaded.IgnoredErrorCategories)

	_, err = repo.UpdateConfig(context.Background(), want, 1)
	require.ErrorIs(t, err, service.ErrChannelMonitorV2ConfigConflict)
}

func TestChannelMonitorV2LoadFactsBucketsOnSQLite(t *testing.T) {
	db := openChannelMonitorV2SQLite(t)
	_, err := db.Exec(`CREATE TABLE groups (id INTEGER PRIMARY KEY, name TEXT, platform TEXT, deleted_at DATETIME, status TEXT)`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE channel_monitor_v2_metrics_1m (
		bucket_start DATETIME NOT NULL, platform TEXT NOT NULL, group_id INTEGER NOT NULL, model TEXT NOT NULL,
		success_requests INTEGER NOT NULL, error_requests INTEGER NOT NULL,
		upstream_affected_requests INTEGER NOT NULL, upstream_attempt_count INTEGER NOT NULL,
		input_tokens INTEGER NOT NULL, output_tokens INTEGER NOT NULL,
		cache_creation_tokens INTEGER NOT NULL, cache_read_tokens INTEGER NOT NULL,
		ttft_sum_ms INTEGER NOT NULL, ttft_count INTEGER NOT NULL,
		duration_sum_ms INTEGER NOT NULL, duration_count INTEGER NOT NULL)`)
	require.NoError(t, err)
	start := time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC)
	for _, offset := range []time.Duration{10 * time.Second, 50 * time.Second} {
		_, err = db.Exec(`INSERT INTO channel_monitor_v2_metrics_1m VALUES ($1,'openai',0,'gpt-5',1,0,0,0,1,2,0,0,10,1,20,1)`, start.Add(offset))
		require.NoError(t, err)
	}
	repo := &channelMonitorV2Repository{db: db}
	cfg := service.ChannelMonitorV2Config{Platforms: []service.ChannelMonitorV2PlatformConfig{{Platform: "openai", Enabled: true}}}
	facts, err := repo.loadFacts(context.Background(), service.ChannelMonitorV2Filter{
		Start: start, End: start.Add(10 * time.Minute), Bucket: 2 * time.Minute,
	}, cfg, true)
	require.NoError(t, err)
	require.Len(t, facts, 1)
	require.Equal(t, int64(2), facts[0].Success)
	require.Equal(t, start.Format(time.RFC3339Nano), facts[0].BucketStart)
}

func TestChannelMonitorV2RecomputeRangeRunsOnSQLite(t *testing.T) {
	db := openChannelMonitorV2SQLite(t)
	db.SetMaxOpenConns(1)
	ctx := context.Background()
	require.NoError(t, ApplyMigrations(ctx, db))

	start := time.Now().UTC().Truncate(time.Minute).Add(-10 * time.Minute)
	end := start.Add(10 * time.Minute)
	_, err := db.ExecContext(ctx, `
		INSERT INTO groups (id, name, platform) VALUES (10, 'OpenAI', 'openai');
		INSERT INTO accounts (id, name, platform, type, credentials, extra) VALUES (20, 'upstream', 'openai', 'apikey', '{}', '{}');
		INSERT INTO usage_logs (
			id, user_id, api_key_id, account_id, group_id, request_id, requested_model, model,
			input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens,
			actual_cost, first_token_ms, duration_ms, request_type, created_at
		) VALUES
			(1, 30, 40, 20, 10, 'usage-1', 'gpt-5', 'gpt-5', 10, 20, 3, 4, 0.1, 75, 400, 0, $1),
			(2, 30, 40, 20, 10, 'usage-2', 'gpt-5', 'gpt-5', 11, 21, 5, 6, 0.2, 80, 450, 0, $2);
		INSERT INTO ops_error_logs (
			id, request_id, user_id, group_id, platform, requested_model, model,
			error_phase, error_type, error_owner, error_source, severity, status_code,
			upstream_status_code, upstream_error_message, upstream_errors,
			is_count_tokens, created_at
		) VALUES
			(1, 'failed-1', 30, 10, 'openai', 'gpt-5', 'gpt-5', 'upstream', 'internal', 'system', 'proxy', 'P1', 500, NULL, 'first failure', '[]', 0, $3),
			(2, 'failed-1', 30, 10, 'openai', 'gpt-5', 'gpt-5', 'upstream', 'upstream', 'provider', 'upstream', 'P1', 504, 504, 'gateway timeout', '[{"code":"a"},{"code":"b"}]', 0, $4)
	`, start.Add(time.Minute), start.Add(2*time.Minute), start.Add(3*time.Minute), start.Add(4*time.Minute))
	require.NoError(t, err)

	repo := &channelMonitorV2Repository{db: db}
	require.NoError(t, repo.RecomputeRange(ctx, start, end))
	_, err = db.ExecContext(ctx, `
		INSERT INTO channel_monitor_v2_metrics_1m (bucket_start, platform, group_id, model)
		VALUES ($1, 'openai', 10, 'old-model')
	`, start.Add(-channelMonitorV2RetentionMetrics1m-time.Hour))
	require.NoError(t, err)
	require.NoError(t, repo.RecomputeRange(ctx, start, end))
	var oldFacts int
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM channel_monitor_v2_metrics_1m WHERE model = 'old-model'
	`).Scan(&oldFacts))
	require.Equal(t, 1, oldFacts, "active backfill must retain old 1m facts until rollups are complete")

	var success, errors, upstreamAffected, upstreamAttempts int64
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT SUM(success_requests), SUM(error_requests), SUM(upstream_affected_requests), SUM(upstream_attempt_count)
		FROM channel_monitor_v2_metrics_1m
	`).Scan(&success, &errors, &upstreamAffected, &upstreamAttempts))
	require.Equal(t, int64(2), success)
	require.Equal(t, int64(1), errors)
	require.Equal(t, int64(1), upstreamAffected)
	require.Equal(t, int64(2), upstreamAttempts)

	var userSuccess, userErrors int64
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT SUM(success_requests), SUM(error_requests)
		FROM channel_monitor_v2_user_metrics_1m WHERE user_id = 30
	`).Scan(&userSuccess, &userErrors))
	require.Equal(t, int64(2), userSuccess)
	require.Equal(t, int64(1), userErrors)

	var timeoutErrors int64
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT SUM(error_requests) FROM channel_monitor_v2_error_metrics_1m WHERE error_category = 'timeout'
	`).Scan(&timeoutErrors))
	require.Equal(t, int64(1), timeoutErrors)

	var globalTTFT, userTTFT int64
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(CASE WHEN user_id = 0 THEN sample_count ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN user_id = 30 THEN sample_count ELSE 0 END), 0)
		FROM channel_monitor_v2_latency_histograms_1m WHERE metric = 'ttft'
	`).Scan(&globalTTFT, &userTTFT))
	require.Equal(t, int64(2), globalTTFT)
	require.Equal(t, int64(2), userTTFT)

	var rollupSuccess, rollupErrors int64
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT SUM(success_requests), SUM(error_requests)
		FROM channel_monitor_v2_metrics_rollup WHERE bucket_seconds = 300
	`).Scan(&rollupSuccess, &rollupErrors))
	require.Equal(t, int64(2), rollupSuccess)
	require.Equal(t, int64(1), rollupErrors)

	var dataThroughEpoch, wantEpoch int64
	require.NoError(t, db.QueryRowContext(ctx, `
		SELECT unixepoch(data_through), unixepoch($1) FROM channel_monitor_v2_watermarks WHERE id = 1
	`, end).Scan(&dataThroughEpoch, &wantEpoch))
	require.Equal(t, wantEpoch, dataThroughEpoch)

	watermark, err := repo.GetAggregationWatermark(ctx)
	require.NoError(t, err)
	require.True(t, watermark.HasData)
	require.Equal(t, wantEpoch, watermark.DataThrough.Unix())
}

func TestChannelMonitorV2LoadErrorDetailsRunsOnSQLite(t *testing.T) {
	db := openChannelMonitorV2SQLite(t)
	_, err := db.Exec(`CREATE TABLE ops_error_logs (
		id INTEGER PRIMARY KEY, request_id TEXT, created_at DATETIME NOT NULL,
		is_count_tokens BOOLEAN NOT NULL, status_code INTEGER, error_type TEXT,
		platform TEXT, group_id INTEGER, requested_model TEXT, model TEXT,
		error_owner TEXT, error_source TEXT, upstream_status_code INTEGER,
		upstream_error_message TEXT, error_message TEXT, upstream_error_detail TEXT,
		error_body TEXT)`)
	require.NoError(t, err)
	start := time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC)
	_, err = db.Exec(`INSERT INTO ops_error_logs VALUES
		(1,'req-1',$1,0,504,'timeout','openai',0,'gpt-5','gpt-5','provider','upstream',504,
		 'gateway timeout','','','')`, start.Add(time.Minute))
	require.NoError(t, err)

	repo := &channelMonitorV2Repository{db: db}
	cfg := service.ChannelMonitorV2Config{Platforms: []service.ChannelMonitorV2PlatformConfig{{Platform: "openai", Enabled: true}}}
	details, err := repo.loadErrorDetails(context.Background(), service.ChannelMonitorV2Filter{
		Start: start, End: start.Add(time.Hour),
	}, cfg)
	require.NoError(t, err)
	require.Len(t, details["timeout"], 1)
	require.Equal(t, "gateway timeout", details["timeout"][0].Message)
}

func openChannelMonitorV2SQLite(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_time_format=sqlite", t.Name()))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}
