package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestAffiliateRepositoryAccrueFrozenQuotaRunsOnSQLite(t *testing.T) {
	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared&_time_format=sqlite&_fk=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	require.NoError(t, ApplyMigrations(context.Background(), db))

	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	t.Cleanup(func() { _ = client.Close() })

	const inviterID = 101
	const inviteeID = 102
	_, err = db.Exec(`INSERT INTO users (id, email, password_hash, role, status, concurrency, balance, created_at, updated_at)
		VALUES (?, 'inviter@example.com', 'hash', 'user', 'active', 1, 0, datetime('now'), datetime('now')),
		       (?, 'invitee@example.com', 'hash', 'user', 'active', 1, 0, datetime('now'), datetime('now'))`, inviterID, inviteeID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO user_affiliates (user_id, aff_code, created_at, updated_at)
		VALUES (?, 'INVITER00001', datetime('now'), datetime('now'))`, inviterID)
	require.NoError(t, err)

	repo := &affiliateRepository{client: client}
	before := time.Now().UTC().Add(90 * time.Minute)
	applied, err := repo.AccrueQuota(context.Background(), inviterID, inviteeID, 3.5, 2, nil)
	require.NoError(t, err)
	require.True(t, applied)

	var frozenQuota float64
	var frozenUntilText string
	require.NoError(t, db.QueryRow(`SELECT ua.aff_frozen_quota, l.frozen_until
		FROM user_affiliates ua JOIN user_affiliate_ledger l ON l.user_id = ua.user_id
		WHERE ua.user_id = ? AND l.action = 'accrue'`, inviterID).Scan(&frozenQuota, &frozenUntilText))
	require.InDelta(t, 3.5, frozenQuota, 1e-9)
	frozenUntil, ok := parseSQLiteTime(frozenUntilText)
	require.True(t, ok, "parse frozen_until %q", frozenUntilText)
	require.True(t, frozenUntil.After(before), "frozen_until = %s, want after %s", frozenUntil, before)

	_, err = db.Exec(`UPDATE user_affiliate_ledger SET frozen_until = datetime('now', '-1 minute') WHERE user_id = ?`, inviterID)
	require.NoError(t, err)
	thawed, err := repo.ThawFrozenQuota(context.Background(), inviterID)
	require.NoError(t, err)
	require.InDelta(t, 3.5, thawed, 1e-9)

	transferred, balance, err := repo.TransferQuotaToBalance(context.Background(), inviterID)
	require.NoError(t, err)
	require.InDelta(t, 3.5, transferred, 1e-9)
	require.InDelta(t, 3.5, balance, 1e-9)

	var transferCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM user_affiliate_ledger
		WHERE user_id = ? AND action = 'transfer' AND balance_after = 3.5 AND aff_quota_after = 0`, inviterID).Scan(&transferCount))
	require.Equal(t, 1, transferCount)
}
