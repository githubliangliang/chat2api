package setup

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

// Config paths
const (
	ConfigFileName             = "config.yaml"
	InstallLockFile            = ".installed"
	defaultUserConcurrency     = 5
	simpleModeAdminConcurrency = 30
	defaultMigrationTimeout    = 60 * time.Second
)

func setupDefaultAdminConcurrency() int {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("RUN_MODE")), config.RunModeSimple) {
		return simpleModeAdminConcurrency
	}
	return defaultUserConcurrency
}

// GetDataDir returns the data directory for storing config and lock files.
// Priority: DATA_DIR env > /app/data (if exists and writable) > current directory
func GetDataDir() string {
	// Check DATA_DIR environment variable first
	if dir := os.Getenv("DATA_DIR"); dir != "" {
		return dir
	}

	// Check if /app/data exists and is writable (Docker environment)
	dockerDataDir := "/app/data"
	if info, err := os.Stat(dockerDataDir); err == nil && info.IsDir() {
		// Try to check if writable by creating a temp file
		testFile := dockerDataDir + "/.write_test"
		if f, err := os.Create(testFile); err == nil {
			_ = f.Close()
			_ = os.Remove(testFile)
			return dockerDataDir
		}
	}

	// Default to current directory
	return "."
}

// GetConfigFilePath returns the full path to config.yaml
func GetConfigFilePath() string {
	return GetDataDir() + "/" + ConfigFileName
}

// GetInstallLockPath returns the full path to .installed lock file
func GetInstallLockPath() string {
	return GetDataDir() + "/" + InstallLockFile
}

// SetupConfig holds the setup configuration
type SetupConfig struct {
	Database                DatabaseConfig `json:"database" yaml:"database"`
	Redis                   RedisConfig    `json:"redis" yaml:"redis"`
	Admin                   AdminConfig    `json:"admin" yaml:"-"` // Not stored in config file
	Server                  ServerConfig   `json:"server" yaml:"server"`
	JWT                     JWTConfig      `json:"jwt" yaml:"jwt"`
	Timezone                string         `json:"timezone" yaml:"timezone"` // e.g. "Asia/Shanghai", "UTC"
	MigrationTimeoutSeconds int            `json:"migration_timeout_seconds" yaml:"migration_timeout_seconds,omitempty"`
}

type DatabaseConfig struct {
	// Driver is retained for config compatibility; SQLite is the only target.
	Driver string `json:"driver" yaml:"driver"`
	// Path: SQLite database file path.
	Path string `json:"path" yaml:"path"`
	// Synchronous mirrors config.DatabaseConfig.Synchronous so the wizard opens
	// the file with the same durability mode the server will use. Empty means
	// NORMAL, matching the runtime default.
	Synchronous string `json:"synchronous,omitempty" yaml:"synchronous,omitempty"`
	// Host/Port/User/Password/DBName/SSLMode are ignored leftovers kept so
	// older setup JSON / YAML still unmarshals.
	Host     string `json:"host,omitempty" yaml:"host,omitempty"`
	Port     int    `json:"port,omitempty" yaml:"port,omitempty"`
	User     string `json:"user,omitempty" yaml:"user,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	DBName   string `json:"dbname,omitempty" yaml:"dbname,omitempty"`
	SSLMode  string `json:"sslmode,omitempty" yaml:"sslmode,omitempty"`
}

// IsSQLite reports whether setup targets SQLite.
func (d *DatabaseConfig) IsSQLite() bool {
	return true
}

// SQLitePath returns the sqlite file path with default.
func (d *DatabaseConfig) SQLitePath() string {
	if strings.TrimSpace(d.Path) != "" {
		return strings.TrimSpace(d.Path)
	}
	return "./data/sub2api.db"
}

type RedisConfig struct {
	// Enabled: when false/nil-and-empty-host, Redis is optional (embedded at runtime).
	// Pointer preserves "unset" vs explicit false for YAML/JSON.
	Enabled   *bool  `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Host      string `json:"host" yaml:"host"`
	Port      int    `json:"port" yaml:"port"`
	Username  string `json:"username" yaml:"username"`
	Password  string `json:"password" yaml:"password"`
	DB        int    `json:"db" yaml:"db"`
	EnableTLS bool   `json:"enable_tls" yaml:"enable_tls"`
}

// IsEnabled reports whether external Redis is required during setup.
func (r *RedisConfig) IsEnabled() bool {
	if r.Enabled != nil {
		return *r.Enabled
	}
	// Default: enabled when host is non-empty (backward compatible).
	return strings.TrimSpace(r.Host) != ""
}

type AdminConfig struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type ServerConfig struct {
	Host string `json:"host" yaml:"host"`
	Port int    `json:"port" yaml:"port"`
	Mode string `json:"mode" yaml:"mode"`
}

type JWTConfig struct {
	Secret     string `json:"secret" yaml:"secret"`
	ExpireHour int    `json:"expire_hour" yaml:"expire_hour"`
}

const (
	adminBootstrapReasonEmptyDatabase          = "empty_database"
	adminBootstrapReasonAdminExists            = "admin_exists"
	adminBootstrapReasonUsersExistWithoutAdmin = "users_exist_without_admin"
)

type adminBootstrapDecision struct {
	shouldCreate bool
	reason       string
}

func decideAdminBootstrap(totalUsers, adminUsers int64) adminBootstrapDecision {
	if adminUsers > 0 {
		return adminBootstrapDecision{
			shouldCreate: false,
			reason:       adminBootstrapReasonAdminExists,
		}
	}
	if totalUsers > 0 {
		return adminBootstrapDecision{
			shouldCreate: false,
			reason:       adminBootstrapReasonUsersExistWithoutAdmin,
		}
	}
	return adminBootstrapDecision{
		shouldCreate: true,
		reason:       adminBootstrapReasonEmptyDatabase,
	}
}

func skipSetupEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SKIP_SETUP"))) {
	case "true", "1", "yes":
		return true
	default:
		return false
	}
}

// NeedsSetup checks if the system needs initial setup
// Uses multiple checks to prevent attackers from forcing re-setup by deleting config
func NeedsSetup() bool {
	if skipSetupEnabled() {
		logger.L().Debug("setup.needs_setup_bypassed", zap.String("reason", "skip_setup_enabled"))
		return false
	}

	// Check 1: Config file must not exist
	if _, err := os.Stat(GetConfigFilePath()); !os.IsNotExist(err) {
		return false // Config exists, no setup needed
	}

	// Check 2: Installation lock file (harder to bypass)
	if _, err := os.Stat(GetInstallLockPath()); !os.IsNotExist(err) {
		return false // Lock file exists, already installed
	}

	return true
}

func buildSQLiteDSN(cfg *DatabaseConfig) string {
	// Share the runtime builder so the wizard writes with the same pragmas the
	// server will later read with; a divergence here is invisible until the
	// first mismatched DATETIME scan or an unexpected durability change.
	return config.BuildSQLiteDSN(cfg.SQLitePath(), cfg.Synchronous)
}

// TestDatabaseConnection tests the database connection and creates database if not exists
func TestDatabaseConnection(cfg *DatabaseConfig) error {
	return testSQLiteConnection(cfg)
}

func testSQLiteConnection(cfg *DatabaseConfig) error {
	path := cfg.SQLitePath()
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create sqlite directory: %w", err)
		}
	}
	db, err := sql.Open("sqlite", buildSQLiteDSN(cfg))
	if err != nil {
		return fmt.Errorf("failed to open SQLite: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.LegacyPrintf("setup", "failed to close sqlite connection: %v", err)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("sqlite ping failed: %w", err)
	}
	return nil
}

// TestRedisConnection tests the Redis connection.
// When Redis is disabled, the check is skipped (embedded Redis is used at runtime).
func TestRedisConnection(cfg *RedisConfig) error {
	if cfg == nil || !cfg.IsEnabled() {
		logger.LegacyPrintf("setup", "%s", "Redis disabled; skipping connection test (embedded Redis will be used at runtime)")
		return nil
	}

	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Username: cfg.Username,
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	if cfg.EnableTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: cfg.Host,
		}
	}

	rdb := redis.NewClient(opts)
	defer func() {
		if err := rdb.Close(); err != nil {
			logger.LegacyPrintf("setup", "failed to close redis client: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}

// Install performs the installation with the given configuration
func Install(cfg *SetupConfig) error {
	// Security check: prevent re-installation if already installed
	if !NeedsSetup() {
		return fmt.Errorf("system is already installed, re-installation is not allowed")
	}

	// Generate JWT secret if not provided
	if cfg.JWT.Secret == "" {
		secret, err := generateSecret(32)
		if err != nil {
			return fmt.Errorf("failed to generate jwt secret: %w", err)
		}
		cfg.JWT.Secret = secret
		logger.LegacyPrintf("setup", "%s", "Warning: JWT secret auto-generated. Consider setting a fixed secret for production.")
	}

	// Test connections
	if err := TestDatabaseConnection(&cfg.Database); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	if err := TestRedisConnection(&cfg.Redis); err != nil {
		return fmt.Errorf("redis connection failed: %w", err)
	}

	// Initialize database
	if err := initializeDatabase(cfg); err != nil {
		return fmt.Errorf("database initialization failed: %w", err)
	}

	// Create admin user (only when database is empty and no admin exists).
	if _, _, err := createAdminUser(cfg); err != nil {
		return fmt.Errorf("admin user creation failed: %w", err)
	}

	// Write config file
	if err := writeConfigFile(cfg); err != nil {
		return fmt.Errorf("config file creation failed: %w", err)
	}

	// Create installation lock file to prevent re-setup attacks
	if err := createInstallLock(); err != nil {
		return fmt.Errorf("failed to create install lock: %w", err)
	}

	return nil
}

// createInstallLock creates a lock file to prevent re-installation attacks
func createInstallLock() error {
	content := fmt.Sprintf("installed_at=%s\n", time.Now().UTC().Format(time.RFC3339))
	return os.WriteFile(GetInstallLockPath(), []byte(content), 0400) // Read-only for owner
}

func initializeDatabase(cfg *SetupConfig) error {
	return initializeSQLiteSchema(cfg)
}

// initializeSQLiteSchema creates tables through the same migrations used at startup.
func initializeSQLiteSchema(cfg *SetupConfig) error {
	path := cfg.Database.SQLitePath()
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create sqlite directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", buildSQLiteDSN(&cfg.Database))
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	defer func() {
		if err := db.Close(); err != nil {
			logger.LegacyPrintf("setup", "failed to close sqlite database: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.migrationTimeout())
	defer cancel()
	if err := repository.ApplyMigrations(ctx, db); err != nil {
		return fmt.Errorf("sqlite migrations: %w", err)
	}
	if err := repository.EnsureSQLiteAuxTables(ctx, db); err != nil {
		return fmt.Errorf("sqlite aux tables: %w", err)
	}
	return nil
}

func boolPtr(v bool) *bool { return &v }

func (cfg *SetupConfig) migrationTimeout() time.Duration {
	if cfg != nil && cfg.MigrationTimeoutSeconds > 0 {
		return time.Duration(cfg.MigrationTimeoutSeconds) * time.Second
	}
	return defaultMigrationTimeout
}

func createAdminUser(cfg *SetupConfig) (bool, string, error) {
	db, err := sql.Open("sqlite", buildSQLiteDSN(&cfg.Database))
	if err != nil {
		return false, "", err
	}

	defer func() {
		if err := db.Close(); err != nil {
			logger.LegacyPrintf("setup", "failed to close database connection: %v", err)
		}
	}()

	// 使用超时上下文避免安装流程因数据库异常而长时间阻塞。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var totalUsers int64
	if err := db.QueryRowContext(ctx, "SELECT COUNT(1) FROM users").Scan(&totalUsers); err != nil {
		return false, "", err
	}
	var adminUsers int64
	if err := db.QueryRowContext(ctx, "SELECT COUNT(1) FROM users WHERE role = $1", service.RoleAdmin).Scan(&adminUsers); err != nil {
		return false, "", err
	}
	decision := decideAdminBootstrap(totalUsers, adminUsers)
	if !decision.shouldCreate {
		return false, decision.reason, nil
	}

	if strings.TrimSpace(cfg.Admin.Password) == "" {
		password, genErr := generateSecret(16)
		if genErr != nil {
			return false, "", fmt.Errorf("failed to generate admin password: %w", genErr)
		}
		cfg.Admin.Password = password
		fmt.Printf("Generated admin password (one-time): %s\n", cfg.Admin.Password)
		fmt.Println("IMPORTANT: Save this password! It will not be shown again.")
	}

	admin := &service.User{
		Email:       cfg.Admin.Email,
		Role:        service.RoleAdmin,
		Status:      service.StatusActive,
		Balance:     0,
		Concurrency: setupDefaultAdminConcurrency(),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := admin.SetPassword(cfg.Admin.Password); err != nil {
		return false, "", err
	}

	// Use RFC3339 text for SQLite datetime columns so Ent can scan them back.
	createdAt := admin.CreatedAt.UTC().Format(time.RFC3339Nano)
	updatedAt := admin.UpdatedAt.UTC().Format(time.RFC3339Nano)
	_, err = db.ExecContext(
		ctx,
		`INSERT INTO users (email, password_hash, role, balance, concurrency, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		admin.Email,
		admin.PasswordHash,
		admin.Role,
		admin.Balance,
		admin.Concurrency,
		admin.Status,
		createdAt,
		updatedAt,
	)
	if err != nil {
		return false, "", err
	}
	return true, decision.reason, nil
}

// normalizeRedisForConfigFile returns the Redis block to persist in config.yaml.
//
// The runtime default is redis.enabled=true (viper), so an omitted flag would
// silently re-enable external Redis after an install that opted out. Always
// write the flag explicitly, and drop connection details when disabled so the
// embedded in-process Redis is used at runtime.
func normalizeRedisForConfigFile(cfg RedisConfig) RedisConfig {
	enabled := cfg.IsEnabled()
	cfg.Enabled = boolPtr(enabled)
	if !enabled {
		cfg.Host = ""
		cfg.Username = ""
		cfg.Password = ""
		cfg.DB = 0
		cfg.EnableTLS = false
		if cfg.Port == 0 {
			cfg.Port = 6379
		}
	}
	return cfg
}

func writeConfigFile(cfg *SetupConfig) error {
	// Ensure timezone has a default value
	tz := cfg.Timezone
	if tz == "" {
		tz = "Asia/Shanghai"
	}

	redisCfg := normalizeRedisForConfigFile(cfg.Redis)

	// Prepare config for YAML (exclude sensitive data and admin config)
	yamlConfig := struct {
		Server   ServerConfig   `yaml:"server"`
		Database DatabaseConfig `yaml:"database"`
		Redis    RedisConfig    `yaml:"redis"`
		JWT      struct {
			Secret     string `yaml:"secret"`
			ExpireHour int    `yaml:"expire_hour"`
		} `yaml:"jwt"`
		Default struct {
			UserConcurrency int     `yaml:"user_concurrency"`
			UserBalance     float64 `yaml:"user_balance"`
			APIKeyPrefix    string  `yaml:"api_key_prefix"`
			RateMultiplier  float64 `yaml:"rate_multiplier"`
		} `yaml:"default"`
		RateLimit struct {
			RequestsPerMinute int `yaml:"requests_per_minute"`
			BurstSize         int `yaml:"burst_size"`
		} `yaml:"rate_limit"`
		Timezone string `yaml:"timezone"`
	}{
		Server:   cfg.Server,
		Database: cfg.Database,
		Redis:    redisCfg,
		JWT: struct {
			Secret     string `yaml:"secret"`
			ExpireHour int    `yaml:"expire_hour"`
		}{
			Secret:     cfg.JWT.Secret,
			ExpireHour: cfg.JWT.ExpireHour,
		},
		Default: struct {
			UserConcurrency int     `yaml:"user_concurrency"`
			UserBalance     float64 `yaml:"user_balance"`
			APIKeyPrefix    string  `yaml:"api_key_prefix"`
			RateMultiplier  float64 `yaml:"rate_multiplier"`
		}{
			UserConcurrency: defaultUserConcurrency,
			UserBalance:     0,
			APIKeyPrefix:    "sk-",
			RateMultiplier:  1.0,
		},
		RateLimit: struct {
			RequestsPerMinute int `yaml:"requests_per_minute"`
			BurstSize         int `yaml:"burst_size"`
		}{
			RequestsPerMinute: 60,
			BurstSize:         10,
		},
		Timezone: tz,
	}

	data, err := yaml.Marshal(&yamlConfig)
	if err != nil {
		return err
	}

	return os.WriteFile(GetConfigFilePath(), data, 0600)
}

func generateSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// =============================================================================
// Auto Setup for Docker Deployment
// =============================================================================

// AutoSetupEnabled checks if auto setup is enabled via environment variable
func AutoSetupEnabled() bool {
	val := os.Getenv("AUTO_SETUP")
	return val == "true" || val == "1" || val == "yes"
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// getEnvIntOrDefault gets environment variable as int or returns default value
func getEnvIntOrDefault(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultValue
}

// AutoSetupFromEnv performs automatic setup using environment variables
// This is designed for Docker deployment where all config is passed via env vars
func AutoSetupFromEnv() error {
	logger.LegacyPrintf("setup", "%s", "Auto setup enabled, configuring from environment variables...")
	logger.LegacyPrintf("setup", "Data directory: %s", GetDataDir())

	// Get timezone from TZ or TIMEZONE env var (TZ is standard for Docker)
	tz := getEnvOrDefault("TZ", "")
	if tz == "" {
		tz = getEnvOrDefault("TIMEZONE", "Asia/Shanghai")
	}

	// Build config from environment variables
	redisEnabledEnv := strings.ToLower(strings.TrimSpace(getEnvOrDefault("REDIS_ENABLED", "true")))
	redisEnabled := redisEnabledEnv != "false" && redisEnabledEnv != "0" && redisEnabledEnv != "no"
	dbDriver := "sqlite"

	cfg := &SetupConfig{
		Database: DatabaseConfig{
			Driver: dbDriver,
			Path:   getEnvOrDefault("DATABASE_PATH", "./data/sub2api.db"),
		},
		Redis: RedisConfig{
			Enabled:   boolPtr(redisEnabled),
			Host:      getEnvOrDefault("REDIS_HOST", "localhost"),
			Port:      getEnvIntOrDefault("REDIS_PORT", 6379),
			Username:  getEnvOrDefault("REDIS_USERNAME", ""),
			Password:  getEnvOrDefault("REDIS_PASSWORD", ""),
			DB:        getEnvIntOrDefault("REDIS_DB", 0),
			EnableTLS: getEnvOrDefault("REDIS_ENABLE_TLS", "false") == "true",
		},
		Admin: AdminConfig{
			Email:    getEnvOrDefault("ADMIN_EMAIL", "admin@sub2api.local"),
			Password: getEnvOrDefault("ADMIN_PASSWORD", ""),
		},
		Server: ServerConfig{
			Host: getEnvOrDefault("SERVER_HOST", "0.0.0.0"),
			Port: getEnvIntOrDefault("SERVER_PORT", 8080),
			Mode: getEnvOrDefault("SERVER_MODE", "release"),
		},
		JWT: JWTConfig{
			Secret:     getEnvOrDefault("JWT_SECRET", ""),
			ExpireHour: getEnvIntOrDefault("JWT_EXPIRE_HOUR", 24),
		},
		Timezone:                tz,
		MigrationTimeoutSeconds: getEnvIntOrDefault("SETUP_MIGRATION_TIMEOUT_SECONDS", 0),
	}

	// Generate JWT secret if not provided
	if cfg.JWT.Secret == "" {
		secret, err := generateSecret(32)
		if err != nil {
			return fmt.Errorf("failed to generate jwt secret: %w", err)
		}
		cfg.JWT.Secret = secret
		logger.LegacyPrintf("setup", "%s", "Warning: JWT secret auto-generated. Consider setting a fixed secret for production.")
	}

	// Test database connection
	logger.LegacyPrintf("setup", "%s", "Testing database connection...")
	if err := TestDatabaseConnection(&cfg.Database); err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	logger.LegacyPrintf("setup", "%s", "Database connection successful")

	// Test Redis connection
	logger.LegacyPrintf("setup", "%s", "Testing Redis connection...")
	if err := TestRedisConnection(&cfg.Redis); err != nil {
		return fmt.Errorf("redis connection failed: %w", err)
	}
	logger.LegacyPrintf("setup", "%s", "Redis connection successful")

	// Initialize database
	logger.LegacyPrintf("setup", "%s", "Initializing database...")
	if err := initializeDatabase(cfg); err != nil {
		return fmt.Errorf("database initialization failed: %w", err)
	}
	logger.LegacyPrintf("setup", "%s", "Database initialized successfully")

	// Create admin user
	logger.LegacyPrintf("setup", "%s", "Creating admin user...")
	created, reason, err := createAdminUser(cfg)
	if err != nil {
		return fmt.Errorf("admin user creation failed: %w", err)
	}
	if created {
		logger.LegacyPrintf("setup", "Admin user created: %s", cfg.Admin.Email)
	} else {
		switch reason {
		case adminBootstrapReasonAdminExists:
			logger.LegacyPrintf("setup", "%s", "Admin user already exists, skipping admin bootstrap")
		case adminBootstrapReasonUsersExistWithoutAdmin:
			logger.LegacyPrintf("setup", "%s", "Database already has user data; skipping auto admin bootstrap to avoid password overwrite")
		default:
			logger.LegacyPrintf("setup", "%s", "Admin bootstrap skipped")
		}
	}

	// Write config file
	logger.LegacyPrintf("setup", "%s", "Writing configuration file...")
	if err := writeConfigFile(cfg); err != nil {
		return fmt.Errorf("config file creation failed: %w", err)
	}
	logger.LegacyPrintf("setup", "%s", "Configuration file created")

	// Create installation lock file
	if err := createInstallLock(); err != nil {
		return fmt.Errorf("failed to create install lock: %w", err)
	}
	logger.LegacyPrintf("setup", "%s", "Installation lock created")

	logger.LegacyPrintf("setup", "%s", "Auto setup completed successfully!")
	return nil
}
