package repository

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// SQLiteDumper implements service.DBDumper with VACUUM INTO / file replace.
type SQLiteDumper struct {
	cfg *config.DatabaseConfig
}

// NewSQLiteDumper creates a dumper for the configured SQLite file.
func NewSQLiteDumper(cfg *config.Config) service.DBDumper {
	return &SQLiteDumper{cfg: &cfg.Database}
}

// Dump writes a consistent SQLite snapshot via VACUUM INTO.
func (d *SQLiteDumper) Dump(ctx context.Context) (io.ReadCloser, error) {
	src := d.cfg.SQLitePath()
	if _, err := os.Stat(src); err != nil {
		return nil, fmt.Errorf("sqlite dump source: %w", err)
	}

	tmp, err := os.CreateTemp("", "sub2api-sqlite-dump-*.db")
	if err != nil {
		return nil, fmt.Errorf("create sqlite dump temp: %w", err)
	}
	path := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(path)
		return nil, fmt.Errorf("close sqlite dump temp: %w", err)
	}
	if err := os.Remove(path); err != nil {
		return nil, fmt.Errorf("reset sqlite dump temp: %w", err)
	}

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)", src))
	if err != nil {
		return nil, fmt.Errorf("open sqlite for dump: %w", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", path); err != nil {
		_ = os.Remove(path)
		return nil, fmt.Errorf("sqlite vacuum into: %w", err)
	}

	file, err := os.Open(path)
	if err != nil {
		_ = os.Remove(path)
		return nil, fmt.Errorf("open sqlite dump: %w", err)
	}
	return &removeOnClose{ReadCloser: file, path: path}, nil
}

// Restore replaces the live SQLite file with the incoming snapshot.
func (d *SQLiteDumper) Restore(_ context.Context, data io.Reader) error {
	dest := d.cfg.SQLitePath()
	if dest == "" {
		return fmt.Errorf("sqlite restore path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("create sqlite restore directory: %w", err)
	}

	tmp := dest + ".restore-tmp"
	file, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create sqlite restore temp: %w", err)
	}
	if _, err := io.Copy(file, data); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("write sqlite restore temp: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close sqlite restore temp: %w", err)
	}

	_ = os.Remove(dest + "-wal")
	_ = os.Remove(dest + "-shm")
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace sqlite database: %w", err)
	}
	return nil
}

type removeOnClose struct {
	io.ReadCloser
	path string
}

func (r *removeOnClose) Close() error {
	err := r.ReadCloser.Close()
	if removeErr := os.Remove(r.path); err == nil {
		err = removeErr
	}
	return err
}
