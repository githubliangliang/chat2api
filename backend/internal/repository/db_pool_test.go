package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"

	_ "github.com/lib/pq"
)

func TestClampDBPoolSettings(t *testing.T) {
	tests := []struct {
		name                string
		connMaxLifetime     int
		connMaxIdleTime     int
		wantMaxLifetime     time.Duration
		wantConnMaxIdleTime time.Duration
	}{
		{
			name:                "zero values fall back to safe defaults",
			connMaxLifetime:     0,
			connMaxIdleTime:     0,
			wantMaxLifetime:     30 * time.Minute,
			wantConnMaxIdleTime: 5 * time.Minute,
		},
		{
			name:                "negative values fall back to safe defaults",
			connMaxLifetime:     -1,
			connMaxIdleTime:     -5,
			wantMaxLifetime:     30 * time.Minute,
			wantConnMaxIdleTime: 5 * time.Minute,
		},
		{
			name:                "reasonable values pass through",
			connMaxLifetime:     15,
			connMaxIdleTime:     3,
			wantMaxLifetime:     15 * time.Minute,
			wantConnMaxIdleTime: 3 * time.Minute,
		},
		{
			name:                "values over twenty four hours fall back to safe defaults",
			connMaxLifetime:     24*60 + 1,
			connMaxIdleTime:     24*60 + 1,
			wantMaxLifetime:     30 * time.Minute,
			wantConnMaxIdleTime: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Database: config.DatabaseConfig{
					MaxOpenConns:           50,
					MaxIdleConns:           10,
					ConnMaxLifetimeMinutes: tt.connMaxLifetime,
					ConnMaxIdleTimeMinutes: tt.connMaxIdleTime,
				},
			}

			settings := clampDBPoolSettings(cfg)
			// SQLite-only fork: out-of-range 50/10 clamp to the single-writer pool cap.
			require.Equal(t, 4, settings.MaxOpenConns)
			require.Equal(t, 4, settings.MaxIdleConns)
			require.Equal(t, tt.wantMaxLifetime, settings.ConnMaxLifetime)
			require.Equal(t, tt.wantConnMaxIdleTime, settings.ConnMaxIdleTime)
		})
	}
}

func TestClampDBPoolSettings_SQLitePoolBounds(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			MaxOpenConns:           6,
			MaxIdleConns:           2,
			ConnMaxLifetimeMinutes: 15,
			ConnMaxIdleTimeMinutes: 3,
		},
	}

	settings := clampDBPoolSettings(cfg)
	require.Equal(t, 6, settings.MaxOpenConns, "values within the 1..8 window pass through")
	require.Equal(t, 2, settings.MaxIdleConns)

	cfg.Database.MaxIdleConns = 7
	settings = clampDBPoolSettings(cfg)
	require.Equal(t, 6, settings.MaxIdleConns, "idle above max_open collapses to max_open")
}

func TestApplyDBPoolSettings(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			MaxOpenConns:           40,
			MaxIdleConns:           8,
			ConnMaxLifetimeMinutes: 15,
			ConnMaxIdleTimeMinutes: 3,
		},
	}

	db, err := sql.Open("postgres", "host=127.0.0.1 port=5432 user=postgres sslmode=disable")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	applyDBPoolSettings(db, cfg)
	stats := db.Stats()
	// 40 exceeds the SQLite single-writer window, so the effective pool is the cap.
	require.Equal(t, 4, stats.MaxOpenConnections)
}
