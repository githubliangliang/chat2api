package repository

import (
	"context"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	dbaccountgroup "github.com/Wei-Shaw/sub2api/ent/accountgroup"
	dbgroup "github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func installFailingSchedulerOutbox(t *testing.T, ctx context.Context, exec sqlExecutor) {
	t.Helper()
	_, err := exec.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS scheduler_outbox (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL,
		account_id INTEGER,
		group_id INTEGER,
		payload TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		dedup_key TEXT UNIQUE
	)`)
	require.NoError(t, err)
	_, err = exec.ExecContext(ctx, `
		CREATE TRIGGER fail_scheduler_outbox_insert
		BEFORE INSERT ON scheduler_outbox
		BEGIN
			SELECT RAISE(ABORT, 'scheduler outbox unavailable');
		END`)
	require.NoError(t, err)
}

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

	updated, err = updated.Update().SetExtra(map[string]any{
		service.UpstreamBillingProbeEnabledExtraKey:    true,
		service.UpstreamBillingRateSyncEnabledExtraKey: true,
		service.UpstreamBillingProbeExtraKey:           map[string]any{"status": "old"},
	}).Save(ctx)
	require.NoError(t, err)
	loaded, err := repo.GetByID(ctx, updated.ID)
	require.NoError(t, err)
	rate := 1.5
	require.NoError(t, repo.UpdateUpstreamBillingProbeSnapshot(ctx, loaded, &service.UpstreamBillingProbeSnapshot{Status: service.UpstreamBillingProbeStatusOK}, &rate))
	updated, err = client.Account.Get(ctx, updated.ID)
	require.NoError(t, err)
	require.InDelta(t, rate, updated.RateMultiplier, 0.000001)

	require.NoError(t, repo.UpdateExtra(ctx, updated.ID, map[string]any{service.UpstreamBillingProbeEnabledExtraKey: false}))
	updated, err = client.Account.Get(ctx, updated.ID)
	require.NoError(t, err)
	require.NotContains(t, updated.Extra, service.UpstreamBillingProbeExtraKey)
}

func TestAccountRepositoryBindGroupsRollsBackWhenSchedulerOutboxWriteFails(t *testing.T) {
	repo, client := newOllamaCloudUsageSQLiteRepository(t)
	ctx := context.Background()

	account, err := client.Account.Create().
		SetName("bind-groups-atomic").
		SetPlatform(service.PlatformGrok).
		SetType(service.AccountTypeAPIKey).
		SetStatus(service.StatusActive).
		SetSchedulable(true).
		SetCredentials(map[string]any{"api_key": "test"}).
		SetExtra(map[string]any{}).
		Save(ctx)
	require.NoError(t, err)
	group, err := client.Group.Create().
		SetName("bind-groups-atomic-group").
		SetPlatform(service.PlatformGrok).
		SetStatus(service.StatusActive).
		Save(ctx)
	require.NoError(t, err)

	installFailingSchedulerOutbox(t, ctx, repo.sql)

	err = repo.BindGroups(ctx, account.ID, []int64{group.ID})
	require.Error(t, err)

	bindings, err := client.AccountGroup.Query().Where(
		dbaccountgroup.AccountIDEQ(account.ID),
		dbaccountgroup.GroupIDEQ(group.ID),
	).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, bindings)
}

func TestAccountRepositoryCreateRollsBackWhenSchedulerOutboxWriteFails(t *testing.T) {
	repo, client := newOllamaCloudUsageSQLiteRepository(t)
	ctx := context.Background()
	installFailingSchedulerOutbox(t, ctx, repo.sql)

	account := &service.Account{
		Name:        "atomic-account-create",
		Platform:    service.PlatformGrok,
		Type:        service.AccountTypeAPIKey,
		Status:      service.StatusActive,
		Schedulable: true,
		Credentials: map[string]any{"api_key": "test"},
		Extra:       map[string]any{},
	}
	err := repo.Create(ctx, account)
	require.Error(t, err)

	count, err := client.Account.Query().Where(dbaccount.NameEQ("atomic-account-create")).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestGroupRepositoryCreateRollsBackWhenSchedulerOutboxWriteFails(t *testing.T) {
	accountRepo, client := newOllamaCloudUsageSQLiteRepository(t)
	groupRepo := newGroupRepositoryWithSQL(client, accountRepo.sql)
	ctx := context.Background()
	installFailingSchedulerOutbox(t, ctx, accountRepo.sql)

	group := &service.Group{
		Name:             "atomic-group-create",
		Platform:         service.PlatformGrok,
		Status:           service.StatusActive,
		RateMultiplier:   1,
		SubscriptionType: service.SubscriptionTypeStandard,
	}
	err := groupRepo.Create(ctx, group)
	require.Error(t, err)

	count, err := client.Group.Query().Where(dbgroup.NameEQ("atomic-group-create")).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestGroupRepositoryUpdateRollsBackWhenSchedulerOutboxWriteFails(t *testing.T) {
	accountRepo, client := newOllamaCloudUsageSQLiteRepository(t)
	groupRepo := newGroupRepositoryWithSQL(client, accountRepo.sql)
	ctx := context.Background()

	created, err := client.Group.Create().
		SetName("group-before-update").
		SetPlatform(service.PlatformGrok).
		SetStatus(service.StatusActive).
		SetRateMultiplier(1).
		SetSubscriptionType(service.SubscriptionTypeStandard).
		Save(ctx)
	require.NoError(t, err)
	group := groupEntityToService(created)
	group.Name = "group-after-update"
	installFailingSchedulerOutbox(t, ctx, accountRepo.sql)

	err = groupRepo.Update(ctx, group)
	require.Error(t, err)

	got, err := client.Group.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "group-before-update", got.Name)
}

func TestAccountRepositoryRecoveryRollsBackWhenSchedulerOutboxWriteFails(t *testing.T) {
	type recoveryCase struct {
		name    string
		prepare func(*dbent.Account) (*dbent.Account, error)
		invoke  func(*accountRepository, *dbent.Account) error
		assert  func(*testing.T, *dbent.Account)
	}

	limitedAt := time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC)
	resetAt := limitedAt.Add(time.Hour)
	overloadUntil := limitedAt.Add(30 * time.Minute)
	tempUntil := limitedAt.Add(15 * time.Minute)
	cases := []recoveryCase{
		{
			name: "clear error",
			prepare: func(account *dbent.Account) (*dbent.Account, error) {
				return account.Update().SetStatus(service.StatusError).SetErrorMessage("broken").Save(context.Background())
			},
			invoke: func(repo *accountRepository, account *dbent.Account) error {
				return repo.ClearError(context.Background(), account.ID)
			},
			assert: func(t *testing.T, account *dbent.Account) {
				require.Equal(t, service.StatusError, account.Status)
				require.NotNil(t, account.ErrorMessage)
				require.Equal(t, "broken", *account.ErrorMessage)
			},
		},
		{
			name: "enable scheduling",
			prepare: func(account *dbent.Account) (*dbent.Account, error) {
				return account.Update().SetSchedulable(false).Save(context.Background())
			},
			invoke: func(repo *accountRepository, account *dbent.Account) error {
				return repo.SetSchedulable(context.Background(), account.ID, true)
			},
			assert: func(t *testing.T, account *dbent.Account) {
				require.False(t, account.Schedulable)
			},
		},
		{
			name: "clear observed rate limit",
			prepare: func(account *dbent.Account) (*dbent.Account, error) {
				return account.Update().SetRateLimitedAt(limitedAt).SetRateLimitResetAt(resetAt).Save(context.Background())
			},
			invoke: func(repo *accountRepository, account *dbent.Account) error {
				_, err := repo.ClearRateLimitIfObserved(context.Background(), account.ID, limitedAt, resetAt)
				return err
			},
			assert: func(t *testing.T, account *dbent.Account) {
				require.NotNil(t, account.RateLimitedAt)
				require.NotNil(t, account.RateLimitResetAt)
			},
		},
		{
			name: "clear rate limit",
			prepare: func(account *dbent.Account) (*dbent.Account, error) {
				return account.Update().SetRateLimitedAt(limitedAt).SetRateLimitResetAt(resetAt).SetOverloadUntil(overloadUntil).Save(context.Background())
			},
			invoke: func(repo *accountRepository, account *dbent.Account) error {
				return repo.ClearRateLimit(context.Background(), account.ID)
			},
			assert: func(t *testing.T, account *dbent.Account) {
				require.NotNil(t, account.RateLimitedAt)
				require.NotNil(t, account.RateLimitResetAt)
				require.NotNil(t, account.OverloadUntil)
			},
		},
		{
			name: "clear temporary unschedulable state",
			prepare: func(account *dbent.Account) (*dbent.Account, error) {
				return account.Update().SetTempUnschedulableUntil(tempUntil).SetTempUnschedulableReason("temporary").Save(context.Background())
			},
			invoke: func(repo *accountRepository, account *dbent.Account) error {
				return repo.ClearTempUnschedulable(context.Background(), account.ID)
			},
			assert: func(t *testing.T, account *dbent.Account) {
				require.NotNil(t, account.TempUnschedulableUntil)
				require.NotNil(t, account.TempUnschedulableReason)
				require.Equal(t, "temporary", *account.TempUnschedulableReason)
			},
		},
		{
			name: "clear model rate limits",
			prepare: func(account *dbent.Account) (*dbent.Account, error) {
				return account.Update().SetExtra(map[string]any{"model_rate_limits": map[string]any{"grok-imagine-image": map[string]any{"reason": "limited"}}}).Save(context.Background())
			},
			invoke: func(repo *accountRepository, account *dbent.Account) error {
				return repo.ClearModelRateLimits(context.Background(), account.ID)
			},
			assert: func(t *testing.T, account *dbent.Account) {
				require.Contains(t, account.Extra, "model_rate_limits")
			},
		},
		{
			name: "clear antigravity quota scopes",
			prepare: func(account *dbent.Account) (*dbent.Account, error) {
				return account.Update().SetExtra(map[string]any{"antigravity_quota_scopes": map[string]any{"scope": "limited"}}).Save(context.Background())
			},
			invoke: func(repo *accountRepository, account *dbent.Account) error {
				return repo.ClearAntigravityQuotaScopes(context.Background(), account.ID)
			},
			assert: func(t *testing.T, account *dbent.Account) {
				require.Contains(t, account.Extra, "antigravity_quota_scopes")
			},
		},
		{
			name: "reset quota usage",
			prepare: func(account *dbent.Account) (*dbent.Account, error) {
				return account.Update().SetExtra(map[string]any{"quota_used": 7.0, "quota_daily_used": 3.0, "quota_weekly_used": 5.0}).Save(context.Background())
			},
			invoke: func(repo *accountRepository, account *dbent.Account) error {
				return repo.ResetQuotaUsed(context.Background(), account.ID)
			},
			assert: func(t *testing.T, account *dbent.Account) {
				require.InDelta(t, 7, account.Extra["quota_used"], 0.000001)
				require.InDelta(t, 3, account.Extra["quota_daily_used"], 0.000001)
				require.InDelta(t, 5, account.Extra["quota_weekly_used"], 0.000001)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, client := newOllamaCloudUsageSQLiteRepository(t)
			ctx := context.Background()
			account, err := client.Account.Create().
				SetName("recovery-" + tc.name).
				SetPlatform(service.PlatformGrok).
				SetType(service.AccountTypeOAuth).
				SetStatus(service.StatusActive).
				SetSchedulable(true).
				SetCredentials(map[string]any{"access_token": "test"}).
				SetExtra(map[string]any{}).
				Save(ctx)
			require.NoError(t, err)
			account, err = tc.prepare(account)
			require.NoError(t, err)
			installFailingSchedulerOutbox(t, ctx, repo.sql)

			err = tc.invoke(repo, account)
			require.Error(t, err)

			got, err := client.Account.Get(ctx, account.ID)
			require.NoError(t, err)
			tc.assert(t, got)
		})
	}
}
