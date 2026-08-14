package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/migrations"
)

// schema_migrations 记录已应用的迁移文件及其校验和。
// - filename: 迁移文件名（主键）
// - checksum: 文件内容 SHA256，用于检测迁移被篡改
// - applied_at: 应用时间（PG: TIMESTAMPTZ / SQLite: TEXT）
const schemaMigrationsTableDDLSQLite = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	filename   TEXT PRIMARY KEY,
	checksum   TEXT NOT NULL,
	applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);
`

const atlasSchemaRevisionsTableDDLSQLite = `
CREATE TABLE IF NOT EXISTS atlas_schema_revisions (
	version TEXT PRIMARY KEY,
	description TEXT NOT NULL,
	type INTEGER NOT NULL,
	applied INTEGER NOT NULL DEFAULT 0,
	total INTEGER NOT NULL DEFAULT 0,
	executed_at TEXT NOT NULL DEFAULT (datetime('now')),
	execution_time BIGINT NOT NULL DEFAULT 0,
	error TEXT NULL,
	error_stmt TEXT NULL,
	hash TEXT NOT NULL DEFAULT '',
	partial_hashes TEXT NULL,
	operator_version TEXT NULL
);
`

const nonTransactionalMigrationSuffix = "_notx.sql"
const paymentOrdersOutTradeNoUniqueMigration = "120_enforce_payment_orders_out_trade_no_unique_notx.sql"
const paymentOrdersOutTradeNoUniqueIndex = "paymentorder_out_trade_no_unique"
const schedulerOutboxPendingDedupKeyMigration = "153_scheduler_outbox_pending_dedup_key_index_notx.sql"
const schedulerOutboxPendingDedupKeyIndex = "idx_scheduler_outbox_pending_dedup_key"
const latestAPIKeyIPIndexMigration = "174_add_usage_logs_api_key_latest_ip_index_notx.sql"
const latestAPIKeyIPIndex = "idx_usage_logs_api_key_latest_ip"
const usageLogsUpstreamModelMismatchIndexMigration = "195_add_usage_log_upstream_model_mismatch_index_notx.sql"
const usageLogsUpstreamModelMismatchIndex = "idx_usage_logs_upstream_model_mismatch_created_at"

type migrationChecksumCompatibilityRule struct {
	fileChecksum       string
	acceptedDBChecksum map[string]struct{}
	acceptedChecksums  map[string]struct{}
}

// migrationChecksumCompatibilityRules 仅用于兼容历史上误修改过的迁移文件 checksum。
// 规则必须同时匹配「迁移名 + 数据库 checksum + 当前文件 checksum」且两者都落在该迁移的已知版本集合内才会放行，
// 避免放宽全局校验，也允许将误改的历史 migration 回滚为已发布版本而不要求人工修 checksum。
var migrationChecksumCompatibilityRules = map[string]migrationChecksumCompatibilityRule{
	// 001 was applied by the SQLite release before the migration checksum was
	// recalculated from trimmed content. Keep existing databases bootable while
	// retaining an exact filename/checksum allowlist.
	"001_init.sql": newMigrationChecksumCompatibilityRule(
		"506edaa298f7e91f16bebff9b7254c6b55df42f1e0226f2222e42c3994372050",
		"233d8e2bbae47d42f2d9d4846c86d1d04761178787157d9db69486174d1484b4",
	),
	"003_subscription.sql": newMigrationChecksumCompatibilityRule(
		"3584b0776ee4f611780f64b14cdbfa6eb15cf2bb60461ae990124bf8ec10bcdf",
		"43aa801af72b07b1de71f7816b45c1d20ac65579b4e1fc96628918f885b0636d",
	),
	"005_schema_parity.sql": newMigrationChecksumCompatibilityRule(
		"124b19826d952385e99e71bb439d7a06f6dc8c2d104ae6d98c27f6a806092f97",
		"f89fbc9e66cd4d12ca896e18688acdbb60f128f554c81c65c117989db430eec1",
	),
	"007_add_user_allowed_groups.sql": newMigrationChecksumCompatibilityRule(
		"166ec51bf5f7811248be74bf68d53c8aae60c89797dd867140fe95a6aebb6afc",
		"6059c7f38c74beb3de4fbe1a9e57f2c899fafe3c912645a57966cce10dec3833",
	),
	"012_add_user_subscription_soft_delete.sql": newMigrationChecksumCompatibilityRule(
		"65bf29db6c3896920efc6b2e781dee37c6ca0aa9f7cff051c2e3b05ddefea3a0",
		"7808c16bb31d21f9cadaf93afdad211cb6d1a37277aa03870ed4186e44518fcc",
	),
	"018_user_attributes.sql": newMigrationChecksumCompatibilityRule(
		"a2b4fc4bd733d9879013fac4ec6480353b0ddff0cd796d9f4173501de5651929",
		"c2ba5246dbcdbe08304ad90e90930dca4ebb468902dffe36ccf223c452a5e5b0",
	),
	"026_ops_metrics_aggregation_tables.sql": newMigrationChecksumCompatibilityRule(
		"41243e673302fca66411af826e098c96c6be9c978526e6871c6f0152e55e3fd7",
		"1f5ca6c26d9030f01386ef55ebd8efb33c093c642567bdb456aca9e17ad2040c",
	),
	"027_usage_billing_consistency.sql": newMigrationChecksumCompatibilityRule(
		"ec669ffee87a33c0fd77a0232cadc9e2034ebd6fe9afb6e73850482614a405eb",
		"a43d96c3be7507e6d5353fbca31269b4b37fb53f83f38190f1a9b9bd53473b3d",
	),
	"030_add_account_expires_at.sql": newMigrationChecksumCompatibilityRule(
		"da440856089f04cda3e9d52a936e2ef06050db60971bba3ec245e90db2035e1d",
		"96c62808ea398ffb0eea8110cf485716589ab6649d4cacfb1eed03cec51ded02",
	),
	"033_add_promo_codes.sql": newMigrationChecksumCompatibilityRule(
		"ffa53d98d9c53e49bf1d7fc5ef5dca1838d93019835257a42099a2562357cd76",
		"a39c7b76e32ed4d74f451d75b8b6b4cee718041fc1190bcea6a97c390c83d0c8",
	),
	"054_drop_legacy_cache_columns.sql":                       newMigrationChecksumCompatibilityRule("82de761156e03876653e7a6a4eee883cd927847036f779b0b9f34c42a8af7a7d", "182c193f3359946cf094090cd9e57d5c3fd9abaffbc1e8fc378646b8a6fa12b4"),
	"061_add_usage_log_request_type.sql":                      newMigrationChecksumCompatibilityRule("66207e7aa5dd0429c2e2c0fabdaf79783ff157fa0af2e81adff2ee03790ec65c", "08a248652cbab7cfde147fc6ef8cda464f2477674e20b718312faa252e0481c0", "222b4a09c797c22e5922b6b172327c824f5463aaa8760e4f621bc5c22e2be0f3"),
	"109_auth_identity_compat_backfill.sql":                   newMigrationChecksumCompatibilityRule("0580b4602d85435edf9aca1633db580bb3932f26517f75134106f80275ec2ace", "551e498aa5616d2d91096e9d72cf9fb36e418ee22eacc557f8811cadbc9e20ee"),
	"110_pending_auth_and_provider_default_grants.sql":        newMigrationChecksumCompatibilityRule("7d2b4484e088c1ea21f06b31a2738ac3068471f03efcb234e566ea965dcf68bf", "32cf87ee787b1bb36b5c691367c96eee37518fa3eed6f3322cf68795e3745279", "e3d1f433be2b564cfbdc549adf98fce13c5c7b363ebc20fd05b765d0563b0925", "03fd65ec608f2dd0b5929d93b2c9af3a3d15973a80f37ff5ca00ca24d16493b5"),
	"112_add_payment_order_provider_key_snapshot.sql":         newMigrationChecksumCompatibilityRule("b75f8f56d39455682787696a3d92ad25b055444ca328fb7fca9a460a15d68d99", "ffd3e8a2c9295fa9cbefefd629a78268877e5b51bc970a82d9b3f46ec4ebd15e"),
	"115_auth_identity_legacy_external_backfill.sql":          newMigrationChecksumCompatibilityRule("022aadd97bb53e755f0cf7a3a957e0cb1a1353b0c39ec4de3234acd2871fd04f", "4cf39e508be9fd1a5aa41610cbbebeb80385c9adda45bf78a706de9db4f1385f"),
	"116_auth_identity_legacy_external_safety_reports.sql":    newMigrationChecksumCompatibilityRule("07edb09fa8d04ffb172b0621e3c22f4d1757d20a24ae267b3b36b087ab72d488", "f7757bd929ac67ffb08ce69fa4cf20fad39dbff9d5a5085fb2adabb7607e5877"),
	"118_wechat_dual_mode_and_auth_source_defaults.sql":       newMigrationChecksumCompatibilityRule("3bd0af162c0d95eb7b47f42ea0963b02100c417027980b658382ff397162e3b6", "b54194d7a3e4fbf710e0a3590d22a2fe7966804c487052a356e0b55f53ef96b0", "e0cdf835d6c688d64100f483d31bc02ac9ebad414bf1837af239a84bf75b8227", "a38243ca0a72c3a01c0a92b7986423054d6133c0399441f853b99802852720fb", "dd3a0a21ca6c3dc5f22067c8175c7ab3a2510c7640e7b75aac58024b7b7f6f42"),
	"119_enforce_payment_orders_out_trade_no_unique.sql":      newMigrationChecksumCompatibilityRule("0bbe809ae48a9d811dabda1ba1c74955bd71c4a9cc610f9128816818dfa6c11e", "ebd2c67cce0116393fb4f1b5d5116a67c6aceb73820dfb5133d1ff6f36d72d34"),
	"120_enforce_payment_orders_out_trade_no_unique_notx.sql": newMigrationChecksumCompatibilityRule("34aadc0db59a4e390f92a12b73bd74642d9724f33124f73638ae00089ea5e074", "e77921f79d539bc24575cb9c16cbe566d2b23ce816190343d0a7568f6a3fcf61", "707431450603e70a43ce9fbd61e0c12fa67da4875158ccefabacea069587ab22", "04b082b5a239c525154fe9185d324ee2b05ff90da9297e10dba19f9be79aa59a"),
	"123_fix_legacy_auth_source_grant_on_signup_defaults.sql": newMigrationChecksumCompatibilityRule("2ce43c2cd89e9f9e1febd34a407ed9e84d177386c5544b6f02c1f58a21129f57", "6cd33422f215dcd1f486ab6f35c0ea5805d9ca69bb25906d94bc649156657145"),
	"159_batch_image_foundation.sql":                          newMigrationChecksumCompatibilityRule("d902b70982025ec519749faf058aab7631e82c3f48167b9a4ae4db718eb72cce", "82da85b5d98e67a0507647b873a40373e84538e4adafdeed6767c0ac8b6570b2"),
	"161_batch_image_pricing_snapshot.sql":                    newMigrationChecksumCompatibilityRule("4012af3e43636cb6af22e0176d59d1fcc70615c0f310194329461ae462c4fbd6", "96d915c9b7a6941ae99039e0ff3f1a61481eb9bddd933d11c6fadb2274554e87"),
	// 195 originally seeded mode=v2; flipped to v1 (safe default / opt-in v2). Existing DBs
	// that already applied the v2 seed keep their row and the historical checksum.
	"195_channel_monitor_mode.sql": newMigrationChecksumCompatibilityRule("22d5c8b6a52555039b769865a3a70c16230a767121e75e16ce337b12bd7395f5", "13f3792f3e3e53ee96e26415c884cf8062c77172824b54fcc9a8c0c2b1f185ec", "4c74fe33ef2274cc72e1bb49671e651274532c034b29f5b2982c2a4c88d101a6", "521b2598eb9c128df672218bc73abdc7255959068f5b57e4d70afe02ce9d2494"),
	// 这批 seed 迁移在 SQLite 化过程中被改写过一次：INSERT 补上 created_at / updated_at
	// = datetime('now')（settings / channel_monitor_request_templates 的 NOT NULL 列），
	// 只动 seed 的时间戳列，没有结构变更，因此已用旧版构建迁移过的库跳过重放是安全的。
	// 缺了这些历史 checksum 的话，旧库升级时会卡在 "checksum mismatch" 无法启动。
	"111_payment_routing_and_scheduler_flags.sql": newMigrationChecksumCompatibilityRule("1c69e7af9ea0cb312e005994a3959fc8a7ff8cfd12a99dca5b3bf7214193f5f4", "f2e205ac3cc8a812bbff5bdd7d7515e2d7f8c6bfee3e47dbfe190863db899816"),
	"129_seed_claude_code_template.sql":           newMigrationChecksumCompatibilityRule("6ac26c78f7322868caabb4a47262a7c22112856a2bcb13dd998dff15d29e21be", "b820187864d8077e0f78c57c33d323870aa09a5677810f2399249581ed58c5e5"),
	"139_seed_openai_monitor_templates.sql":       newMigrationChecksumCompatibilityRule("3f81be9c6336610c27ad5a2f28dc3d946baa8d154d863fc73166168efc68a4b3", "93d2d36a9df5f122614549c1ce9808a207ca79032f5617a941b52782a5c726cd"),
	"204_channel_monitor_hide_throughput.sql":     newMigrationChecksumCompatibilityRule("5f80e58b97b3fe360f6c919fa11ffd40f1d95be20de545e2b6962b583d7b2070", "b02696eb4f646aafc25dd2e631281fdef770fcf2918f0708abe5ad709357738b"),
	// 220 originally cleared video prices for all non-grok platforms (including composite);
	// composite is now preserved because it may route to Grok accounts.
	"220_clear_non_grok_video_generation_config.sql": newMigrationChecksumCompatibilityRule("85e320b9ec64f2d3fcd8cf705b2b4e76a7b49f7a57140c14bff97f32691c818b", "3da48c8fdffe6390325f43d08b8e353e0a365df43d44a78dbbe655d0deb18402"),
	"219_group_search_price_per_1k.sql":              newMigrationChecksumCompatibilityRule("e86786ebcc3b14206fd2d321380a4e50e80cdadbfcf4962c639255e6a14008db", "df6ffd71b97e30ec2c8fe7b95e15783042dea58c553e32701ee7c42a5619af80"),
	"218_group_audio_voice_pricing.sql":              newMigrationChecksumCompatibilityRule("40ee9f3a2af0e0a5e99dabc878fd0fe98be1011f26bcfcefcac7197f7081f0e7", "c2a5e5b4ffd6968ad1c10593289fbc11192cdea19fec3ed9bce3a84eff9a8351"),
}

// ApplyMigrations 将嵌入的 SQL 迁移文件应用到指定的数据库。
//
// 该函数可以在每次应用启动时安全调用：
// - 已应用的迁移会被自动跳过（通过校验 filename 判断）
// - 如果迁移文件内容被修改（checksum 不匹配），会返回错误
// - 使用 PostgreSQL Advisory Lock 确保多实例并发安全
//
// 参数：
//   - ctx: 上下文，用于超时控制和取消
//   - db: 数据库连接
//
// 返回：
//   - error: 迁移过程中的任何错误
func ApplyMigrations(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("nil sql db")
	}
	return applyMigrationsFS(ctx, db, migrations.FS)
}

// applyMigrationsFS 是迁移执行的核心实现。
// 它从指定的文件系统读取 SQL 迁移文件并按顺序应用。
//
// 迁移执行流程：
//  1. 获取 PostgreSQL Advisory Lock，防止多实例并发迁移
//  2. 确保 schema_migrations 表存在
//  3. 按文件名排序读取所有 .sql 文件
//  4. 对于每个迁移文件：
//     - 计算文件内容的 SHA256 校验和
//     - 检查该迁移是否已应用（通过 filename 查询）
//     - 如果已应用，验证校验和是否匹配
//     - 如果未应用，在事务中执行迁移并记录
//  5. 释放 Advisory Lock
//
// 参数：
//   - ctx: 上下文
//   - db: 数据库连接
//   - fsys: 包含迁移文件的文件系统（通常是 embed.FS）
func applyMigrationsFS(ctx context.Context, db *sql.DB, fsys fs.FS) error {
	if db == nil {
		return errors.New("nil sql db")
	}

	// Keep all migration work on one SQLite connection. SQLite serializes schema
	// writes itself, so no separate database advisory lock is required.
	lockConn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire migrations lock connection: %w", err)
	}
	defer func() { _ = lockConn.Close() }()
	sqlite := true

	// 创建迁移记录表（如果不存在）。
	// 该表记录所有已应用的迁移及其校验和。
	if _, err := lockConn.ExecContext(ctx, schemaMigrationsTableDDLSQLite); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	// 自动对齐 Atlas 基线（如果检测到 legacy schema_migrations 且缺失 atlas_schema_revisions）。
	if err := ensureAtlasBaselineAligned(ctx, lockConn, fsys, sqlite); err != nil {
		return err
	}

	// 获取所有 .sql 迁移文件并按文件名排序。
	// 命名规范：使用零填充数字前缀（如 001_init.sql, 002_add_users.sql）。
	files, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(files) // 确保按文件名顺序执行迁移

	for _, name := range files {
		// 读取迁移文件内容
		contentBytes, err := fs.ReadFile(fsys, name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		content := strings.TrimSpace(string(contentBytes))
		if content == "" {
			continue // 跳过空文件
		}

		// 计算文件内容的 SHA256 校验和，用于检测文件是否被修改。
		// 这是一种防篡改机制：如果有人修改了已应用的迁移文件，系统会拒绝启动。
		sum := sha256.Sum256([]byte(content))
		checksum := hex.EncodeToString(sum[:])

		// 检查该迁移是否已经应用
		var existing string
		rowErr := lockConn.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE filename = $1", name).Scan(&existing)
		if rowErr == nil {
			// 迁移已应用，验证校验和是否匹配
			if existing != checksum {
				// Early SQLite releases stored the checksum of the raw embedded file,
				// while the runner now hashes TrimSpace(content). Accept only the raw
				// checksum of this exact current file.
				if isMigrationRawChecksumCompatible(existing, contentBytes) {
					continue
				}
				// 兼容特定历史误改场景（仅白名单规则），其余仍保持严格不可变约束。
				if isMigrationChecksumCompatible(name, existing, checksum) {
					continue
				}
				// 校验和不匹配意味着迁移文件在应用后被修改，这是危险的。
				// 正确的做法是创建新的迁移文件来进行变更。
				return fmt.Errorf(
					"migration %s checksum mismatch (db=%s file=%s)\n"+
						"This means the migration file was modified after being applied to the database.\n"+
						"Solutions:\n"+
						"  1. Revert to original: git log --oneline -- migrations/%s && git checkout <commit> -- migrations/%s\n"+
						"  2. For new changes, create a new migration file instead of modifying existing ones\n"+
						"Note: Modifying applied migrations breaks the immutability principle and can cause inconsistencies across environments",
					name, existing, checksum, name, name,
				)
			}
			continue // 迁移已应用且校验和匹配，跳过
		}
		if !errors.Is(rowErr, sql.ErrNoRows) {
			return fmt.Errorf("check migration %s: %w", name, rowErr)
		}

		nonTx, err := validateMigrationExecutionMode(name, content)
		if err != nil {
			return fmt.Errorf("validate migration %s: %w", name, err)
		}

		if nonTx {
			if err := prepareNonTransactionalMigration(ctx, lockConn, name, sqlite); err != nil {
				return fmt.Errorf("prepare migration %s: %w", name, err)
			}

			// *_notx.sql：用于 CREATE/DROP INDEX CONCURRENTLY 场景，必须非事务执行。
			// 逐条语句执行，避免将多条 CONCURRENTLY 语句放入同一个隐式事务块。
			statements := splitSQLStatements(content)
			for i, stmt := range statements {
				trimmed := strings.TrimSpace(stmt)
				if trimmed == "" {
					continue
				}
				if stripSQLLineComment(trimmed) == "" {
					continue
				}
				if _, err := lockConn.ExecContext(ctx, trimmed); err != nil {
					if !(sqlite && isSQLiteIgnorableMigrationError(err)) {
						return fmt.Errorf("apply migration %s (non-tx statement %d): %w", name, i+1, err)
					}
				}
			}
			if _, err := lockConn.ExecContext(ctx, "INSERT INTO schema_migrations (filename, checksum) VALUES ($1, $2)", name, checksum); err != nil {
				return fmt.Errorf("record migration %s (non-tx): %w", name, err)
			}
			continue
		}

		// 默认迁移在事务中执行，确保原子性：要么完全成功，要么完全回滚。
		tx, err := lockConn.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}

		// 执行迁移 SQL。
		// SQLite 的 database/sql 驱动通常一次只接受一条语句，需拆分执行；
		// PostgreSQL 可整文件一次 Exec。
		if sqlite {
			for i, stmt := range splitSQLStatements(content) {
				trimmed := strings.TrimSpace(stmt)
				if trimmed == "" || stripSQLLineComment(trimmed) == "" {
					continue
				}
				if _, err := tx.ExecContext(ctx, trimmed); err != nil {
					if isSQLiteIgnorableMigrationError(err) {
						continue
					}
					_ = tx.Rollback()
					return fmt.Errorf("apply migration %s (statement %d): %w", name, i+1, err)
				}
			}
		} else if _, err := tx.ExecContext(ctx, content); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}

		// 记录迁移已完成，保存文件名和校验和
		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (filename, checksum) VALUES ($1, $2)", name, checksum); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}

		// 提交事务
		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}

	return nil
}

// isSQLiteIgnorableMigrationError reports benign SQLite errors when replaying
// migrations against a schema that was first created by Ent Schema.Create
// (columns/indexes/tables already present, or DROP COLUMN of missing cols).
func isSQLiteIgnorableMigrationError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "duplicate column name"):
		return true
	case strings.Contains(msg, "already exists"):
		return true
	case strings.Contains(msg, "no such column"):
		return true
	case strings.Contains(msg, "no such index"):
		return true
	// UNIQUE index create can fail if historical duplicates exist; caller may
	// still need manual cleanup, but empty personal DBs should not hit this.
	default:
		return false
	}
}

type migrationConnection interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

func prepareNonTransactionalMigration(ctx context.Context, db migrationConnection, name string, sqlite bool) error {
	switch name {
	case paymentOrdersOutTradeNoUniqueMigration:
		return preparePaymentOrdersOutTradeNoUniqueMigration(ctx, db, sqlite)
	case schedulerOutboxPendingDedupKeyMigration:
		return dropInvalidIndexIfPresent(ctx, db, schedulerOutboxPendingDedupKeyIndex, sqlite)
	case latestAPIKeyIPIndexMigration:
		return dropInvalidIndexIfPresent(ctx, db, latestAPIKeyIPIndex, sqlite)
	case usageLogsUpstreamModelMismatchIndexMigration:
		return dropInvalidIndexIfPresent(ctx, db, usageLogsUpstreamModelMismatchIndex, sqlite)
	default:
		return nil
	}
}

func preparePaymentOrdersOutTradeNoUniqueMigration(ctx context.Context, db migrationConnection, sqlite bool) error {
	duplicates, err := findDuplicatePaymentOrderOutTradeNos(ctx, db)
	if err != nil {
		return fmt.Errorf("precheck duplicate out_trade_no: %w", err)
	}
	if len(duplicates) > 0 {
		return fmt.Errorf(
			"duplicate out_trade_no values block %s; remediate duplicates before retrying: %s",
			paymentOrdersOutTradeNoUniqueMigration,
			strings.Join(duplicates, ", "),
		)
	}

	return dropInvalidIndexIfPresent(ctx, db, paymentOrdersOutTradeNoUniqueIndex, sqlite)
}

func dropInvalidIndexIfPresent(ctx context.Context, db migrationConnection, indexName string, sqlite bool) error {
	invalid, err := indexIsInvalid(ctx, db, indexName, sqlite)
	if err != nil {
		return fmt.Errorf("check invalid index %s: %w", indexName, err)
	}
	if !invalid {
		return nil
	}

	// CONCURRENTLY is PostgreSQL-only; SQLite migrations use plain DROP INDEX.
	if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName)); err != nil {
		return fmt.Errorf("drop invalid index %s: %w", indexName, err)
	}
	return nil
}

func findDuplicatePaymentOrderOutTradeNos(ctx context.Context, db migrationConnection) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT out_trade_no, COUNT(*) AS duplicate_count
		FROM payment_orders
		WHERE out_trade_no <> ''
		GROUP BY out_trade_no
		HAVING COUNT(*) > 1
		ORDER BY duplicate_count DESC, out_trade_no
		LIMIT 5
	`)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	duplicates := make([]string, 0, 5)
	for rows.Next() {
		var outTradeNo string
		var duplicateCount int
		if err := rows.Scan(&outTradeNo, &duplicateCount); err != nil {
			return nil, err
		}
		duplicates = append(duplicates, fmt.Sprintf("%s (count=%d)", outTradeNo, duplicateCount))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return duplicates, nil
}

func indexIsInvalid(ctx context.Context, db migrationConnection, indexName string, sqlite bool) (bool, error) {
	// SQLite has no "invalid index" state like PostgreSQL concurrent builds.
	if sqlite {
		return false, nil
	}
	var invalid bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM pg_class idx
			JOIN pg_namespace ns ON ns.oid = idx.relnamespace
			JOIN pg_index i ON i.indexrelid = idx.oid
			WHERE ns.nspname = 'public'
			  AND idx.relname = $1
			  AND NOT i.indisvalid
		)
	`, indexName).Scan(&invalid)
	return invalid, err
}

func ensureAtlasBaselineAligned(ctx context.Context, db migrationConnection, fsys fs.FS, sqlite bool) error {
	hasLegacy, err := tableExists(ctx, db, "schema_migrations", sqlite)
	if err != nil {
		return fmt.Errorf("check schema_migrations: %w", err)
	}
	if !hasLegacy {
		return nil
	}

	hasAtlas, err := tableExists(ctx, db, "atlas_schema_revisions", sqlite)
	if err != nil {
		return fmt.Errorf("check atlas_schema_revisions: %w", err)
	}
	if !hasAtlas {
		if _, err := db.ExecContext(ctx, atlasSchemaRevisionsTableDDLSQLite); err != nil {
			return fmt.Errorf("create atlas_schema_revisions: %w", err)
		}
	}

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM atlas_schema_revisions").Scan(&count); err != nil {
		return fmt.Errorf("count atlas_schema_revisions: %w", err)
	}
	if count > 0 {
		return nil
	}

	version, description, hash, err := latestMigrationBaseline(fsys)
	if err != nil {
		return fmt.Errorf("atlas baseline version: %w", err)
	}

	insertSQL := fmt.Sprintf(`
		INSERT INTO atlas_schema_revisions (version, description, type, applied, total, executed_at, execution_time, hash)
		VALUES ($1, $2, $3, 0, 0, %s, 0, $4)
	`, "datetime('now')")
	if _, err := db.ExecContext(ctx, insertSQL, version, description, 1, hash); err != nil {
		return fmt.Errorf("insert atlas baseline: %w", err)
	}
	return nil
}

func tableExists(ctx context.Context, db migrationConnection, tableName string, sqlite bool) (bool, error) {
	if sqlite {
		var n int
		err := db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=$1`,
			tableName,
		).Scan(&n)
		return n > 0, err
	}
	var exists bool
	err := db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		)
	`, tableName).Scan(&exists)
	return exists, err
}

func latestMigrationBaseline(fsys fs.FS) (string, string, string, error) {
	files, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return "", "", "", err
	}
	if len(files) == 0 {
		return "baseline", "baseline", "", nil
	}
	sort.Strings(files)
	name := files[len(files)-1]
	contentBytes, err := fs.ReadFile(fsys, name)
	if err != nil {
		return "", "", "", err
	}
	content := strings.TrimSpace(string(contentBytes))
	sum := sha256.Sum256([]byte(content))
	hash := hex.EncodeToString(sum[:])
	version := strings.TrimSuffix(name, ".sql")
	return version, version, hash, nil
}

func checksumSet(values ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func isMigrationRawChecksumCompatible(dbChecksum string, content []byte) bool {
	sum := sha256.Sum256(content)
	return dbChecksum == hex.EncodeToString(sum[:])
}

func newMigrationChecksumCompatibilityRule(fileChecksum string, acceptedDBChecksums ...string) migrationChecksumCompatibilityRule {
	return migrationChecksumCompatibilityRule{
		fileChecksum:       fileChecksum,
		acceptedDBChecksum: checksumSet(acceptedDBChecksums...),
		acceptedChecksums:  checksumSet(append([]string{fileChecksum}, acceptedDBChecksums...)...),
	}
}

func isMigrationChecksumCompatible(name, dbChecksum, fileChecksum string) bool {
	rule, ok := migrationChecksumCompatibilityRules[name]
	if !ok {
		return false
	}
	_, dbOK := rule.acceptedChecksums[dbChecksum]
	if !dbOK {
		return false
	}
	_, fileOK := rule.acceptedChecksums[fileChecksum]
	return fileOK
}

func validateMigrationExecutionMode(name, content string) (bool, error) {
	normalizedName := strings.ToLower(strings.TrimSpace(name))
	upperContent := strings.ToUpper(content)
	nonTx := strings.HasSuffix(normalizedName, nonTransactionalMigrationSuffix)

	if !nonTx {
		if strings.Contains(upperContent, "CONCURRENTLY") {
			return false, errors.New("CONCURRENTLY statements must be placed in *_notx.sql migrations")
		}
		return false, nil
	}

	if strings.Contains(upperContent, "BEGIN") || strings.Contains(upperContent, "COMMIT") || strings.Contains(upperContent, "ROLLBACK") {
		return false, errors.New("*_notx.sql must not contain transaction control statements (BEGIN/COMMIT/ROLLBACK)")
	}

	statements := splitSQLStatements(content)
	for _, stmt := range statements {
		normalizedStmt := strings.ToUpper(stripSQLLineComment(strings.TrimSpace(stmt)))
		if normalizedStmt == "" {
			continue
		}

		isCreateIndex := strings.Contains(normalizedStmt, "CREATE") && strings.Contains(normalizedStmt, "INDEX")
		isDropIndex := strings.Contains(normalizedStmt, "DROP") && strings.Contains(normalizedStmt, "INDEX")
		// SQLite dialect drops CONCURRENTLY; *_notx.sql still hosts index DDL only.
		if !isCreateIndex && !isDropIndex {
			return false, errors.New("*_notx.sql currently only supports CREATE/DROP INDEX statements")
		}
		if isCreateIndex && !strings.Contains(normalizedStmt, "IF NOT EXISTS") {
			return false, errors.New("CREATE INDEX in *_notx.sql must include IF NOT EXISTS for idempotency")
		}
		if isDropIndex && !strings.Contains(normalizedStmt, "IF EXISTS") {
			return false, errors.New("DROP INDEX in *_notx.sql must include IF EXISTS for idempotency")
		}
	}

	return true, nil
}

// splitSQLStatements splits SQL on ';' while ignoring semicolons inside:
// - line comments (-- ...)
// - block comments (/* ... */)
// - single-quoted string literals
// - double-quoted identifiers
// Naive strings.Split on ';' breaks CREATE TABLE statements when comments
// contain "optional; ..." (see 033_ops_monitoring_vnext.sql).
func splitSQLStatements(content string) []string {
	var out []string
	var b strings.Builder
	b.Grow(len(content) / 4)

	inLineComment := false
	inBlockComment := false
	inSingleQuote := false
	inDoubleQuote := false

	flush := func() {
		part := strings.TrimSpace(b.String())
		b.Reset()
		if part != "" {
			out = append(out, part)
		}
	}

	for i := 0; i < len(content); i++ {
		ch := content[i]
		next := byte(0)
		if i+1 < len(content) {
			next = content[i+1]
		}

		if inLineComment {
			b.WriteByte(ch)
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			b.WriteByte(ch)
			if ch == '*' && next == '/' {
				b.WriteByte(next)
				i++
				inBlockComment = false
			}
			continue
		}
		if inSingleQuote {
			b.WriteByte(ch)
			// SQL escapes single quotes by doubling them
			if ch == '\'' {
				if next == '\'' {
					b.WriteByte(next)
					i++
					continue
				}
				inSingleQuote = false
			}
			continue
		}
		if inDoubleQuote {
			b.WriteByte(ch)
			if ch == '"' {
				if next == '"' {
					b.WriteByte(next)
					i++
					continue
				}
				inDoubleQuote = false
			}
			continue
		}

		// entering comment / string / identifier
		if ch == '-' && next == '-' {
			b.WriteByte(ch)
			b.WriteByte(next)
			i++
			inLineComment = true
			continue
		}
		if ch == '/' && next == '*' {
			b.WriteByte(ch)
			b.WriteByte(next)
			i++
			inBlockComment = true
			continue
		}
		if ch == '\'' {
			b.WriteByte(ch)
			inSingleQuote = true
			continue
		}
		if ch == '"' {
			b.WriteByte(ch)
			inDoubleQuote = true
			continue
		}

		if ch == ';' {
			flush()
			continue
		}
		b.WriteByte(ch)
	}
	flush()
	return out
}

func stripSQLLineComment(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			lines[i] = line[:idx]
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
