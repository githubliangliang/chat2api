package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/migrations"
)

func TestIsMigrationRawChecksumCompatible(t *testing.T) {
	content := []byte("SELECT 1;\n")
	sum := sha256.Sum256(content)
	rawChecksum := hex.EncodeToString(sum[:])

	require.True(t, isMigrationRawChecksumCompatible(rawChecksum, content))
	require.False(t, isMigrationRawChecksumCompatible(
		"0000000000000000000000000000000000000000000000000000000000000000",
		content,
	))
}

func TestIsMigrationChecksumCompatible(t *testing.T) {
	t.Run("001 SQLite历史checksum可兼容当前版本", func(t *testing.T) {
		ok := isMigrationChecksumCompatible(
			"001_init.sql",
			"233d8e2bbae47d42f2d9d4846c86d1d04761178787157d9db69486174d1484b4",
			"506edaa298f7e91f16bebff9b7254c6b55df42f1e0226f2222e42c3994372050",
		)
		require.True(t, ok)
	})

	t.Run("054历史checksum可兼容", func(t *testing.T) {
		ok := isMigrationChecksumCompatible(
			"054_drop_legacy_cache_columns.sql",
			"182c193f3359946cf094090cd9e57d5c3fd9abaffbc1e8fc378646b8a6fa12b4",
			"82de761156e03876653e7a6a4eee883cd927847036f779b0b9f34c42a8af7a7d",
		)
		require.True(t, ok)
	})

	t.Run("054在未知文件checksum下不兼容", func(t *testing.T) {
		ok := isMigrationChecksumCompatible(
			"054_drop_legacy_cache_columns.sql",
			"182c193f3359946cf094090cd9e57d5c3fd9abaffbc1e8fc378646b8a6fa12b4",
			"0000000000000000000000000000000000000000000000000000000000000000",
		)
		require.False(t, ok)
	})

	t.Run("061历史checksum可兼容", func(t *testing.T) {
		ok := isMigrationChecksumCompatible(
			"061_add_usage_log_request_type.sql",
			"08a248652cbab7cfde147fc6ef8cda464f2477674e20b718312faa252e0481c0",
			"66207e7aa5dd0429c2e2c0fabdaf79783ff157fa0af2e81adff2ee03790ec65c",
		)
		require.True(t, ok)
	})

	t.Run("061第二个历史checksum可兼容", func(t *testing.T) {
		ok := isMigrationChecksumCompatible(
			"061_add_usage_log_request_type.sql",
			"222b4a09c797c22e5922b6b172327c824f5463aaa8760e4f621bc5c22e2be0f3",
			"66207e7aa5dd0429c2e2c0fabdaf79783ff157fa0af2e81adff2ee03790ec65c",
		)
		require.True(t, ok)
	})

	t.Run("非白名单迁移不兼容", func(t *testing.T) {
		ok := isMigrationChecksumCompatible(
			"001_init.sql",
			"182c193f3359946cf094090cd9e57d5c3fd9abaffbc1e8fc378646b8a6fa12b4",
			"82de761156e03876653e7a6a4eee883cd927847036f779b0b9f34c42a8af7a7d",
		)
		require.False(t, ok)
	})

	t.Run("109历史checksum可兼容", func(t *testing.T) {
		ok := isMigrationChecksumCompatible(
			"109_auth_identity_compat_backfill.sql",
			"551e498aa5616d2d91096e9d72cf9fb36e418ee22eacc557f8811cadbc9e20ee",
			"0580b4602d85435edf9aca1633db580bb3932f26517f75134106f80275ec2ace",
		)
		require.True(t, ok)
	})

	t.Run("109当前checksum可兼容历史checksum", func(t *testing.T) {
		ok := isMigrationChecksumCompatible(
			"109_auth_identity_compat_backfill.sql",
			"551e498aa5616d2d91096e9d72cf9fb36e418ee22eacc557f8811cadbc9e20ee",
			"0580b4602d85435edf9aca1633db580bb3932f26517f75134106f80275ec2ace",
		)
		require.True(t, ok)
	})

	t.Run("109回滚到历史文件后仍兼容已应用的新checksum", func(t *testing.T) {
		ok := isMigrationChecksumCompatible(
			"109_auth_identity_compat_backfill.sql",
			"0580b4602d85435edf9aca1633db580bb3932f26517f75134106f80275ec2ace",
			"551e498aa5616d2d91096e9d72cf9fb36e418ee22eacc557f8811cadbc9e20ee",
		)
		require.True(t, ok)
	})

	t.Run("110历史checksum可兼容", func(t *testing.T) {
		ok := isMigrationChecksumCompatible(
			"110_pending_auth_and_provider_default_grants.sql",
			"e3d1f433be2b564cfbdc549adf98fce13c5c7b363ebc20fd05b765d0563b0925",
			"32cf87ee787b1bb36b5c691367c96eee37518fa3eed6f3322cf68795e3745279",
		)
		require.True(t, ok)
	})

	t.Run("112历史checksum可兼容", func(t *testing.T) {
		ok := isMigrationChecksumCompatible(
			"112_add_payment_order_provider_key_snapshot.sql",
			"ffd3e8a2c9295fa9cbefefd629a78268877e5b51bc970a82d9b3f46ec4ebd15e",
			"b75f8f56d39455682787696a3d92ad25b055444ca328fb7fca9a460a15d68d99",
		)
		require.True(t, ok)
	})

	t.Run("115历史checksum可兼容修复后的legacy external backfill", func(t *testing.T) {
		ok := isMigrationChecksumCompatible(
			"115_auth_identity_legacy_external_backfill.sql",
			"4cf39e508be9fd1a5aa41610cbbebeb80385c9adda45bf78a706de9db4f1385f",
			"022aadd97bb53e755f0cf7a3a957e0cb1a1353b0c39ec4de3234acd2871fd04f",
		)
		require.True(t, ok)
	})

	t.Run("116历史checksum可兼容修复后的legacy external safety reports", func(t *testing.T) {
		ok := isMigrationChecksumCompatible(
			"116_auth_identity_legacy_external_safety_reports.sql",
			"f7757bd929ac67ffb08ce69fa4cf20fad39dbff9d5a5085fb2adabb7607e5877",
			"07edb09fa8d04ffb172b0621e3c22f4d1757d20a24ae267b3b36b087ab72d488",
		)
		require.True(t, ok)
	})

	t.Run("119历史checksum可兼容占位文件", func(t *testing.T) {
		ok := isMigrationChecksumCompatible(
			"119_enforce_payment_orders_out_trade_no_unique.sql",
			"ebd2c67cce0116393fb4f1b5d5116a67c6aceb73820dfb5133d1ff6f36d72d34",
			"0bbe809ae48a9d811dabda1ba1c74955bd71c4a9cc610f9128816818dfa6c11e",
		)
		require.True(t, ok)
	})

	t.Run("118多个历史checksum都可兼容当前版本", func(t *testing.T) {
		for _, dbChecksum := range []string{
			"a38243ca0a72c3a01c0a92b7986423054d6133c0399441f853b99802852720fb",
			"e0cdf835d6c688d64100f483d31bc02ac9ebad414bf1837af239a84bf75b8227",
		} {
			ok := isMigrationChecksumCompatible(
				"118_wechat_dual_mode_and_auth_source_defaults.sql",
				dbChecksum,
				"b54194d7a3e4fbf710e0a3590d22a2fe7966804c487052a356e0b55f53ef96b0",
			)
			require.True(t, ok)
		}
	})

	t.Run("120多个历史checksum都可兼容新的notx修复版本", func(t *testing.T) {
		for _, dbChecksum := range []string{
			"e77921f79d539bc24575cb9c16cbe566d2b23ce816190343d0a7568f6a3fcf61",
			"707431450603e70a43ce9fbd61e0c12fa67da4875158ccefabacea069587ab22",
			"04b082b5a239c525154fe9185d324ee2b05ff90da9297e10dba19f9be79aa59a",
		} {
			ok := isMigrationChecksumCompatible(
				"120_enforce_payment_orders_out_trade_no_unique_notx.sql",
				dbChecksum,
				"34aadc0db59a4e390f92a12b73bd74642d9724f33124f73638ae00089ea5e074",
			)
			require.True(t, ok)
		}
	})

	t.Run("119未知checksum不兼容", func(t *testing.T) {
		ok := isMigrationChecksumCompatible(
			"119_enforce_payment_orders_out_trade_no_unique.sql",
			"ebd2c67cce0116393fb4f1b5d5116a67c6aceb73820dfb5133d1ff6f36d72d34",
			"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		)
		require.False(t, ok)
	})
}

// sqliteSeedTimestampRewrites 记录被 SQLite 化改写过的 seed 迁移：
// dbChecksum 是改写前（旧构建写进 schema_migrations）的值，fileChecksum 是当前文件的值。
var sqliteSeedTimestampRewrites = map[string]struct{ dbChecksum, fileChecksum string }{
	"110_pending_auth_and_provider_default_grants.sql": {
		"03fd65ec608f2dd0b5929d93b2c9af3a3d15973a80f37ff5ca00ca24d16493b5",
		"7d2b4484e088c1ea21f06b31a2738ac3068471f03efcb234e566ea965dcf68bf",
	},
	"111_payment_routing_and_scheduler_flags.sql": {
		"f2e205ac3cc8a812bbff5bdd7d7515e2d7f8c6bfee3e47dbfe190863db899816",
		"1c69e7af9ea0cb312e005994a3959fc8a7ff8cfd12a99dca5b3bf7214193f5f4",
	},
	"118_wechat_dual_mode_and_auth_source_defaults.sql": {
		"dd3a0a21ca6c3dc5f22067c8175c7ab3a2510c7640e7b75aac58024b7b7f6f42",
		"3bd0af162c0d95eb7b47f42ea0963b02100c417027980b658382ff397162e3b6",
	},
	"129_seed_claude_code_template.sql": {
		"b820187864d8077e0f78c57c33d323870aa09a5677810f2399249581ed58c5e5",
		"6ac26c78f7322868caabb4a47262a7c22112856a2bcb13dd998dff15d29e21be",
	},
	"139_seed_openai_monitor_templates.sql": {
		"93d2d36a9df5f122614549c1ce9808a207ca79032f5617a941b52782a5c726cd",
		"3f81be9c6336610c27ad5a2f28dc3d946baa8d154d863fc73166168efc68a4b3",
	},
	"195_channel_monitor_mode.sql": {
		"521b2598eb9c128df672218bc73abdc7255959068f5b57e4d70afe02ce9d2494",
		"22d5c8b6a52555039b769865a3a70c16230a767121e75e16ce337b12bd7395f5",
	},
	"204_channel_monitor_hide_throughput.sql": {
		"b02696eb4f646aafc25dd2e631281fdef770fcf2918f0708abe5ad709357738b",
		"5f80e58b97b3fe360f6c919fa11ffd40f1d95be20de545e2b6962b583d7b2070",
	},
}

// SQLite 化过程中，这批 seed 迁移被改写过一次：INSERT 补上 created_at / updated_at =
// datetime('now')（settings、channel_monitor_request_templates 的 NOT NULL 列）。
// 用改写前的构建迁移过的库里记的是旧 checksum，升级时必须仍能放行，否则启动直接失败。
//
// 每条用例同时校验「当前文件的 checksum」确实等于规则里登记的值：
// 再改这些文件时本测试会失败，提醒把新 checksum 也登记进 allowlist。
func TestMigrationChecksumCompatibility_SQLiteSeedTimestampRewrites(t *testing.T) {
	for name, want := range sqliteSeedTimestampRewrites {
		t.Run(name, func(t *testing.T) {
			content, err := fs.ReadFile(migrations.FS, name)
			require.NoError(t, err)
			sum := sha256.Sum256([]byte(strings.TrimSpace(string(content))))
			require.Equal(t, want.fileChecksum, hex.EncodeToString(sum[:]),
				"迁移文件已改动：请把新 checksum 加进 migrationChecksumCompatibilityRules 与本用例")

			require.True(t, isMigrationChecksumCompatible(name, want.dbChecksum, want.fileChecksum),
				"旧库的历史 checksum 必须仍可启动")
			require.False(t, isMigrationChecksumCompatible(
				name,
				"0000000000000000000000000000000000000000000000000000000000000000",
				want.fileChecksum,
			))
		})
	}
}
