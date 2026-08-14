package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type quotaSnapshotCacheStub struct {
	setAccounts []*service.Account
}

func (s *quotaSnapshotCacheStub) GetSnapshot(context.Context, service.SchedulerBucket) ([]*service.Account, bool, error) {
	return nil, false, nil
}
func (s *quotaSnapshotCacheStub) CaptureBucketWriteToken(_ context.Context, bucket service.SchedulerBucket) (service.SchedulerBucketWriteToken, error) {
	return service.SchedulerBucketWriteToken{Bucket: bucket, Epoch: 1}, nil
}
func (s *quotaSnapshotCacheStub) SetSnapshot(context.Context, service.SchedulerBucket, service.SchedulerBucketWriteToken, []service.Account) error {
	return nil
}
func (s *quotaSnapshotCacheStub) RetireBucket(context.Context, service.SchedulerBucket) error {
	return nil
}
func (s *quotaSnapshotCacheStub) ReopenBucket(_ context.Context, bucket service.SchedulerBucket) (service.SchedulerBucketWriteToken, error) {
	return service.SchedulerBucketWriteToken{Bucket: bucket, Epoch: 1}, nil
}
func (s *quotaSnapshotCacheStub) TryAcquireGroupLifecycleLease(_ context.Context, groupID int64, _ time.Duration) (service.SchedulerGroupLifecycleLease, bool, error) {
	return service.SchedulerGroupLifecycleLease{GroupID: groupID, OwnerToken: "quota-snapshot"}, true, nil
}
func (s *quotaSnapshotCacheStub) ReleaseGroupLifecycleLease(context.Context, service.SchedulerGroupLifecycleLease) error {
	return nil
}
func (s *quotaSnapshotCacheStub) GetAccount(context.Context, int64) (*service.Account, error) {
	return nil, nil
}
func (s *quotaSnapshotCacheStub) SetAccount(_ context.Context, account *service.Account) error {
	s.setAccounts = append(s.setAccounts, account)
	return nil
}
func (s *quotaSnapshotCacheStub) DeleteAccount(context.Context, int64) error { return nil }
func (s *quotaSnapshotCacheStub) UpdateLastUsed(context.Context, map[int64]time.Time) error {
	return nil
}
func (s *quotaSnapshotCacheStub) TryLockBucket(context.Context, service.SchedulerBucket, time.Duration) (bool, error) {
	return true, nil
}
func (s *quotaSnapshotCacheStub) UnlockBucket(context.Context, service.SchedulerBucket) error {
	return nil
}
func (s *quotaSnapshotCacheStub) ListBuckets(context.Context) ([]service.SchedulerBucket, error) {
	return nil, nil
}
func (s *quotaSnapshotCacheStub) GetOutboxWatermark(context.Context) (int64, error) { return 0, nil }
func (s *quotaSnapshotCacheStub) SetOutboxWatermark(context.Context, int64) error   { return nil }

func TestAccountQuotaJustExceededDetectsTotalAndDaily(t *testing.T) {
	t.Parallel()

	require.False(t, accountQuotaJustExceeded(service.AccountTypeAPIKey, map[string]any{
		"quota_limit": 5.0, "quota_used": 1.0,
	}, map[string]any{
		"quota_limit": 5.0, "quota_used": 4.0,
	}))
	require.True(t, accountQuotaJustExceeded(service.AccountTypeAPIKey, map[string]any{
		"quota_limit": 5.0, "quota_used": 4.0,
	}, map[string]any{
		"quota_limit": 5.0, "quota_used": 5.0,
	}))
	require.True(t, accountQuotaJustExceeded(service.AccountTypeAPIKey, map[string]any{
		"quota_daily_limit": 1.0, "quota_daily_used": 0.4, "quota_daily_start": time.Now().UTC().Format(time.RFC3339Nano),
	}, map[string]any{
		"quota_daily_limit": 1.0, "quota_daily_used": 1.1, "quota_daily_start": time.Now().UTC().Format(time.RFC3339Nano),
	}))
}

func TestIncrementQuotaUsedSyncsSchedulerSnapshotWhenQuotaExceeded(t *testing.T) {
	cache := &quotaSnapshotCacheStub{}
	repo, client := newOllamaCloudUsageSQLiteRepository(t)
	repo.schedulerCache = cache
	ctx := context.Background()
	_, err := repo.sql.ExecContext(ctx, `CREATE TABLE scheduler_outbox (
		id INTEGER PRIMARY KEY AUTOINCREMENT, event_type TEXT NOT NULL, account_id INTEGER,
		group_id INTEGER, payload TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		dedup_key TEXT UNIQUE)`)
	require.NoError(t, err)

	created, err := client.Account.Create().
		SetName("quota").
		SetPlatform(service.PlatformAnthropic).
		SetType(service.AccountTypeAPIKey).
		SetCredentials(map[string]any{"api_key": "k"}).
		SetExtra(map[string]any{"quota_limit": 5.0, "quota_used": 4.0}).
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, repo.IncrementQuotaUsed(ctx, created.ID, 1.5))
	require.NotEmpty(t, cache.setAccounts, "quota crossing must refresh the scheduler account snapshot")
	latest := cache.setAccounts[len(cache.setAccounts)-1]
	require.Equal(t, created.ID, latest.ID)
	require.True(t, latest.IsQuotaExceeded(), "snapshot extra must show the account as over quota so selection can switch away")
}
