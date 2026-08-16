package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// simple 模式全量重建按平台整表加载；共享 fake 只桩了 ungrouped 变体。
type initialPurgeAccountRepo struct {
	outboxCleanupAccountRepo
}

func (r *initialPurgeAccountRepo) ListSchedulableByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}

func (r *initialPurgeAccountRepo) ListSchedulableByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}

// 事故回归：开机水位跳过的历史 outbox 行永远不被消费，其 dedup_key 永久占位，
// 后续同 key 事件在入队时被吞。开机全量重建成功后必须清掉开机前的全部历史行。
func TestSchedulerSnapshotServiceInitialRebuildPurgesPreBootOutbox(t *testing.T) {
	cache := &outboxCleanupCache{}
	repo := &outboxCleanupRepo{
		rows:         int64Range(1, 4648),
		lockAcquired: true,
	}
	cfg := &config.Config{RunMode: config.RunModeSimple}
	svc := NewSchedulerSnapshotService(cache, repo, &initialPurgeAccountRepo{}, nil, cfg)

	svc.runInitialRebuild()

	if repo.maxIDCalls == 0 {
		t.Fatal("expected startup to capture the pre-boot outbox max id")
	}
	if len(repo.rows) != 0 {
		t.Fatalf("expected all pre-boot outbox rows purged, %d remain", len(repo.rows))
	}
	for _, call := range repo.deleteCalls {
		if call.watermark != 4648 {
			t.Fatalf("purge must stop at the boot max id, got watermark %d", call.watermark)
		}
	}
}

// 重建失败时不得清理：历史事件仍是尚未被快照覆盖的事实来源。
func TestSchedulerSnapshotServiceInitialRebuildSkipsPurgeWhenRebuildFails(t *testing.T) {
	cache := &outboxCleanupCache{}
	repo := &outboxCleanupRepo{
		rows:         int64Range(1, 10),
		lockAcquired: true,
	}
	// standard 模式 + groupRepo 为 nil → rebuildFullSnapshot 必然失败
	svc := NewSchedulerSnapshotService(cache, repo, &initialPurgeAccountRepo{}, nil, nil)

	svc.runInitialRebuild()

	if len(repo.deleteCalls) != 0 {
		t.Fatalf("purge must not run after a failed rebuild, got %d delete calls", len(repo.deleteCalls))
	}
	if len(repo.rows) != 10 {
		t.Fatalf("rows must remain intact after a failed rebuild, %d remain", len(repo.rows))
	}
}

// 空表时不触发清理调用。
func TestSchedulerSnapshotServiceInitialRebuildSkipsPurgeWhenOutboxEmpty(t *testing.T) {
	cache := &outboxCleanupCache{}
	repo := &outboxCleanupRepo{lockAcquired: true}
	cfg := &config.Config{RunMode: config.RunModeSimple}
	svc := NewSchedulerSnapshotService(cache, repo, &initialPurgeAccountRepo{}, nil, cfg)

	svc.runInitialRebuild()

	if len(repo.deleteCalls) != 0 {
		t.Fatalf("no purge expected for an empty outbox, got %d delete calls", len(repo.deleteCalls))
	}
}
