package repository

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"

	_ "modernc.org/sqlite"
)

// The DSN is only useful if the pragma actually lands on the connection, so
// assert against a real handle rather than the DSN string.
func TestBuildSQLiteDSN_SynchronousAppliedToConnection(t *testing.T) {
	cases := []struct {
		name       string
		configured string
		wantPragma string
	}{
		{"default is normal", "", "1"},
		{"explicit normal", "normal", "1"},
		{"explicit full", "full", "2"},
		{"case insensitive", "FULL", "2"},
		{"off", "off", "0"},
		{"unrecognized falls back to normal", "sometimes", "1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "probe.db")
			db, err := sql.Open("sqlite", config.BuildSQLiteDSN(path, tc.configured))
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			defer func() { _ = db.Close() }()

			var got string
			if err := db.QueryRow("PRAGMA synchronous").Scan(&got); err != nil {
				t.Fatalf("read pragma: %v", err)
			}
			if got != tc.wantPragma {
				t.Fatalf("PRAGMA synchronous = %s, want %s", got, tc.wantPragma)
			}

			// The pragmas the fork already depended on must survive the refactor:
			// WAL for concurrent readers, and _time_format so DATETIME scans back.
			var journal string
			if err := db.QueryRow("PRAGMA journal_mode").Scan(&journal); err != nil {
				t.Fatalf("read journal_mode: %v", err)
			}
			if journal != "wal" {
				t.Fatalf("journal_mode = %s, want wal", journal)
			}
			var fk string
			if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
				t.Fatalf("read foreign_keys: %v", err)
			}
			if fk != "1" {
				t.Fatalf("foreign_keys = %s, want 1", fk)
			}
		})
	}
}

// The runtime driver, migrations and the setup wizard must not drift apart.
func TestSQLiteDSN_RuntimeMatchesConfiguredBuilder(t *testing.T) {
	cfg := &config.Config{}
	cfg.Database.Path = "/tmp/example.db"
	cfg.Database.Synchronous = "full"

	want := config.BuildSQLiteDSN("/tmp/example.db", "full")
	if got := cfg.Database.DSN(); got != want {
		t.Fatalf("DSN() = %q, want %q", got, want)
	}
}
