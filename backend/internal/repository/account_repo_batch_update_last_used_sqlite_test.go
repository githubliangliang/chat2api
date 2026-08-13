package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// Regression: DeferredService flushes account activity through
// BatchUpdateLastUsed, whose PostgreSQL casts and ANY array fail on SQLite.
func TestAccountRepositoryBatchUpdateLastUsed_SQLite(t *testing.T) {
	registerSQLitePGCompatFunctions()

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	for _, stmt := range []string{
		`CREATE TABLE accounts (
			id INTEGER PRIMARY KEY,
			last_used_at DATETIME,
			updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
			deleted_at DATETIME
		)`,
		`CREATE TABLE scheduler_outbox (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_type TEXT NOT NULL,
			account_id BIGINT,
			group_id BIGINT,
			payload TEXT,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			dedup_key TEXT UNIQUE
		)`,
		`INSERT INTO accounts (id) VALUES (1), (2)`,
		`INSERT INTO accounts (id, deleted_at) VALUES (3, datetime('now'))`,
	} {
		_, err = db.Exec(stmt)
		require.NoError(t, err)
	}

	repo := newAccountRepositoryWithSQL(nil, db, nil)
	first := time.Date(2026, 8, 13, 8, 12, 0, 0, time.UTC)
	second := first.Add(time.Minute)
	deleted := first.Add(2 * time.Minute)

	err = repo.BatchUpdateLastUsed(context.Background(), map[int64]time.Time{
		1: first,
		2: second,
		3: deleted,
	})
	require.NoError(t, err)

	var gotFirst, gotSecond time.Time
	require.NoError(t, db.QueryRow(`SELECT last_used_at FROM accounts WHERE id = 1`).Scan(&gotFirst))
	require.NoError(t, db.QueryRow(`SELECT last_used_at FROM accounts WHERE id = 2`).Scan(&gotSecond))
	require.Equal(t, first, gotFirst)
	require.Equal(t, second, gotSecond)

	var deletedLastUsed sql.NullTime
	require.NoError(t, db.QueryRow(`SELECT last_used_at FROM accounts WHERE id = 3`).Scan(&deletedLastUsed))
	require.False(t, deletedLastUsed.Valid)
}
