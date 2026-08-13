package repository

import (
	"context"
	"database/sql"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestListOllamaCloudUsageGroupAccountsSQLiteUsesJSON1AndIn(t *testing.T) {
	db, err := sql.Open("sqlite", "file:ollama-cloud-group-sqlite?mode=memory&cache=shared")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE accounts (
		id INTEGER PRIMARY KEY,
		deleted_at DATETIME,
		platform TEXT,
		type TEXT,
		credentials TEXT
	)`)
	require.NoError(t, err)

	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	defer client.Close()
	repo := newAccountRepositoryWithSQL(client, db, nil)

	accounts, err := repo.ListOllamaCloudUsageGroupAccounts(context.Background(), []*service.Account{{
		Credentials: map[string]any{"api_key": "missing", "base_url": "https://ollama.com"},
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeAPIKey,
	}})
	require.NoError(t, err)
	require.Empty(t, accounts)
}
