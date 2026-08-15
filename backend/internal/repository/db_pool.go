// Package repository contains persistence infrastructure helpers.
//
// SQLite uses a single-writer model, so the pool is clamped to a small size
// to avoid lock contention. Idle/lifetime caps still apply so leaked
// connections cannot accumulate.
package repository

import (
	"database/sql"
	"log/slog"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

const (
	defaultConnMaxLifetime = 30 * time.Minute
	defaultConnMaxIdleTime = 5 * time.Minute
	maxConfiguredConnAge   = 24 * time.Hour
)

type dbPoolSettings struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

func clampDBPoolSettings(cfg *config.Config) dbPoolSettings {
	maxOpen := cfg.Database.MaxOpenConns
	maxIdle := cfg.Database.MaxIdleConns
	// SQLite has a single-writer model; a large pool only increases lock contention.
	if cfg.Database.IsSQLite() {
		if maxOpen <= 0 || maxOpen > 8 {
			maxOpen = 4
		}
		if maxIdle <= 0 || maxIdle > maxOpen {
			maxIdle = maxOpen
		}
	}
	return dbPoolSettings{
		MaxOpenConns:    maxOpen,
		MaxIdleConns:    maxIdle,
		ConnMaxLifetime: clampDBPoolDuration("database.conn_max_lifetime_minutes", cfg.Database.ConnMaxLifetimeMinutes, defaultConnMaxLifetime),
		ConnMaxIdleTime: clampDBPoolDuration("database.conn_max_idle_time_minutes", cfg.Database.ConnMaxIdleTimeMinutes, defaultConnMaxIdleTime),
	}
}

func clampDBPoolDuration(key string, minutes int, fallback time.Duration) time.Duration {
	if minutes <= 0 || minutes > int(maxConfiguredConnAge/time.Minute) {
		slog.Warn("database connection pool duration clamped",
			"key", key,
			"before", minutes,
			"after", int(fallback/time.Minute),
		)
		return fallback
	}

	return time.Duration(minutes) * time.Minute
}

func applyDBPoolSettings(db *sql.DB, cfg *config.Config) {
	settings := clampDBPoolSettings(cfg)
	db.SetMaxOpenConns(settings.MaxOpenConns)
	db.SetMaxIdleConns(settings.MaxIdleConns)
	db.SetConnMaxLifetime(settings.ConnMaxLifetime)
	db.SetConnMaxIdleTime(settings.ConnMaxIdleTime)

	slog.Info("database connection pool configured",
		slog.Group("effective",
			slog.Int("max_open", settings.MaxOpenConns),
			slog.Int("max_idle", settings.MaxIdleConns),
			slog.Duration("max_lifetime", settings.ConnMaxLifetime),
			slog.Duration("max_idle_time", settings.ConnMaxIdleTime),
		),
	)
}
