package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestUsageBillingRepositoryApplySQLiteAppliesAllEffects(t *testing.T) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	for _, statement := range []string{
		`CREATE TABLE usage_billing_dedup (id INTEGER PRIMARY KEY AUTOINCREMENT, request_id TEXT, api_key_id INTEGER, request_fingerprint TEXT, UNIQUE(request_id, api_key_id))`,
		`CREATE TABLE usage_billing_dedup_archive (request_id TEXT, api_key_id INTEGER, request_fingerprint TEXT)`,
		`CREATE TABLE users (id INTEGER PRIMARY KEY, balance REAL, frozen_balance REAL DEFAULT 0, deleted_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE api_keys (id INTEGER PRIMARY KEY, quota REAL, quota_used REAL, status TEXT, usage_5h REAL, usage_1d REAL, usage_7d REAL, window_5h_start DATETIME, window_1d_start DATETIME, window_7d_start DATETIME, deleted_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE accounts (id INTEGER PRIMARY KEY, extra TEXT, deleted_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE scheduler_outbox (id INTEGER PRIMARY KEY AUTOINCREMENT, event_type TEXT, account_id INTEGER, group_id INTEGER, payload TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP, dedup_key TEXT UNIQUE)`,
		`INSERT INTO users (id, balance) VALUES (1, 10)`,
		`INSERT INTO api_keys (id, quota, quota_used, status, usage_5h, usage_1d, usage_7d, window_5h_start, window_1d_start, window_7d_start) VALUES (2, 2, 0, 'active', 0, 0, 0, '2020-01-01', '2020-01-01', '2020-01-01')`,
		`INSERT INTO accounts (id, extra) VALUES (3, '{"quota_limit":10,"quota_daily_limit":5,"quota_weekly_limit":8}')`,
	} {
		_, err = db.Exec(statement)
		require.NoError(t, err)
	}

	repo := NewUsageBillingRepository(nil, db)
	result, err := repo.Apply(context.Background(), &service.UsageBillingCommand{
		RequestID:           "sqlite-all-effects",
		APIKeyID:            2,
		UserID:              1,
		AccountID:           3,
		AccountType:         service.AccountTypeAPIKey,
		BalanceCost:         1.25,
		APIKeyQuotaCost:     2,
		APIKeyRateLimitCost: 3,
		AccountQuotaCost:    4,
	})
	require.NoError(t, err)
	require.True(t, result.Applied)
	require.True(t, result.APIKeyQuotaExhausted)
	require.NotNil(t, result.QuotaState)
	require.InDelta(t, 4, result.QuotaState.TotalUsed, 0.000001)
	require.InDelta(t, 4, result.QuotaState.DailyUsed, 0.000001)
	require.InDelta(t, 4, result.QuotaState.WeeklyUsed, 0.000001)

	var balance, quotaUsed, usage5h float64
	var status string
	require.NoError(t, db.QueryRow(`SELECT balance FROM users WHERE id = 1`).Scan(&balance))
	require.NoError(t, db.QueryRow(`SELECT quota_used, status, usage_5h FROM api_keys WHERE id = 2`).Scan(&quotaUsed, &status, &usage5h))
	require.InDelta(t, 8.75, balance, 0.000001)
	require.InDelta(t, 2, quotaUsed, 0.000001)
	require.Equal(t, service.StatusAPIKeyQuotaExhausted, status)
	require.InDelta(t, 3, usage5h, 0.000001)

	var dailyStart, weeklyStart string
	require.NoError(t, db.QueryRow(`SELECT json_extract(extra, '$.quota_daily_start'), json_extract(extra, '$.quota_weekly_start') FROM accounts WHERE id = 3`).Scan(&dailyStart, &weeklyStart))
	_, err = time.Parse(time.RFC3339Nano, dailyStart)
	require.NoError(t, err)
	_, err = time.Parse(time.RFC3339Nano, weeklyStart)
	require.NoError(t, err)

	duplicate, err := repo.Apply(context.Background(), &service.UsageBillingCommand{
		RequestID: "sqlite-all-effects", APIKeyID: 2, UserID: 1,
		AccountID: 3, AccountType: service.AccountTypeAPIKey,
		BalanceCost: 1.25, APIKeyQuotaCost: 2, APIKeyRateLimitCost: 3, AccountQuotaCost: 4,
	})
	require.NoError(t, err)
	require.False(t, duplicate.Applied)
}
