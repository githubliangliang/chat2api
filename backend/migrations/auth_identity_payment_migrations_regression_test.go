package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration112AddsProviderKeyWithSQLiteSyntax(t *testing.T) {
	content, err := FS.ReadFile("112_add_payment_order_provider_key_snapshot.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "ADD COLUMN provider_key VARCHAR(30)")
	require.Contains(t, sql, "CAST(id AS TEXT)")
	require.NotContains(t, sql, "::text")
}

func TestMigration118DoesNotForceOverwriteAuthSourceGrantDefaults(t *testing.T) {
	content, err := FS.ReadFile("118_wechat_dual_mode_and_auth_source_defaults.sql")
	require.NoError(t, err)

	sql := string(content)
	require.NotContains(t, sql, "UPDATE settings")
	require.NotContains(t, sql, "SET value = 'false'")
	require.True(t, strings.Contains(sql, "ON CONFLICT (key) DO NOTHING"))
	require.Contains(t, sql, "THEN ''")
}

func TestMigration120CreatesPaymentOrderUniqueIndexWithSQLiteSyntax(t *testing.T) {
	followupContent, err := FS.ReadFile("120_enforce_payment_orders_out_trade_no_unique_notx.sql")
	require.NoError(t, err)

	followupSQL := string(followupContent)
	require.Contains(t, followupSQL, "explicit duplicate out_trade_no precheck")
	require.Contains(t, followupSQL, "stale invalid paymentorder_out_trade_no_unique index")
	require.Contains(t, followupSQL, "CREATE UNIQUE INDEX IF NOT EXISTS paymentorder_out_trade_no_unique")
	require.Contains(t, followupSQL, "DROP INDEX  IF EXISTS paymentorder_out_trade_no")
	require.Contains(t, followupSQL, "WHERE out_trade_no <> ''")
	require.NotContains(t, followupSQL, "CONCURRENTLY")
}

func TestMigration110SeedsAuthSourceSignupGrantsDisabledByDefault(t *testing.T) {
	content, err := FS.ReadFile("110_pending_auth_and_provider_default_grants.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "('auth_source_default_email_grant_on_signup', 'false', datetime('now'))")
	require.Contains(t, sql, "('auth_source_default_linuxdo_grant_on_signup', 'false', datetime('now'))")
	require.Contains(t, sql, "('auth_source_default_oidc_grant_on_signup', 'false', datetime('now'))")
	require.Contains(t, sql, "('auth_source_default_wechat_grant_on_signup', 'false', datetime('now'))")
	require.NotContains(t, sql, "('auth_source_default_email_grant_on_signup', 'true')")
}

func TestMigration135DropsLegacyProviderIndexesWithSQLiteSyntax(t *testing.T) {
	content, err := FS.ReadFile("135_allow_email_oauth_provider_types.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "DROP INDEX IF EXISTS users_signup_source_check")
	require.Contains(t, sql, "DROP INDEX IF EXISTS auth_identities_provider_type_check")
	require.Contains(t, sql, "DROP INDEX IF EXISTS auth_identity_channels_provider_type_check")
	require.Contains(t, sql, "DROP INDEX IF EXISTS pending_auth_sessions_provider_type_check")
}

func TestMigration151AddsAccountAutoPauseExpiryPartialIndex(t *testing.T) {
	content, err := FS.ReadFile("151_account_autopause_expiry_index_notx.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_accounts_autopause_expiry_due")
	require.Contains(t, sql, "ON accounts (expires_at)")
	require.Contains(t, sql, "WHERE deleted_at IS NULL")
	require.Contains(t, sql, "schedulable = TRUE")
	require.Contains(t, sql, "auto_pause_on_expired = TRUE")
	require.Contains(t, sql, "expires_at IS NOT NULL")
}

func TestMigration158BackfillsGrokMediaGenerationGroups(t *testing.T) {
	content, err := FS.ReadFile("158_enable_grok_media_generation_groups.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "UPDATE groups")
	require.Contains(t, sql, "SET allow_image_generation = true")
	require.Contains(t, sql, "WHERE platform = 'grok'")
	require.Contains(t, sql, "AND allow_image_generation = false")
}

func TestMigration154AddsSparkShadowColumnsAndConstraintsWithoutHotIndexes(t *testing.T) {
	content, err := FS.ReadFile("154_account_spark_shadow.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "ADD COLUMN parent_account_id BIGINT")
	require.Contains(t, sql, "ADD COLUMN quota_dimension VARCHAR(20) NOT NULL DEFAULT 'global'")
	require.NotContains(t, sql, "CREATE INDEX")
	require.NotContains(t, sql, "CREATE UNIQUE INDEX")
	require.NotContains(t, sql, "CONCURRENTLY")
}

func TestMigration154aAddsSparkShadowIndexesWithSQLiteSyntax(t *testing.T) {
	content, err := FS.ReadFile("154a_account_spark_shadow_indexes_notx.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_accounts_parent_account_id")
	require.Contains(t, sql, "CREATE UNIQUE INDEX IF NOT EXISTS uq_accounts_spark_shadow_per_parent")
	require.Contains(t, sql, "ON accounts (parent_account_id)")
	require.Contains(t, sql, "WHERE parent_account_id IS NOT NULL")
	require.Contains(t, sql, "quota_dimension = 'spark'")
	require.Contains(t, sql, "deleted_at IS NULL")
	require.NotContains(t, sql, "CONCURRENTLY")
}
