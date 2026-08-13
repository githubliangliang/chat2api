package securityaudit

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestCreateStagingWithCapacityRunsOnSQLite(t *testing.T) {
	db, err := sql.Open("sqlite", "file:prompt-audit-admission?mode=memory&cache=shared&_time_format=sqlite")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	_, err = db.Exec(`CREATE TABLE prompt_audit_jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		request_id TEXT NOT NULL DEFAULT '', user_id INTEGER,
		username_snapshot TEXT NOT NULL DEFAULT '', user_email_snapshot TEXT NOT NULL DEFAULT '',
		api_key_id INTEGER, api_key_name_snapshot TEXT NOT NULL DEFAULT '', group_id INTEGER,
		group_name TEXT NOT NULL DEFAULT '', provider TEXT NOT NULL DEFAULT '', endpoint TEXT NOT NULL DEFAULT '',
		protocol TEXT NOT NULL DEFAULT '', model TEXT NOT NULL DEFAULT '', prompt_hash TEXT NOT NULL DEFAULT '',
		redacted_preview TEXT NOT NULL DEFAULT '', prompt_length INTEGER NOT NULL DEFAULT 0,
		message_count INTEGER NOT NULL DEFAULT 0, stage TEXT NOT NULL DEFAULT 'http',
		execution_mode TEXT NOT NULL DEFAULT 'async_audit', config_version INTEGER NOT NULL DEFAULT 1,
		status TEXT NOT NULL DEFAULT 'staging', attempts INTEGER NOT NULL DEFAULT 0,
		max_attempts INTEGER NOT NULL DEFAULT 3, claim_version INTEGER NOT NULL DEFAULT 0,
		next_attempt_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, processing_started_at DATETIME,
		processed_at DATETIME, last_error_code TEXT NOT NULL DEFAULT '', last_error_message TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	require.NoError(t, err)

	repo := NewPostgreSQLRepository(db)
	job, err := repo.CreateStagingWithCapacity(context.Background(), PromptSnapshot{RequestID: "req-1"}, 1, 3, 1)
	require.NoError(t, err)
	require.Equal(t, "staging", job.Status)

	_, err = repo.CreateStagingWithCapacity(context.Background(), PromptSnapshot{RequestID: "req-2"}, 1, 3, 1)
	require.ErrorIs(t, err, ErrQueueFull)
}
