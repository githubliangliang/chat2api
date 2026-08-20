package repository

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestEnsureConfiguredAdminCreatesAdminFromConfigOnEmptyDatabase(t *testing.T) {
	client := newSecuritySecretTestClient(t)
	cfg := &config.Config{
		Default: config.DefaultConfig{
			AdminEmail:      "admin@example.com",
			AdminPassword:   "correct-horse-battery",
			UserBalance:     0,
			UserConcurrency: 7,
		},
	}

	created, err := ensureConfiguredAdmin(context.Background(), client, cfg)
	require.NoError(t, err)
	require.True(t, created)

	admin, err := client.User.Query().Only(context.Background())
	require.NoError(t, err)
	require.Equal(t, "admin@example.com", admin.Email)
	require.Equal(t, "admin", admin.Role)
	require.Equal(t, 7, admin.Concurrency)
	require.True(t, (&service.User{PasswordHash: admin.PasswordHash}).CheckPassword("correct-horse-battery"))
}

func TestEnsureConfiguredAdminDoesNotOverwriteExistingUsers(t *testing.T) {
	client := newSecuritySecretTestClient(t)
	_, err := client.User.Create().
		SetEmail("existing@example.com").
		SetPasswordHash("existing-hash").
		SetRole("user").
		SetStatus("active").
		Save(context.Background())
	require.NoError(t, err)

	created, err := ensureConfiguredAdmin(context.Background(), client, &config.Config{
		Default: config.DefaultConfig{
			AdminEmail:    "admin@example.com",
			AdminPassword: "correct-horse-battery",
		},
	})
	require.NoError(t, err)
	require.False(t, created)
	require.Len(t, client.User.Query().AllX(context.Background()), 1)

	created, err = ensureConfiguredAdmin(context.Background(), client, &config.Config{
		Default: config.DefaultConfig{AdminEmail: "legacy@example.com"},
	})
	require.NoError(t, err)
	require.False(t, created)
}

func TestEnsureConfiguredAdminRejectsIncompleteCredentialsOnEmptyDatabase(t *testing.T) {
	client := newSecuritySecretTestClient(t)

	_, err := ensureConfiguredAdmin(context.Background(), client, &config.Config{
		Default: config.DefaultConfig{AdminEmail: "admin@example.com"},
	})
	require.ErrorContains(t, err, "must be provided together")

	_, err = ensureConfiguredAdmin(context.Background(), client, &config.Config{
		Default: config.DefaultConfig{
			AdminEmail:    "admin@example.com",
			AdminPassword: "short",
		},
	})
	require.ErrorContains(t, err, "between 8 and 128")

	for _, password := range []string{"admin123", "CHANGE_ME_STRONG_PASSWORD"} {
		_, err = ensureConfiguredAdmin(context.Background(), client, &config.Config{
			Default: config.DefaultConfig{
				AdminEmail:    "admin@example.com",
				AdminPassword: password,
			},
		})
		require.ErrorContains(t, err, "must be changed from the example value")
	}
	count, countErr := client.User.Query().Count(context.Background())
	require.NoError(t, countErr)
	require.Zero(t, count)
}

func TestInitEntBootstrapsConfiguredAdmin(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("CONFIG_FILE", "")
	t.Setenv("DATA_DIR", t.TempDir())
	t.Setenv("JWT_SECRET", strings.Repeat("j", 32))

	cfg, err := config.LoadForBootstrap()
	require.NoError(t, err)
	cfg.Database.Path = filepath.Join(t.TempDir(), "sub2api.db")
	cfg.Timezone = "UTC"
	cfg.Default.AdminEmail = "configured@example.com"
	cfg.Default.AdminPassword = "correct-horse-battery"

	client, db, err := InitEnt(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	t.Cleanup(func() { _ = db.Close() })

	admin, err := client.User.Query().Only(context.Background())
	require.NoError(t, err)
	require.Equal(t, "configured@example.com", admin.Email)
	require.True(t, (&service.User{PasswordHash: admin.PasswordHash}).CheckPassword("correct-horse-battery"))
}
