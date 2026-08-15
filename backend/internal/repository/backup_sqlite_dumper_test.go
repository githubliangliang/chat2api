package repository

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestSQLiteDumperDumpAndRestore(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "app.db")
	db, err := sql.Open("sqlite", "file:"+src)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT); INSERT INTO notes(body) VALUES ('hello')`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	dumper := NewSQLiteDumper(&config.Config{Database: config.DatabaseConfig{Path: src}})
	reader, err := dumper.Dump(context.Background())
	require.NoError(t, err)
	snapshot, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.NotEmpty(t, snapshot)

	require.NoError(t, os.Remove(src))
	require.NoError(t, dumper.Restore(context.Background(), bytes.NewReader(snapshot)))

	db, err = sql.Open("sqlite", "file:"+src)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	var body string
	require.NoError(t, db.QueryRow(`SELECT body FROM notes`).Scan(&body))
	require.Equal(t, "hello", body)
}
