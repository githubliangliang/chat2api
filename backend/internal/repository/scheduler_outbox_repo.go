package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type schedulerOutboxRepository struct {
	db *sql.DB
}

type schedulerOutboxCleanupLease struct {
	once    sync.Once
	release func()
}

var schedulerOutboxCleanupLock sync.Mutex

const schedulerOutboxDefaultCleanSize = 5000

func NewSchedulerOutboxRepository(db *sql.DB) service.SchedulerOutboxRepository {
	return &schedulerOutboxRepository{db: db}
}

func (r *schedulerOutboxRepository) ListAfterAndReleaseDedup(ctx context.Context, afterID int64, limit int) ([]service.SchedulerOutboxEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.QueryContext(ctx, `
		SELECT id, event_type, account_id, group_id, payload, created_at
		FROM scheduler_outbox
		WHERE id > $1
		ORDER BY id ASC
		LIMIT $2
	`, afterID, limit)
	if err != nil {
		return nil, err
	}
	events := make([]service.SchedulerOutboxEvent, 0, limit)
	selectedIDs := make([]int64, 0, limit)
	for rows.Next() {
		var (
			payloadRaw []byte
			accountID  sql.NullInt64
			groupID    sql.NullInt64
			createdRaw any
			event      service.SchedulerOutboxEvent
		)
		if err := rows.Scan(&event.ID, &event.EventType, &accountID, &groupID, &payloadRaw, &createdRaw); err != nil {
			return nil, err
		}
		createdAt, err := scanSchedulerOutboxTime(createdRaw)
		if err != nil {
			return nil, fmt.Errorf("scheduler outbox row %d: %w", event.ID, err)
		}
		event.CreatedAt = createdAt
		if accountID.Valid {
			v := accountID.Int64
			event.AccountID = &v
		}
		if groupID.Valid {
			v := groupID.Int64
			event.GroupID = &v
		}
		if len(payloadRaw) > 0 {
			var payload map[string]any
			if err := json.Unmarshal(payloadRaw, &payload); err != nil {
				return nil, err
			}
			event.Payload = payload
		}
		events = append(events, event)
		selectedIDs = append(selectedIDs, event.ID)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if len(selectedIDs) > 0 {
		where, args := sqlInt64In("id", selectedIDs, 1)
		if _, err := tx.ExecContext(ctx, "UPDATE scheduler_outbox SET dedup_key = NULL WHERE "+where+" AND dedup_key IS NOT NULL", args...); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return events, nil
}

func (r *schedulerOutboxRepository) FirstCreatedAtAfter(ctx context.Context, afterID int64) (time.Time, bool, error) {
	var createdRaw any
	err := r.db.QueryRowContext(ctx, `
		SELECT created_at
		FROM scheduler_outbox
		WHERE id > $1
		ORDER BY id ASC
		LIMIT 1
	`, afterID).Scan(&createdRaw)
	if err == sql.ErrNoRows {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	createdAt, err := scanSchedulerOutboxTime(createdRaw)
	if err != nil {
		return time.Time{}, false, err
	}
	return createdAt, true, nil
}

// scanSchedulerOutboxTime 以列类型无关的方式解析 created_at。
// 线上 SQLite 的 scheduler_outbox 由 EnsureSQLiteAuxTables 以 TEXT 列先建成
// （迁移 036 的 DATETIME 定义被 IF NOT EXISTS 空转），modernc 驱动对 TEXT 列
// 返回 string 而非 time.Time；直接 Scan(*time.Time) 会让 outbox poller 每轮
// 报错、事件永不消费。无时区后缀的字面量按 CURRENT_TIMESTAMP 语义视为 UTC。
func scanSchedulerOutboxTime(value any) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case []byte:
		return parseSchedulerOutboxTimeString(string(v))
	case string:
		return parseSchedulerOutboxTimeString(v)
	default:
		return time.Time{}, fmt.Errorf("unsupported created_at type %T", value)
	}
}

func parseSchedulerOutboxTimeString(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	// 带时区的形态：Z07:00 同时吃 "Z"（sqlite_pg_compat 的 NOW() 返回
	// RFC3339Nano UTC）和 "+08:00"（Go time.Time 经 ent 写入）。
	for _, layout := range []string{
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999Z07:00",
	} {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05.999999999",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized created_at literal %q", value)
}

func (r *schedulerOutboxRepository) MaxID(ctx context.Context) (int64, error) {
	var maxID int64
	if err := r.db.QueryRowContext(ctx, "SELECT COALESCE(MAX(id), 0) FROM scheduler_outbox").Scan(&maxID); err != nil {
		return 0, err
	}
	return maxID, nil
}

func (r *schedulerOutboxRepository) DeleteConsumedUpTo(ctx context.Context, watermark int64, limit int) (int64, error) {
	if watermark <= 0 {
		return 0, nil
	}
	if limit <= 0 {
		limit = schedulerOutboxDefaultCleanSize
	}
	// A short grace period keeps newly committed rows available for a poll cycle
	// before cleanup advances past the watermark.
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM scheduler_outbox
		WHERE id IN (
			SELECT id FROM scheduler_outbox
			WHERE id <= $1 AND created_at < datetime('now', '-10 seconds')
			ORDER BY id ASC
			LIMIT $2
		)
	`, watermark, limit)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *schedulerOutboxRepository) TryAcquireCleanupLock(ctx context.Context) (service.SchedulerOutboxCleanupLease, bool, error) {
	if !schedulerOutboxCleanupLock.TryLock() {
		return nil, false, nil
	}
	return &schedulerOutboxCleanupLease{release: schedulerOutboxCleanupLock.Unlock}, true, nil
}

func (l *schedulerOutboxCleanupLease) Release() {
	if l == nil || l.release == nil {
		return
	}
	l.once.Do(l.release)
}

func enqueueSchedulerOutbox(ctx context.Context, exec sqlExecutor, eventType string, accountID *int64, groupID *int64, payload any) error {
	if exec == nil {
		return nil
	}
	var payloadArg any
	var payloadJSON []byte
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		payloadArg = encoded
		payloadJSON = encoded
	}
	query := `
		INSERT INTO scheduler_outbox (event_type, account_id, group_id, payload)
		VALUES ($1, $2, $3, $4)
	`
	args := []any{eventType, accountID, groupID, payloadArg}
	if schedulerOutboxEventSupportsDedup(eventType) {
		dedupKey := schedulerOutboxDedupKey(eventType, accountID, groupID, payloadJSON)
		// 同 key 旧行可能早已被消费、或被开机水位跳过而永久滞留。此前用
		// ON CONFLICT DO NOTHING 会把新事件静默吞掉：调度桶永远学不到这次
		// 变更（典型事故：重新启用账号后调度器仍视其为不存在）。改为先删
		// 后插：未消费的旧行由新行取代（等价合并），已消费/滞留的旧行不再
		// 挡路，且不依赖水位状态。单机 SQLite 单写者下两步无并发冲突。
		if _, err := exec.ExecContext(ctx,
			`DELETE FROM scheduler_outbox WHERE dedup_key = $1`, dedupKey); err != nil {
			return err
		}
		_, err := exec.ExecContext(ctx,
			`INSERT INTO scheduler_outbox (event_type, account_id, group_id, payload, dedup_key) VALUES ($1, $2, $3, $4, $5)`,
			eventType, accountID, groupID, payloadArg, dedupKey)
		return err
	}
	_, err := exec.ExecContext(ctx, query, args...)
	return err
}

func schedulerOutboxDedupKey(eventType string, accountID *int64, groupID *int64, payloadJSON []byte) string {
	h := sha256.New()
	_, _ = h.Write([]byte(eventType))
	_, _ = h.Write([]byte{0})
	if accountID != nil {
		_, _ = h.Write([]byte(strconv.FormatInt(*accountID, 10)))
	}
	_, _ = h.Write([]byte{0})
	if groupID != nil {
		_, _ = h.Write([]byte(strconv.FormatInt(*groupID, 10)))
	}
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(payloadJSON)
	return fmt.Sprintf("scheduler_outbox:%s", hex.EncodeToString(h.Sum(nil)))
}

func schedulerOutboxEventSupportsDedup(eventType string) bool {
	switch eventType {
	case service.SchedulerOutboxEventAccountChanged,
		service.SchedulerOutboxEventGroupChanged,
		service.SchedulerOutboxEventFullRebuild:
		return true
	default:
		return false
	}
}
