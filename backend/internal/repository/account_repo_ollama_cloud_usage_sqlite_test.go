package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func newOllamaCloudUsageSQLiteRepository(t *testing.T) (*accountRepository, *dbent.Client) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared&_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	require.NoError(t, client.Schema.Create(context.Background()))
	t.Cleanup(func() { _ = client.Close() })
	return newAccountRepositoryWithSQL(client, db, nil), client
}

func TestListOllamaCloudUsageGroupAccountsSQLiteUsesJSON1AndIn(t *testing.T) {
	db, err := sql.Open("sqlite", "file:ollama-cloud-group-sqlite?mode=memory&cache=shared")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	_, err = db.Exec(`CREATE TABLE accounts (
		id INTEGER PRIMARY KEY,
		deleted_at DATETIME,
		platform TEXT,
		type TEXT,
		credentials TEXT
	)`)
	require.NoError(t, err)

	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	defer func() { _ = client.Close() }()
	repo := newAccountRepositoryWithSQL(client, db, nil)

	accounts, err := repo.ListOllamaCloudUsageGroupAccounts(context.Background(), []*service.Account{{
		Credentials: map[string]any{"api_key": "missing", "base_url": "https://ollama.com"},
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeAPIKey,
	}})
	require.NoError(t, err)
	require.Empty(t, accounts)
}

func TestSaveOllamaCloudUsageSessionSQLiteUpdatesExactAPIKeyGroup(t *testing.T) {
	repo, client := newOllamaCloudUsageSQLiteRepository(t)
	ctx := context.Background()
	credentials := map[string]any{"api_key": "shared", "base_url": "https://ollama.com"}
	first, err := client.Account.Create().SetName("first").SetPlatform(service.PlatformOpenAI).SetType(service.AccountTypeAPIKey).SetCredentials(credentials).SetExtra(map[string]any{}).Save(ctx)
	require.NoError(t, err)
	second, err := client.Account.Create().SetName("second").SetPlatform(service.PlatformAnthropic).SetType(service.AccountTypeAPIKey).SetCredentials(credentials).SetExtra(map[string]any{}).Save(ctx)
	require.NoError(t, err)
	_, err = client.Account.Create().SetName("other").SetPlatform(service.PlatformOpenAI).SetType(service.AccountTypeAPIKey).SetCredentials(map[string]any{"api_key": "other", "base_url": "https://ollama.com"}).SetExtra(map[string]any{}).Save(ctx)
	require.NoError(t, err)

	anchor := &service.Account{ID: first.ID, Name: first.Name, Platform: first.Platform, Type: first.Type, Credentials: credentials, Extra: map[string]any{}}
	require.NoError(t, repo.SaveOllamaCloudUsageSession(ctx, anchor, "ciphertext", true))

	for _, id := range []int64{first.ID, second.ID} {
		got, err := client.Account.Get(ctx, id)
		require.NoError(t, err)
		require.Equal(t, "ciphertext", got.Extra[service.OllamaCloudUsageSessionExtraKey])
		require.Equal(t, true, got.Extra[service.OllamaCloudUsageAutoRefreshExtraKey])
	}
}

func TestListDueOllamaCloudUsageAccountsSQLiteUsesActivityDueTime(t *testing.T) {
	repo, client := newOllamaCloudUsageSQLiteRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	fetched := now.Add(-time.Hour)
	lastAttempt := fetched
	nextRefresh := now.Add(time.Hour)
	account, err := client.Account.Create().SetName("due").SetPlatform(service.PlatformOpenAI).SetType(service.AccountTypeAPIKey).
		SetStatus(service.StatusActive).SetCredentials(map[string]any{"api_key": "due-key", "base_url": "https://ollama.com"}).
		SetLastUsedAt(now.Add(-10 * time.Minute)).SetExtra(map[string]any{
		service.OllamaCloudUsageSessionExtraKey: "ciphertext", service.OllamaCloudUsageAutoRefreshExtraKey: true,
		service.OllamaCloudUsageSnapshotExtraKey: map[string]any{"status": "ok", "fetched_at": fetched, "last_attempt_at": lastAttempt, "next_refresh_at": nextRefresh},
	}).Save(ctx)
	require.NoError(t, err)

	got, err := repo.ListDueOllamaCloudUsageAccounts(ctx, now, time.Minute, 2*time.Hour, 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, int64(account.ID), got[0].ID)
}
