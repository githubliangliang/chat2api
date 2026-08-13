package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestOpenEntDriverUsesSQLiteAsOnlyDialect(t *testing.T) {
	cfg := &config.Config{}
	cfg.Database.Driver = "postgres"
	cfg.Database.Path = t.TempDir() + "/sub2api.db"

	drv, err := openEntDriver(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = drv.Close() })
	require.Equal(t, dialect.SQLite, drv.Dialect())
}

func TestSchedulerOutboxRepositoryRunsOnSQLite(t *testing.T) {
	db, err := sql.Open("sqlite", "file:scheduler-outbox-sqlite?mode=memory&cache=shared&_time_format=sqlite")
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
		created_at DATETIME NOT NULL
	)`)
	require.NoError(t, err)
	oldCreatedAt := time.Now().UTC().Add(-time.Minute)
	_, err = db.Exec(`INSERT INTO scheduler_outbox
		(event_type, account_id, payload, dedup_key, created_at)
		VALUES (?, ?, ?, ?, ?), (?, ?, ?, ?, ?)`,
		service.SchedulerOutboxEventAccountChanged, 7, `{"group_ids":[3]}`, "dedup-1", oldCreatedAt,
		service.SchedulerOutboxEventAccountChanged, 8, nil, "dedup-2", oldCreatedAt)
	require.NoError(t, err)

	repo := &schedulerOutboxRepository{db: db}
	events, err := repo.ListAfterAndReleaseDedup(context.Background(), 0, 1)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.EqualValues(t, 7, *events[0].AccountID)
	require.EqualValues(t, []any{float64(3)}, events[0].Payload["group_ids"])

	var firstDedup sql.NullString
	var secondDedup string
	require.NoError(t, db.QueryRow(`SELECT dedup_key FROM scheduler_outbox WHERE id=1`).Scan(&firstDedup))
	require.NoError(t, db.QueryRow(`SELECT dedup_key FROM scheduler_outbox WHERE id=2`).Scan(&secondDedup))
	require.False(t, firstDedup.Valid)
	require.Equal(t, "dedup-2", secondDedup)

	deleted, err := repo.DeleteConsumedUpTo(context.Background(), 2, 1)
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)
	var remaining int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM scheduler_outbox`).Scan(&remaining))
	require.Equal(t, 1, remaining)

	lease, acquired, err := repo.TryAcquireCleanupLock(context.Background())
	require.NoError(t, err)
	require.True(t, acquired)
	require.NotNil(t, lease)
	lease.Release()
}
