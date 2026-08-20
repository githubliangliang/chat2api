// Package repository 提供应用程序的基础设施层组件。
// 包括数据库连接初始化、ORM 客户端管理、Redis 连接、数据库迁移等核心功能。
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/migrations"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite" // SQLite driver (pure Go)
)

// InitEnt 初始化 Ent ORM 客户端并返回客户端实例和底层的 *sql.DB。
//
// 该函数执行以下操作：
//  1. 初始化全局时区设置，确保时间处理一致性
//  2. 建立 SQLite 数据库连接
//  3. 自动执行数据库迁移，确保 schema 与代码同步
//  4. 创建并返回 Ent 客户端实例
//
// 重要提示：调用者必须负责关闭返回的 ent.Client（关闭时会自动关闭底层的 driver/db）。
//
// 参数：
//   - cfg: 应用程序配置，包含数据库连接信息和时区设置
//
// 返回：
//   - *ent.Client: Ent ORM 客户端，用于执行数据库操作
//   - *sql.DB: 底层的 SQL 数据库连接，可用于直接执行原生 SQL
//   - error: 初始化过程中的错误
func InitEnt(cfg *config.Config) (*ent.Client, *sql.DB, error) {
	// 优先初始化时区设置，确保所有时间操作使用统一的时区。
	// 这对于跨时区部署和日志时间戳的一致性至关重要。
	if err := timezone.Init(cfg.Timezone); err != nil {
		return nil, nil, err
	}

	drv, err := openEntDriver(cfg)
	if err != nil {
		return nil, nil, err
	}
	applyDBPoolSettings(drv.DB(), cfg)

	// 确保数据库 schema 已准备就绪。
	// 以 backend/migrations/*.sql 为 schema 权威来源（SQLite 方言）。
	migrationCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := prepareSchema(migrationCtx, drv, cfg); err != nil {
		_ = drv.Close()
		return nil, nil, err
	}

	// 创建 Ent 客户端，绑定到已配置的数据库驱动。
	client := ent.NewClient(ent.Driver(drv))

	// 启动阶段：从配置或数据库中确保系统密钥可用。
	if err := ensureBootstrapSecrets(migrationCtx, client, cfg); err != nil {
		_ = client.Close()
		return nil, nil, err
	}
	// 在密钥补齐后执行完整配置校验，避免空 jwt.secret 导致服务运行时失败。
	if err := cfg.Validate(); err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("validate config after secret bootstrap: %w", err)
	}

	adminCreated, err := ensureConfiguredAdmin(migrationCtx, client, cfg)
	if err != nil {
		_ = client.Close()
		return nil, nil, err
	}
	if adminCreated {
		logger.LegacyPrintf("repository", "initial admin user created from config: %s", cfg.Default.AdminEmail)
	}

	// SIMPLE 模式：启动时补齐各平台默认分组。
	// - anthropic/openai/gemini: 确保存在 <platform>-default
	// - antigravity: 仅要求存在 >=2 个未软删除分组（用于 claude/gemini 混合调度场景）
	if cfg.RunMode == config.RunModeSimple {
		seedCtx, seedCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer seedCancel()
		if err := ensureSimpleModeDefaultGroups(seedCtx, client); err != nil {
			_ = client.Close()
			return nil, nil, err
		}
		if err := ensureSimpleModeAdminConcurrency(seedCtx, client); err != nil {
			_ = client.Close()
			return nil, nil, err
		}
	}

	return client, drv.DB(), nil
}

// openEntDriver opens the application's SQLite driver.
func openEntDriver(cfg *config.Config) (*entsql.Driver, error) {
	return openSQLiteDriver(cfg)
}

func openSQLiteDriver(cfg *config.Config) (*entsql.Driver, error) {
	path := cfg.Database.SQLitePath()
	// Ensure parent directory exists for paths like ./data/sub2api.db
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create sqlite data directory %q: %w", dir, err)
		}
	}

	// Register PG-shaped helpers (NOW/GREATEST/LEAST) before opening connections.
	registerSQLitePGCompatFunctions()

	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_time_format=sqlite", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// SQLite: keep a small pool; WAL allows concurrent readers with one writer.
	if cfg.Database.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	} else {
		db.SetMaxOpenConns(1)
	}
	logger.LegacyPrintf("repository", "using SQLite database at %s", path)
	return entsql.OpenDB(dialect.SQLite, db), nil
}

// prepareSchema applies SQL migrations (SQLite dialect under migrations/*.sql).
// SQLite uses the embedded migration files as the schema source.
func prepareSchema(ctx context.Context, drv *entsql.Driver, cfg *config.Config) error {
	return prepareSchemaWithFS(ctx, drv, cfg, migrations.FS)
}

func prepareSchemaWithFS(ctx context.Context, drv *entsql.Driver, cfg *config.Config, migrationsFS fs.FS) error {
	// Personal SQLite DBs may have been bootstrapped by Ent Schema.Create
	// without schema_migrations. Always patch critical usage/billing objects,
	// then apply migrations (duplicate column/index errors are ignored).
	if err := EnsureSQLiteAuxTables(ctx, drv.DB()); err != nil {
		return fmt.Errorf("sqlite aux tables: %w", err)
	}
	if err := applyMigrationsFS(ctx, drv.DB(), migrationsFS); err != nil {
		return err
	}
	// Re-run aux after migrations so newly created tables get indexes/stubs.
	if err := EnsureSQLiteAuxTables(ctx, drv.DB()); err != nil {
		return fmt.Errorf("sqlite aux tables (post-migrate): %w", err)
	}
	return nil
}
