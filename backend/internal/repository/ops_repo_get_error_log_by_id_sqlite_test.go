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

// Regression: personal SQLite deploys 500 on request-error detail and correlated
// upstream-error endpoints when their queries use PostgreSQL-only SQL.
func TestOpsRequestErrorQueries_SQLite_NoPostgresOnlySyntax(t *testing.T) {
	registerSQLitePGCompatFunctions()

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", t.Name()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	for _, stmt := range []string{
		`CREATE TABLE ops_error_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			request_id VARCHAR(64),
			client_request_id VARCHAR(64),
			user_id BIGINT,
			api_key_id BIGINT,
			account_id BIGINT,
			group_id BIGINT,
			client_ip TEXT,
			platform VARCHAR(32),
			model VARCHAR(100),
			request_path VARCHAR(256),
			stream BOOLEAN NOT NULL DEFAULT false,
			user_agent TEXT,
			error_phase VARCHAR(32) NOT NULL,
			error_type VARCHAR(64) NOT NULL,
			severity VARCHAR(8) NOT NULL DEFAULT 'P2',
			status_code INT,
			is_business_limited BOOLEAN NOT NULL DEFAULT false,
			is_count_tokens BOOLEAN NOT NULL DEFAULT false,
			error_message TEXT,
			error_body TEXT,
			error_source VARCHAR(64),
			error_owner VARCHAR(32),
			upstream_status_code INT,
			upstream_error_message TEXT,
			upstream_error_detail TEXT,
			upstream_errors TEXT,
			auth_latency_ms BIGINT,
			routing_latency_ms BIGINT,
			upstream_latency_ms BIGINT,
			response_latency_ms BIGINT,
			time_to_first_token_ms BIGINT,
			created_at DATETIME NOT NULL DEFAULT (datetime('now')),
			resolved BOOLEAN NOT NULL DEFAULT false,
			resolved_at DATETIME,
			resolved_by_user_id BIGINT,
			inbound_endpoint VARCHAR(256),
			upstream_endpoint VARCHAR(256),
			requested_model VARCHAR(100),
			upstream_model VARCHAR(100),
			request_type SMALLINT,
			api_key_prefix VARCHAR(32)
		)`,
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT
		)`,
		`CREATE TABLE accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT
		)`,
		`CREATE TABLE groups (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT
		)`,
		`CREATE TABLE api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			deleted_at DATETIME
		)`,
	} {
		_, err = db.Exec(stmt)
		require.NoError(t, err)
	}

	repo := NewOpsRepository(db).(*opsRepository)
	ctx := context.Background()

	upstreamJSON := `[{"kind":"http","status":502}]`
	clientIP := "203.0.113.10"
	createdAt := time.Now().UTC()
	requestErrorID, err := repo.InsertErrorLog(ctx, &service.OpsInsertErrorLogInput{
		RequestID:          "request-120",
		ErrorPhase:         "request",
		ErrorType:          "request_error",
		ErrorOwner:         "client",
		Severity:           "error",
		StatusCode:         500,
		ClientIP:           &clientIP,
		UpstreamErrorsJSON: &upstreamJSON,
		CreatedAt:          createdAt,
		APIKeyPrefix:       "sk-test",
	})
	require.NoError(t, err)
	require.Positive(t, requestErrorID)

	detail, err := repo.GetErrorLogByID(ctx, requestErrorID)
	require.NoError(t, err)
	require.Equal(t, requestErrorID, detail.ID)
	require.Equal(t, 500, detail.StatusCode)
	require.Equal(t, "sk-test", detail.APIKeyPrefix)
	require.Equal(t, upstreamJSON, detail.UpstreamErrors)
	require.NotNil(t, detail.ClientIP)
	require.Equal(t, "203.0.113.10", *detail.ClientIP)

	upstreamErrorID, err := repo.InsertErrorLog(ctx, &service.OpsInsertErrorLogInput{
		RequestID:  "request-120",
		ErrorPhase: "upstream",
		ErrorType:  "upstream_error",
		ErrorOwner: "provider",
		Severity:   "error",
		StatusCode: 502,
		CreatedAt:  createdAt,
	})
	require.NoError(t, err)
	require.Positive(t, upstreamErrorID)

	startTime := createdAt.Add(-time.Hour)
	endTime := createdAt.Add(time.Hour)
	list, err := repo.ListErrorLogs(ctx, &service.OpsErrorLogFilter{
		Page:                     1,
		PageSize:                 100,
		StartTime:                &startTime,
		EndTime:                  &endTime,
		View:                     "all",
		ErrorPhasesAny:           []string{"upstream", "account_auth"},
		IncludeRecoveredUpstream: true,
		Owner:                    "provider",
		RequestID:                "request-120",
	})
	require.NoError(t, err)
	require.Equal(t, 1, list.Total)
	require.Len(t, list.Errors, 1)
	require.Equal(t, upstreamErrorID, list.Errors[0].ID)

	upstreamDetail, err := repo.GetErrorLogByID(ctx, upstreamErrorID)
	require.NoError(t, err)
	require.Equal(t, upstreamErrorID, upstreamDetail.ID)
	require.Equal(t, 502, upstreamDetail.StatusCode)
}
