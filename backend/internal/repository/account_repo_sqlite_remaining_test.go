package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAccountRepositorySQLiteRemainingPaths(t *testing.T) {
	repo, client := newOllamaCloudUsageSQLiteRepository(t)
	ctx := context.Background()
	_, err := repo.sql.ExecContext(ctx, `CREATE TABLE scheduler_outbox (
		id INTEGER PRIMARY KEY AUTOINCREMENT, event_type TEXT NOT NULL, account_id INTEGER,
		group_id INTEGER, payload TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		dedup_key TEXT UNIQUE)`)
	require.NoError(t, err)
	probeExtra := func(rate float64) map[string]any {
		return map[string]any{service.UpstreamBillingProbeExtraKey: map[string]any{
			"status": service.UpstreamBillingProbeStatusOK,
			"data":   map[string]any{"effective_rate_multiplier": rate},
		}}
	}
	low, err := client.Account.Create().SetName("low").SetPlatform(service.PlatformOpenAI).SetType(service.AccountTypeAPIKey).SetCredentials(map[string]any{"api_key": "low"}).SetExtra(probeExtra(0.2)).Save(ctx)
	require.NoError(t, err)
	_, err = client.Account.Create().SetName("high").SetPlatform(service.PlatformOpenAI).SetType(service.AccountTypeAPIKey).SetCredentials(map[string]any{"api_key": "high"}).SetExtra(probeExtra(0.8)).Save(ctx)
	require.NoError(t, err)

	accounts, _, err := repo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 10, SortBy: "upstream_billing_rate", SortOrder: "desc"}, "", "", "", "", 0, "")
	require.NoError(t, err)
	require.Equal(t, []string{"high", "low"}, []string{accounts[0].Name, accounts[1].Name})

	name := "bulk-low"
	rows, err := repo.BulkUpdate(ctx, []int64{low.ID}, service.AccountBulkUpdate{Name: &name, Credentials: map[string]any{"base_url": "https://ollama.com"}, Extra: map[string]any{"custom": true}})
	require.NoError(t, err)
	require.EqualValues(t, 1, rows)
	updated, err := client.Account.Get(ctx, low.ID)
	require.NoError(t, err)
	require.Equal(t, name, updated.Name)
	require.Equal(t, "low", updated.Credentials["api_key"])
	require.Equal(t, true, updated.Extra["custom"])

	now := time.Now().UTC()
	updated, err = updated.Update().SetExtra(map[string]any{
		service.UpstreamBillingProbeEnabledExtraKey: true,
		service.UpstreamBillingProbeExtraKey:        map[string]any{"status": "ok", "next_probe_at": now.Add(-time.Minute).Format(time.RFC3339Nano)},
		"quota_limit":                               5.0, "quota_daily_limit": 5.0, "quota_weekly_limit": 5.0,
	}).Save(ctx)
	require.NoError(t, err)
	due, err := repo.ListDueUpstreamBillingProbeAccounts(ctx, now, 10)
	require.NoError(t, err)
	require.Len(t, due, 1)
	require.Equal(t, updated.ID, due[0].ID)

	require.NoError(t, repo.IncrementQuotaUsed(ctx, updated.ID, 2))
	updated, err = client.Account.Get(ctx, updated.ID)
	require.NoError(t, err)
	require.InDelta(t, 2, updated.Extra["quota_used"], 0.000001)
	require.NoError(t, repo.ResetQuotaUsed(ctx, updated.ID))
	updated, err = client.Account.Get(ctx, updated.ID)
	require.NoError(t, err)
	require.InDelta(t, 0, updated.Extra["quota_used"], 0.000001)
}
