package repository

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSplitSQLStatementsIgnoresSemicolonsInComments(t *testing.T) {
	content := `
-- Provider/upstream details (optional; useful for trends & account health)
CREATE TABLE IF NOT EXISTS ops_error_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    -- note: optional; keep together
    name TEXT
);

-- 2) ops_metrics_daily (optional; for longer windows)
CREATE INDEX IF NOT EXISTS idx_x ON ops_error_logs (id);

INSERT INTO t(name) VALUES ('a;b');
INSERT INTO t(name) VALUES ("c;d");
/* block; comment */ SELECT 1;
`

	stmts := splitSQLStatements(content)
	if len(stmts) != 5 {
		t.Fatalf("expected 5 statements, got %d:\n%s", len(stmts), strings.Join(stmts, "\n---\n"))
	}
	if !strings.Contains(stmts[0], "CREATE TABLE IF NOT EXISTS ops_error_logs") {
		t.Fatalf("stmt0 should be CREATE TABLE, got: %q", stmts[0])
	}
	if !strings.Contains(stmts[0], "optional; useful") {
		t.Fatalf("stmt0 should keep comment semicolon, got: %q", stmts[0])
	}
	if !strings.Contains(stmts[1], "CREATE INDEX") {
		t.Fatalf("stmt1 should be CREATE INDEX, got: %q", stmts[1])
	}
	if !strings.Contains(stmts[2], "'a;b'") {
		t.Fatalf("stmt2 should keep string semicolon, got: %q", stmts[2])
	}
	if !strings.Contains(stmts[3], `"c;d"`) {
		t.Fatalf("stmt3 should keep quoted id semicolon, got: %q", stmts[3])
	}
	if !strings.Contains(stmts[4], "SELECT 1") {
		t.Fatalf("stmt4 should be SELECT 1, got: %q", stmts[4])
	}
}

func TestSplitSQLStatements033OpsMonitoringVnext(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// backend/internal/repository -> backend/migrations
	migrationPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations", "033_ops_monitoring_vnext.sql")
	content, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	stmts := splitSQLStatements(string(content))
	// Must not split CREATE TABLE ops_error_logs mid-body due to comment semicolons.
	for i, s := range stmts {
		trim := strings.TrimSpace(s)
		if strings.HasPrefix(trim, "useful for trends") {
			t.Fatalf("statement %d is a comment fragment split by semicolon: %q", i+1, trim)
		}
		if strings.HasPrefix(trim, "for longer windows") {
			t.Fatalf("statement %d is a comment fragment split by semicolon: %q", i+1, trim)
		}
	}
	var foundCreate bool
	for _, s := range stmts {
		if strings.Contains(s, "CREATE TABLE IF NOT EXISTS ops_error_logs") &&
			strings.Contains(s, "created_at DATETIME NOT NULL DEFAULT (datetime('now'))") {
			foundCreate = true
		}
	}
	if !foundCreate {
		t.Fatalf("expected intact CREATE TABLE ops_error_logs among %d statements", len(stmts))
	}
}
