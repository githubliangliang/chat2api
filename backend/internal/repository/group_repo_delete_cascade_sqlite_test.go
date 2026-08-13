package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func newGroupEntRepoSQLite(t *testing.T) (*groupRepository, *dbent.Client) {
	t.Helper()

	// Soft-delete cascade SQL uses NOW(); register before opening connections.
	registerSQLitePGCompatFunctions()

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", t.Name()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	// DeleteCascade raw SQL also touches join tables that may not exist on a
	// bare Ent schema; create minimal stubs when missing.
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS user_allowed_groups (
			created_at datetime NOT NULL,
			user_id integer NOT NULL,
			group_id integer NOT NULL,
			PRIMARY KEY (user_id, group_id)
		)`,
		`CREATE TABLE IF NOT EXISTS account_groups (
			priority integer NOT NULL DEFAULT 50,
			created_at datetime NOT NULL,
			account_id integer NOT NULL,
			group_id integer NOT NULL,
			PRIMARY KEY (account_id, group_id)
		)`,
		`CREATE TABLE IF NOT EXISTS scheduler_outbox (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			event_type TEXT NOT NULL,
			account_id INTEGER,
			group_id INTEGER,
			payload TEXT,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			dedup_key TEXT
		)`,
	} {
		_, err = db.Exec(stmt)
		require.NoError(t, err)
	}

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	return newGroupRepositoryWithSQL(client, db), client
}

// Regression: SQLite rejects "SELECT ... FOR UPDATE". DeleteCascade must not
// use PostgreSQL row-lock syntax on the personal SQLite deploy path.
func TestGroupRepository_DeleteCascade_SQLite_NoForUpdate(t *testing.T) {
	repo, _ := newGroupEntRepoSQLite(t)
	ctx := context.Background()

	g := &service.Group{
		Name:             "sqlite-delete-cascade",
		Platform:         service.PlatformAnthropic,
		RateMultiplier:   1.0,
		Status:           service.StatusActive,
		SubscriptionType: service.SubscriptionTypeStandard,
	}
	require.NoError(t, repo.Create(ctx, g))
	require.NotZero(t, g.ID)

	affected, err := repo.DeleteCascade(ctx, g.ID)
	require.NoError(t, err)
	require.Empty(t, affected)

	_, err = repo.GetByID(ctx, g.ID)
	require.ErrorIs(t, err, service.ErrGroupNotFound)
}

func TestGroupRepository_DeleteCascade_SQLite_AlreadyDeleted(t *testing.T) {
	repo, _ := newGroupEntRepoSQLite(t)
	ctx := context.Background()

	g := &service.Group{
		Name:             "sqlite-delete-twice",
		Platform:         service.PlatformAnthropic,
		RateMultiplier:   1.0,
		Status:           service.StatusActive,
		SubscriptionType: service.SubscriptionTypeStandard,
	}
	require.NoError(t, repo.Create(ctx, g))
	require.NoError(t, repo.Delete(ctx, g.ID))

	_, err := repo.DeleteCascade(ctx, g.ID)
	require.ErrorIs(t, err, service.ErrGroupNotFound)
}
