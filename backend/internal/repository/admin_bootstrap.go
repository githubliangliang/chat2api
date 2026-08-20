package repository

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// ensureConfiguredAdmin creates the first administrator when a deployment
// supplied default.admin_email and default.admin_password in config.yaml.
// Existing databases are left untouched so a config edit cannot overwrite an
// existing account or silently reset its password.
func ensureConfiguredAdmin(ctx context.Context, client *ent.Client, cfg *config.Config) (bool, error) {
	if client == nil {
		return false, fmt.Errorf("nil ent client")
	}
	if cfg == nil {
		return false, fmt.Errorf("nil config")
	}

	users, err := client.User.Query().Count(mixins.SkipSoftDelete(ctx))
	if err != nil {
		return false, fmt.Errorf("count users for admin bootstrap: %w", err)
	}
	if users != 0 {
		return false, nil
	}

	email := strings.TrimSpace(cfg.Default.AdminEmail)
	password := cfg.Default.AdminPassword
	if email == "" && strings.TrimSpace(password) == "" {
		return false, nil
	}
	if email == "" || strings.TrimSpace(password) == "" {
		return false, fmt.Errorf("default.admin_email and default.admin_password must be provided together")
	}
	if _, err := mail.ParseAddress(email); err != nil || len(email) > 254 {
		return false, fmt.Errorf("invalid default.admin_email")
	}
	if len(password) < 8 || len(password) > 128 {
		return false, fmt.Errorf("default.admin_password must be between 8 and 128 characters")
	}
	switch password {
	case "admin123", "CHANGE_ME_STRONG_PASSWORD":
		return false, fmt.Errorf("default.admin_password must be changed from the example value")
	}

	admin := &service.User{}
	if err := admin.SetPassword(password); err != nil {
		return false, fmt.Errorf("hash default admin password: %w", err)
	}

	concurrency := cfg.Default.UserConcurrency
	if concurrency <= 0 {
		concurrency = 5
	}
	_, err = client.User.Create().
		SetEmail(email).
		SetPasswordHash(admin.PasswordHash).
		SetRole(service.RoleAdmin).
		SetBalance(cfg.Default.UserBalance).
		SetConcurrency(concurrency).
		SetStatus(service.StatusActive).
		SetSignupSource("email").
		Save(ctx)
	if err != nil {
		return false, fmt.Errorf("create configured admin: %w", err)
	}

	return true, nil
}
