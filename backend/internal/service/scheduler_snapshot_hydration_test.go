//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type snapshotHydrationCache struct {
	snapshot []*Account
	accounts map[int64]*Account
}

func (c *snapshotHydrationCache) GetSnapshot(ctx context.Context, bucket SchedulerBucket) ([]*Account, bool, error) {
	return c.snapshot, true, nil
}

func (c *snapshotHydrationCache) CaptureBucketWriteToken(ctx context.Context, bucket SchedulerBucket) (SchedulerBucketWriteToken, error) {
	return SchedulerBucketWriteToken{Bucket: bucket, Epoch: 1}, nil
}

func (c *snapshotHydrationCache) SetSnapshot(ctx context.Context, bucket SchedulerBucket, token SchedulerBucketWriteToken, accounts []Account) error {
	return nil
}

func (c *snapshotHydrationCache) RetireBucket(ctx context.Context, bucket SchedulerBucket) error {
	return nil
}

func (c *snapshotHydrationCache) ReopenBucket(ctx context.Context, bucket SchedulerBucket) (SchedulerBucketWriteToken, error) {
	return SchedulerBucketWriteToken{Bucket: bucket, Epoch: 1}, nil
}

func (c *snapshotHydrationCache) TryAcquireGroupLifecycleLease(context.Context, int64, time.Duration) (SchedulerGroupLifecycleLease, bool, error) {
	return SchedulerGroupLifecycleLease{}, false, nil
}

func (c *snapshotHydrationCache) ReleaseGroupLifecycleLease(context.Context, SchedulerGroupLifecycleLease) error {
	return nil
}

func (c *snapshotHydrationCache) GetAccount(ctx context.Context, accountID int64) (*Account, error) {
	if c.accounts == nil {
		return nil, nil
	}
	return c.accounts[accountID], nil
}

func (c *snapshotHydrationCache) SetAccount(ctx context.Context, account *Account) error {
	return nil
}

func (c *snapshotHydrationCache) DeleteAccount(ctx context.Context, accountID int64) error {
	return nil
}

func (c *snapshotHydrationCache) UpdateLastUsed(ctx context.Context, updates map[int64]time.Time) error {
	return nil
}

func (c *snapshotHydrationCache) TryLockBucket(ctx context.Context, bucket SchedulerBucket, ttl time.Duration) (bool, error) {
	return true, nil
}

func (c *snapshotHydrationCache) UnlockBucket(ctx context.Context, bucket SchedulerBucket) error {
	return nil
}

func (c *snapshotHydrationCache) ListBuckets(ctx context.Context) ([]SchedulerBucket, error) {
	return nil, nil
}

func (c *snapshotHydrationCache) GetOutboxWatermark(ctx context.Context) (int64, error) {
	return 0, nil
}

func (c *snapshotHydrationCache) SetOutboxWatermark(ctx context.Context, id int64) error {
	return nil
}

func TestOpenAISelectAccountWithLoadAwareness_HydratesSelectedAccountFromSchedulerSnapshot(t *testing.T) {
	cache := &snapshotHydrationCache{
		snapshot: []*Account{
			{
				ID:          1,
				Platform:    PlatformOpenAI,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    1,
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"gpt-4": "gpt-4",
					},
				},
			},
		},
		accounts: map[int64]*Account{
			1: {
				ID:          1,
				Platform:    PlatformOpenAI,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    1,
				Credentials: map[string]any{
					"api_key":       "sk-live",
					"model_mapping": map[string]any{"gpt-4": "gpt-4"},
				},
			},
		},
	}

	schedulerSnapshot := NewSchedulerSnapshotService(cache, nil, nil, nil, nil)
	groupID := int64(2)
	svc := &OpenAIGatewayService{
		schedulerSnapshot: schedulerSnapshot,
		cache:             &stubGatewayCache{},
	}

	selection, err := svc.SelectAccountWithLoadAwareness(context.Background(), &groupID, "", "gpt-4", nil)
	if err != nil {
		t.Fatalf("SelectAccountWithLoadAwareness error: %v", err)
	}
	if selection == nil || selection.Account == nil {
		t.Fatalf("expected selected account")
	}
	if got := selection.Account.GetOpenAIApiKey(); got != "sk-live" {
		t.Fatalf("expected hydrated api key, got %q", got)
	}
}

func TestOpenAINewAcquiredSelectionResult_ReleasesSlotWhenHydrationFails(t *testing.T) {
	cache := &snapshotHydrationCache{
		accounts: map[int64]*Account{},
	}
	schedulerSnapshot := NewSchedulerSnapshotService(cache, nil, stubOpenAIAccountRepo{}, nil, nil)
	svc := &OpenAIGatewayService{
		schedulerSnapshot: schedulerSnapshot,
	}
	releaseCalls := 0

	selection, err := svc.newAcquiredSelectionResult(context.Background(), &Account{ID: 1001}, func() {
		releaseCalls++
	})

	if err == nil {
		t.Fatalf("expected hydration error")
	}
	if selection != nil {
		t.Fatalf("expected nil selection on hydration error")
	}
	if releaseCalls != 1 {
		t.Fatalf("expected release to be called once, got %d", releaseCalls)
	}
}

func TestGatewaySelectAccountWithLoadAwareness_HydratesSelectedAccountFromSchedulerSnapshot(t *testing.T) {
	cache := &snapshotHydrationCache{
		snapshot: []*Account{
			{
				ID:          9,
				Platform:    PlatformAnthropic,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    1,
			},
		},
		accounts: map[int64]*Account{
			9: {
				ID:          9,
				Platform:    PlatformAnthropic,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    1,
				Credentials: map[string]any{
					"api_key": "anthropic-live-key",
				},
			},
		},
	}

	schedulerSnapshot := NewSchedulerSnapshotService(cache, nil, nil, nil, nil)
	svc := &GatewayService{
		schedulerSnapshot: schedulerSnapshot,
		cache:             &mockGatewayCacheForPlatform{},
		cfg:               testConfig(),
	}

	result, err := svc.SelectAccountWithLoadAwareness(context.Background(), nil, "", "claude-3-5-sonnet-20241022", nil, "", 0)
	if err != nil {
		t.Fatalf("SelectAccountWithLoadAwareness error: %v", err)
	}
	if result == nil || result.Account == nil {
		t.Fatalf("expected selected account")
	}
	if got := result.Account.GetCredential("api_key"); got != "anthropic-live-key" {
		t.Fatalf("expected hydrated api key, got %q", got)
	}
}

func TestGatewaySelectAccountWithLoadAwareness_SkipsFreshlyDisabledSnapshotAccount(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Account)
	}{
		{
			name: "manually unschedulable",
			mutate: func(account *Account) {
				account.Schedulable = false
			},
		},
		{
			name: "manually disabled",
			mutate: func(account *Account) {
				account.Status = StatusDisabled
			},
		},
		{
			name: "local quota exhausted",
			mutate: func(account *Account) {
				account.Extra = map[string]any{"quota_limit": 10.0, "quota_used": 10.0}
			},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stalePrimary := &Account{ID: int64(19 + i*10), Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0}
			staleBackup := &Account{ID: stalePrimary.ID + 1, Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 10}
			dbPrimary := *stalePrimary
			tt.mutate(&dbPrimary)
			dbBackup := *staleBackup

			cache := &snapshotHydrationCache{
				snapshot: []*Account{stalePrimary, staleBackup},
				accounts: map[int64]*Account{stalePrimary.ID: stalePrimary, staleBackup.ID: staleBackup},
			}
			cfg := testConfig()
			cfg.Gateway.Scheduling.LoadBatchEnabled = true
			concurrencyCache := &mockConcurrencyCache{}
			svc := &GatewayService{
				accountRepo:        stubOpenAIAccountRepo{accounts: []Account{dbPrimary, dbBackup}},
				schedulerSnapshot:  NewSchedulerSnapshotService(cache, nil, nil, nil, nil),
				cache:              &mockGatewayCacheForPlatform{},
				cfg:                cfg,
				concurrencyService: NewConcurrencyService(concurrencyCache),
			}

			result, err := svc.SelectAccountWithLoadAwareness(context.Background(), nil, "", "claude-3-5-sonnet-20241022", nil, "", 0)
			if err != nil {
				t.Fatalf("SelectAccountWithLoadAwareness error: %v", err)
			}
			if result == nil || result.Account == nil {
				t.Fatalf("expected selected account")
			}
			if result.Account.ID != dbBackup.ID {
				t.Fatalf("expected healthy backup account %d, got %d", dbBackup.ID, result.Account.ID)
			}
			if concurrencyCache.releaseAccountCalls != 1 {
				t.Fatalf("expected stale account slot to be released once, got %d", concurrencyCache.releaseAccountCalls)
			}
		})
	}
}

func TestGatewaySelectAccountWithLoadAwareness_WaitPlanSkipsFreshlyDisabledSnapshotAccount(t *testing.T) {
	stalePrimary := &Account{ID: 59, Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 0}
	staleBackup := &Account{ID: 60, Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, Priority: 10}
	dbPrimary := *stalePrimary
	dbPrimary.Status = StatusDisabled
	dbBackup := *staleBackup
	cache := &snapshotHydrationCache{
		snapshot: []*Account{stalePrimary, staleBackup},
		accounts: map[int64]*Account{stalePrimary.ID: stalePrimary, staleBackup.ID: staleBackup},
	}
	cfg := testConfig()
	cfg.Gateway.Scheduling.LoadBatchEnabled = true
	svc := &GatewayService{
		accountRepo:       stubOpenAIAccountRepo{accounts: []Account{dbPrimary, dbBackup}},
		schedulerSnapshot: NewSchedulerSnapshotService(cache, nil, nil, nil, nil),
		cache:             &mockGatewayCacheForPlatform{},
		cfg:               cfg,
		concurrencyService: NewConcurrencyService(&mockConcurrencyCache{acquireResults: map[int64]bool{
			stalePrimary.ID: false,
			staleBackup.ID:  false,
		}}),
	}

	result, err := svc.SelectAccountWithLoadAwareness(context.Background(), nil, "", "claude-3-5-sonnet-20241022", nil, "", 0)
	if err != nil {
		t.Fatalf("SelectAccountWithLoadAwareness error: %v", err)
	}
	if result == nil || result.Account == nil || result.WaitPlan == nil {
		t.Fatalf("expected wait plan selection")
	}
	if result.Account.ID != dbBackup.ID || result.WaitPlan.AccountID != dbBackup.ID {
		t.Fatalf("expected healthy backup wait plan for account %d, got account=%d wait_plan=%d", dbBackup.ID, result.Account.ID, result.WaitPlan.AccountID)
	}
}

func TestGatewaySelectAccountWithLoadAwareness_SkipsAntigravityGeminiFamilyRateLimitedSnapshot(t *testing.T) {
	resetAt := time.Now().Add(10 * time.Minute).Format(time.RFC3339)
	cache := &snapshotHydrationCache{
		snapshot: []*Account{
			{
				ID:          1,
				Platform:    PlatformAntigravity,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    1,
				AccountGroups: []AccountGroup{
					{AccountID: 1, GroupID: 22},
				},
				GroupIDs: []int64{22},
				Extra: map[string]any{
					"mixed_scheduling": true,
					modelRateLimitsKey: map[string]any{
						antigravityGeminiModelRateLimitKey: map[string]any{
							"rate_limit_reset_at": resetAt,
						},
					},
				},
			},
			{
				ID:          2,
				Platform:    PlatformAntigravity,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    2,
				AccountGroups: []AccountGroup{
					{AccountID: 2, GroupID: 22},
				},
				GroupIDs: []int64{22},
				Extra: map[string]any{
					"mixed_scheduling": true,
				},
			},
		},
		accounts: map[int64]*Account{
			1: {ID: 1, Platform: PlatformAntigravity, Type: AccountTypeOAuth},
			2: {ID: 2, Platform: PlatformAntigravity, Type: AccountTypeOAuth},
		},
	}
	groupID := int64(22)
	svc := &GatewayService{
		schedulerSnapshot: NewSchedulerSnapshotService(cache, nil, nil, nil, nil),
		groupRepo: &mockGroupRepoForGateway{
			groups: map[int64]*Group{
				groupID: {
					ID:       groupID,
					Platform: PlatformGemini,
					Status:   StatusActive,
					Hydrated: true,
				},
			},
		},
		concurrencyService: NewConcurrencyService(&mockConcurrencyCache{}),
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				Scheduling: config.GatewaySchedulingConfig{
					LoadBatchEnabled:         true,
					StickySessionMaxWaiting:  3,
					StickySessionWaitTimeout: time.Second,
					FallbackWaitTimeout:      time.Second,
					FallbackMaxWaiting:       10,
				},
			},
		},
	}

	result, err := svc.SelectAccountWithLoadAwareness(context.Background(), &groupID, "", "gemini-3-flash-preview", nil, "", 0)
	if err != nil {
		t.Fatalf("SelectAccountWithLoadAwareness error: %v", err)
	}
	if result == nil || result.Account == nil {
		t.Fatalf("expected selected account")
	}
	if result.Account.ID != 2 {
		t.Fatalf("expected scheduler to skip Gemini-family limited antigravity account 1, got %d", result.Account.ID)
	}
}

func TestListSchedulableAccountsReloadsWhenSnapshotHasNoSchedulableAccount(t *testing.T) {
	until := time.Now().Add(10 * time.Minute)
	cooled := &Account{
		ID:                     19,
		Platform:               PlatformOpenAI,
		Type:                   AccountTypeAPIKey,
		Status:                 StatusActive,
		Schedulable:            true,
		TempUnschedulableUntil: &until,
	}
	sibling := Account{
		ID:          16,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
	}
	cache := &snapshotHydrationCache{snapshot: []*Account{cooled}}
	repo := stubOpenAIAccountRepo{accounts: []Account{sibling, *cooled}}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.DbFallbackEnabled = true
	svc := NewSchedulerSnapshotService(cache, nil, repo, nil, cfg)

	groupID := int64(3)
	accounts, _, err := svc.ListSchedulableAccounts(context.Background(), &groupID, PlatformOpenAI, false)
	if err != nil {
		t.Fatalf("ListSchedulableAccounts error: %v", err)
	}
	ids := make([]int64, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.ID)
	}
	if len(ids) != 2 {
		t.Fatalf("expected db fallback to return both accounts, got %v", ids)
	}
}

func TestSnapshotHasSchedulableAccountIgnoresCooledDownAccounts(t *testing.T) {
	until := time.Now().Add(time.Minute)
	if snapshotHasSchedulableAccount([]Account{{
		ID: 19, Status: StatusActive, Schedulable: true, TempUnschedulableUntil: &until,
	}}) {
		t.Fatal("cooled-down account must not count as schedulable")
	}
	if !snapshotHasSchedulableAccount([]Account{{
		ID: 16, Status: StatusActive, Schedulable: true,
	}}) {
		t.Fatal("healthy account must count as schedulable")
	}
}
