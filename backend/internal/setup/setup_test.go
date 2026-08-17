package setup

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/repository"

	"gopkg.in/yaml.v3"
)

func TestDecideAdminBootstrap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		totalUsers int64
		adminUsers int64
		should     bool
		reason     string
	}{
		{
			name:       "empty database should create admin",
			totalUsers: 0,
			adminUsers: 0,
			should:     true,
			reason:     adminBootstrapReasonEmptyDatabase,
		},
		{
			name:       "admin exists should skip",
			totalUsers: 10,
			adminUsers: 1,
			should:     false,
			reason:     adminBootstrapReasonAdminExists,
		},
		{
			name:       "users exist without admin should skip",
			totalUsers: 5,
			adminUsers: 0,
			should:     false,
			reason:     adminBootstrapReasonUsersExistWithoutAdmin,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := decideAdminBootstrap(tc.totalUsers, tc.adminUsers)
			if got.shouldCreate != tc.should {
				t.Fatalf("shouldCreate=%v, want %v", got.shouldCreate, tc.should)
			}
			if got.reason != tc.reason {
				t.Fatalf("reason=%q, want %q", got.reason, tc.reason)
			}
		})
	}
}

func TestSetupDefaultAdminConcurrency(t *testing.T) {
	t.Run("simple mode admin uses higher concurrency", func(t *testing.T) {
		t.Setenv("RUN_MODE", "simple")
		if got := setupDefaultAdminConcurrency(); got != simpleModeAdminConcurrency {
			t.Fatalf("setupDefaultAdminConcurrency()=%d, want %d", got, simpleModeAdminConcurrency)
		}
	})

	t.Run("standard mode keeps existing default", func(t *testing.T) {
		t.Setenv("RUN_MODE", "standard")
		if got := setupDefaultAdminConcurrency(); got != defaultUserConcurrency {
			t.Fatalf("setupDefaultAdminConcurrency()=%d, want %d", got, defaultUserConcurrency)
		}
	})
}

func TestNeedsSetupSkipsWhenSkipSetupIsEnabled(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "true", value: "true"},
		{name: "one", value: "1"},
		{name: "yes", value: "yes"},
		{name: "trimmed mixed case true", value: "  TrUe  "},
		{name: "trimmed mixed case yes", value: "  YeS  "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATA_DIR", t.TempDir())
			t.Setenv("SKIP_SETUP", tc.value)

			if NeedsSetup() {
				t.Fatalf("NeedsSetup() = true, want false when SKIP_SETUP is enabled")
			}
		})
	}
}

func TestNeedsSetupFallsBackToFileDetectionWhenSkipSetupIsDisabled(t *testing.T) {
	tests := []struct {
		name         string
		skipSetupSet bool
		skipSetup    string
		markerFile   string
		want         bool
	}{
		{
			name: "unset without installation files",
			want: true,
		},
		{
			name:         "false without installation files",
			skipSetupSet: true,
			skipSetup:    " false ",
			want:         true,
		},
		{
			name:         "invalid value without installation files",
			skipSetupSet: true,
			skipSetup:    "enabled",
			want:         true,
		},
		{
			name:         "config file exists",
			skipSetupSet: true,
			skipSetup:    "false",
			markerFile:   ConfigFileName,
			want:         false,
		},
		{
			name:         "install lock file exists",
			skipSetupSet: true,
			skipSetup:    "invalid",
			markerFile:   InstallLockFile,
			want:         false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dataDir := t.TempDir()
			t.Setenv("DATA_DIR", dataDir)
			if tc.skipSetupSet {
				t.Setenv("SKIP_SETUP", tc.skipSetup)
			} else {
				originalValue, wasSet := os.LookupEnv("SKIP_SETUP")
				if err := os.Unsetenv("SKIP_SETUP"); err != nil {
					t.Fatalf("Unsetenv(SKIP_SETUP) error = %v", err)
				}
				t.Cleanup(func() {
					if wasSet {
						_ = os.Setenv("SKIP_SETUP", originalValue)
						return
					}
					_ = os.Unsetenv("SKIP_SETUP")
				})
			}

			if tc.markerFile != "" {
				if err := os.WriteFile(filepath.Join(dataDir, tc.markerFile), nil, 0o600); err != nil {
					t.Fatalf("WriteFile(%s) error = %v", tc.markerFile, err)
				}
			}

			if got := NeedsSetup(); got != tc.want {
				t.Fatalf("NeedsSetup() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSetupMigrationTimeout(t *testing.T) {
	t.Run("uses default timeout when unset", func(t *testing.T) {
		cfg := &SetupConfig{}
		if got := cfg.migrationTimeout(); got != 60*time.Second {
			t.Fatalf("migrationTimeout()=%s, want 60s", got)
		}
	})

	t.Run("uses configured timeout", func(t *testing.T) {
		cfg := &SetupConfig{MigrationTimeoutSeconds: 300}
		if got := cfg.migrationTimeout(); got != 300*time.Second {
			t.Fatalf("migrationTimeout()=%s, want 300s", got)
		}
	})
}

func TestInitializeSQLiteSchemaSupportsStartupMigrations(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "sub2api.db")
	cfg := &SetupConfig{
		Database: DatabaseConfig{
			Driver: "sqlite",
			Path:   databasePath,
		},
	}

	if err := initializeSQLiteSchema(cfg); err != nil {
		t.Fatalf("initializeSQLiteSchema() error = %v", err)
	}

	db, err := sql.Open("sqlite", buildSQLiteDSN(&cfg.Database))
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)

	if err := repository.ApplyMigrations(context.Background(), db); err != nil {
		t.Fatalf("startup ApplyMigrations() error = %v", err)
	}

	var supportedModelScopes string
	var messagesDispatchModelConfig string
	var modelsListConfig string
	var reasoningEffortMappings string
	err = db.QueryRow(`
		SELECT supported_model_scopes,
		       messages_dispatch_model_config,
		       models_list_config,
		       reasoning_effort_mappings
		FROM groups
		WHERE name = 'default'
	`).Scan(
		&supportedModelScopes,
		&messagesDispatchModelConfig,
		&modelsListConfig,
		&reasoningEffortMappings,
	)
	if err != nil {
		t.Fatalf("query default group error = %v", err)
	}

	var scopes []string
	if err := json.Unmarshal([]byte(supportedModelScopes), &scopes); err != nil {
		t.Fatalf("supported_model_scopes is invalid JSON: %v", err)
	}
	wantScopes := []string{"claude", "gemini_text", "gemini_image"}
	if !reflect.DeepEqual(scopes, wantScopes) {
		t.Fatalf("supported_model_scopes = %#v, want %#v", scopes, wantScopes)
	}

	for name, value := range map[string]string{
		"messages_dispatch_model_config": messagesDispatchModelConfig,
		"models_list_config":             modelsListConfig,
	} {
		var object map[string]any
		if err := json.Unmarshal([]byte(value), &object); err != nil {
			t.Fatalf("%s is invalid JSON: %v", name, err)
		}
		if len(object) != 0 {
			t.Fatalf("%s = %#v, want empty object", name, object)
		}
	}

	var mappings []any
	if err := json.Unmarshal([]byte(reasoningEffortMappings), &mappings); err != nil {
		t.Fatalf("reasoning_effort_mappings is invalid JSON: %v", err)
	}
	if len(mappings) != 0 {
		t.Fatalf("reasoning_effort_mappings = %#v, want empty array", mappings)
	}
}

func TestWriteConfigFileKeepsDefaultUserConcurrency(t *testing.T) {
	t.Setenv("RUN_MODE", "simple")
	t.Setenv("DATA_DIR", t.TempDir())

	if err := writeConfigFile(&SetupConfig{}); err != nil {
		t.Fatalf("writeConfigFile() error = %v", err)
	}

	data, err := os.ReadFile(GetConfigFilePath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(data), "user_concurrency: 5") {
		t.Fatalf("config missing default user concurrency, got:\n%s", string(data))
	}
}

func TestWriteConfigFileIncludesRedisUsername(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())

	if err := writeConfigFile(&SetupConfig{
		Redis: RedisConfig{
			Host:     "redis",
			Port:     6379,
			Username: "app-user",
		},
	}); err != nil {
		t.Fatalf("writeConfigFile() error = %v", err)
	}

	data, err := os.ReadFile(GetConfigFilePath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !strings.Contains(string(data), "username: app-user") {
		t.Fatalf("config missing Redis username, got:\n%s", string(data))
	}
}

func TestWriteConfigFilePersistsRedisEnabledFlag(t *testing.T) {
	tests := []struct {
		name  string
		redis RedisConfig
		want  RedisConfig
	}{
		{
			name:  "external redis stays enabled",
			redis: RedisConfig{Host: "redis", Port: 6379, Password: "secret"},
			want:  RedisConfig{Enabled: boolPtr(true), Host: "redis", Port: 6379, Password: "secret"},
		},
		{
			name:  "disabled redis drops connection details",
			redis: RedisConfig{Enabled: boolPtr(false), Host: "localhost", Port: 6379, Password: "secret", DB: 3, EnableTLS: true},
			want:  RedisConfig{Enabled: boolPtr(false), Host: "", Port: 6379},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATA_DIR", t.TempDir())

			if err := writeConfigFile(&SetupConfig{Redis: tc.redis}); err != nil {
				t.Fatalf("writeConfigFile() error = %v", err)
			}

			data, err := os.ReadFile(GetConfigFilePath())
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}

			var parsed struct {
				Redis RedisConfig `yaml:"redis"`
			}
			if err := yaml.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("Unmarshal() error = %v, config:\n%s", err, string(data))
			}

			// The runtime default is redis.enabled=true, so the flag must always be explicit.
			if parsed.Redis.Enabled == nil {
				t.Fatalf("redis.enabled not written, got:\n%s", string(data))
			}
			if got, want := *parsed.Redis.Enabled, *tc.want.Enabled; got != want {
				t.Fatalf("redis.enabled = %v, want %v", got, want)
			}
			if parsed.Redis.Host != tc.want.Host {
				t.Fatalf("redis.host = %q, want %q", parsed.Redis.Host, tc.want.Host)
			}
			if parsed.Redis.Port != tc.want.Port {
				t.Fatalf("redis.port = %d, want %d", parsed.Redis.Port, tc.want.Port)
			}
			if parsed.Redis.Password != tc.want.Password {
				t.Fatalf("redis.password = %q, want %q", parsed.Redis.Password, tc.want.Password)
			}
			if parsed.Redis.DB != tc.want.DB {
				t.Fatalf("redis.db = %d, want %d", parsed.Redis.DB, tc.want.DB)
			}
			if parsed.Redis.EnableTLS != tc.want.EnableTLS {
				t.Fatalf("redis.enable_tls = %v, want %v", parsed.Redis.EnableTLS, tc.want.EnableTLS)
			}
		})
	}
}

func TestDatabaseConfigUsesSQLiteAsOnlyTarget(t *testing.T) {
	cfg := &DatabaseConfig{
		Driver: "postgres",
		Path:   "/tmp/sub2api.db",
	}

	if !cfg.IsSQLite() {
		t.Fatal("legacy driver value must still select SQLite")
	}
	if dsn := buildSQLiteDSN(cfg); !strings.Contains(dsn, "file:/tmp/sub2api.db") {
		t.Fatalf("SQLite DSN = %q, want configured database path", dsn)
	}
}
