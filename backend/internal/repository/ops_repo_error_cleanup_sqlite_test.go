package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestOpsRepositoryDeleteAllErrorLogs(t *testing.T) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE ops_error_logs (id INTEGER PRIMARY KEY, created_at DATETIME)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO ops_error_logs (id) VALUES (1), (2), (3)`)
	require.NoError(t, err)

	repo := &opsRepository{db: db}
	deleted, err := repo.DeleteAllErrorLogs(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, 3, deleted)

	var remaining int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM ops_error_logs`).Scan(&remaining))
	require.Zero(t, remaining)
}
