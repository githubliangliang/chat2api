package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLatestAPIKeyIPIndexMigration(t *testing.T) {
	content, err := FS.ReadFile("174_add_usage_logs_api_key_latest_ip_index_notx.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_usage_logs_api_key_latest_ip")
	require.Contains(t, sql, "ON usage_logs (api_key_id, created_at DESC, id DESC)")
	require.Contains(t, sql, "WHERE ip_address IS NOT NULL AND ip_address <> ''")
	require.NotContains(t, sql, "CONCURRENTLY")
	require.NotContains(t, sql, "INCLUDE (ip_address)")
}
