package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// 事故回归：scheduler_outbox 历史行（已消费或被开机水位跳过）带着 dedup_key
// 永久滞留时，旧实现的 ON CONFLICT DO NOTHING 会把后续同 key 事件静默吞掉，
// 调度桶因此永远学不到「账号重新启用」。新实现改为删旧插新。
func newEnqueueOutboxSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared&_time_format=sqlite")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TABLE scheduler_outbox (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL,
		account_id INTEGER,
		group_id INTEGER,
		payload TEXT,
		dedup_key TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	require.NoError(t, err)
	// 与线上一致的部分唯一索引
	_, err = db.Exec(`CREATE UNIQUE INDEX idx_scheduler_outbox_pending_dedup_key
		ON scheduler_outbox (dedup_key) WHERE dedup_key IS NOT NULL`)
	require.NoError(t, err)
	return db
}

func TestEnqueueSchedulerOutboxReplacesStaleDedupRow(t *testing.T) {
	db := newEnqueueOutboxSQLiteDB(t)
	ctx := context.Background()
	accountID := int64(2)
	key := schedulerOutboxDedupKey(service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil)

	// 模拟被开机水位跳过的历史行：dedup_key 仍占位
	stale := time.Now().UTC().Add(-72 * time.Hour)
	_, err := db.ExecContext(ctx, `INSERT INTO scheduler_outbox
		(event_type, account_id, dedup_key, created_at) VALUES (?, ?, ?, ?)`,
		service.SchedulerOutboxEventAccountChanged, accountID, key, stale)
	require.NoError(t, err)
	var staleID int64
	require.NoError(t, db.QueryRowContext(ctx, `SELECT id FROM scheduler_outbox WHERE dedup_key = ?`, key).Scan(&staleID))

	// 新事件必须落库（旧实现在此被 DO NOTHING 吞掉）
	require.NoError(t, enqueueSchedulerOutbox(ctx, db,
		service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil))

	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM scheduler_outbox`).Scan(&count))
	require.Equal(t, 1, count, "stale row must be replaced, not accumulated")
	var newID int64
	require.NoError(t, db.QueryRowContext(ctx, `SELECT id FROM scheduler_outbox WHERE dedup_key = ?`, key).Scan(&newID))
	require.Greater(t, newID, staleID, "replacement must land above the consumed watermark")
}

func TestEnqueueSchedulerOutboxCoalescesPendingDuplicates(t *testing.T) {
	db := newEnqueueOutboxSQLiteDB(t)
	ctx := context.Background()
	accountID := int64(18)

	require.NoError(t, enqueueSchedulerOutbox(ctx, db,
		service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil))
	require.NoError(t, enqueueSchedulerOutbox(ctx, db,
		service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil))

	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM scheduler_outbox`).Scan(&count))
	require.Equal(t, 1, count, "pending duplicates must coalesce into one row")

	// 无 dedup 的事件类型保持原语义：逐条累积
	require.NoError(t, enqueueSchedulerOutbox(ctx, db,
		service.SchedulerOutboxEventAccountLastUsed, nil, nil, map[string]any{"last_used": map[string]int64{"18": 1}}))
	require.NoError(t, enqueueSchedulerOutbox(ctx, db,
		service.SchedulerOutboxEventAccountLastUsed, nil, nil, map[string]any{"last_used": map[string]int64{"18": 2}}))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM scheduler_outbox`).Scan(&count))
	require.Equal(t, 3, count)
}

// 事故回归：线上 SQLite 的 scheduler_outbox 由 EnsureSQLiteAuxTables 以
// created_at TEXT 建表（迁移 036 的 DATETIME 被 IF NOT EXISTS 空转），
// modernc 对 TEXT 列返回 string，直接 Scan(*time.Time) 让 poller 每轮报
// "unsupported Scan"、事件永不消费。读取端必须列类型无关。
func TestSchedulerOutboxScansTextCreatedAtColumn(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared&_time_format=sqlite")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	// 与线上一致：TEXT 列 + CURRENT_TIMESTAMP 缺省
	_, err = db.Exec(`CREATE TABLE scheduler_outbox (
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL,
		account_id INTEGER,
		group_id INTEGER,
		payload TEXT,
		dedup_key TEXT,
		created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO scheduler_outbox (event_type, account_id, payload) VALUES
		(?, 2, NULL), (?, 18, '{"group_ids":[3]}')`,
		service.SchedulerOutboxEventAccountChanged, service.SchedulerOutboxEventAccountChanged)
	require.NoError(t, err)
	// 带时区偏移的历史行也必须能读（旧库中存在混合格式）
	_, err = db.Exec(`INSERT INTO scheduler_outbox (event_type, account_id, created_at) VALUES
		(?, 22, '2026-08-16 03:29:45.451687748+08:00')`, service.SchedulerOutboxEventAccountChanged)
	require.NoError(t, err)

	repo := &schedulerOutboxRepository{db: db}
	events, err := repo.ListAfterAndReleaseDedup(context.Background(), 0, 10)
	require.NoError(t, err, "TEXT created_at must not break the poller scan")
	require.Len(t, events, 3)
	for _, ev := range events {
		require.False(t, ev.CreatedAt.IsZero(), "row %d created_at must parse", ev.ID)
	}

	got, ok, err := repo.FirstCreatedAtAfter(context.Background(), 0)
	require.NoError(t, err)
	require.True(t, ok)
	require.False(t, got.IsZero())
}

// created_at 在本仓库有四种来源，写入格式各不相同，解析端必须全部覆盖。
// 尤其 NOW()：sqlite_pg_compat 把它注册成返回 RFC3339Nano UTC 字符串（带 Z），
// 而 55 处仓储 SQL 都在用 NOW() 盖时间戳——一旦有人给 created_at 也用上，
// 解析不了就会复现 poller 每轮报错、事件永不消费的老问题。
func TestParseSchedulerOutboxTimeCoversEveryWriterFormat(t *testing.T) {
	for name, literal := range map[string]string{
		"CURRENT_TIMESTAMP(aux 表默认值)": "2026-08-16 07:30:00",
		"datetime('now')(迁移默认值)":      "2026-08-16 07:30:00.123",
		"NOW()(sqlite_pg_compat)":     time.Now().UTC().Format(time.RFC3339Nano),
		"Go time.Time 带时区偏移":          "2026-08-16 03:29:45.451687748+08:00",
		"RFC3339 带 T 与偏移":             "2026-08-16T03:29:45.451687748+08:00",
	} {
		got, err := parseSchedulerOutboxTimeString(literal)
		require.NoErrorf(t, err, "%s: %q 必须可解析", name, literal)
		require.Falsef(t, got.IsZero(), "%s: 解析结果不能是零值", name)
	}

	_, err := parseSchedulerOutboxTimeString("not-a-time")
	require.Error(t, err, "无法识别的字面量必须报错，不能静默返回零值")
}

func TestSetSchedulableSyncsSnapshotOnEnable(t *testing.T) {
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
		SetName("toggle").
		SetPlatform(service.PlatformOpenAI).
		SetType(service.AccountTypeAPIKey).
		SetCredentials(map[string]any{"api_key": "k"}).
		SetSchedulable(false).
		Save(ctx)
	require.NoError(t, err)

	// 重新启用必须立即回写调度缓存：只靠 outbox 的最终一致，
	// 旧 meta（schedulable=false）会让账号在下一次桶重建前一直被调度层拒绝。
	require.NoError(t, repo.SetSchedulable(ctx, created.ID, true))
	require.NotEmpty(t, cache.setAccounts, "enabling must refresh the scheduler account snapshot")
	latest := cache.setAccounts[len(cache.setAccounts)-1]
	require.Equal(t, created.ID, latest.ID)
	require.True(t, latest.Schedulable, "synced snapshot must carry schedulable=true")

	// 关闭路径原有行为保持
	cache.setAccounts = nil
	require.NoError(t, repo.SetSchedulable(ctx, created.ID, false))
	require.NotEmpty(t, cache.setAccounts)
	require.False(t, cache.setAccounts[len(cache.setAccounts)-1].Schedulable)
}
