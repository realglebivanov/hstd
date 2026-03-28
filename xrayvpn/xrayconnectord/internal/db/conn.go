package db

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/realglebivanov/hstd/hstdlib"
	_ "modernc.org/sqlite"
)

type DB struct {
	db *sql.DB
}

const migration = `
		CREATE TABLE IF NOT EXISTS links (
			idx     INTEGER PRIMARY KEY,
			comment TEXT NOT NULL DEFAULT '',
			enabled INTEGER NOT NULL DEFAULT 1
		);
		CREATE TABLE IF NOT EXISTS devices (
			link_idx INTEGER NOT NULL REFERENCES links(idx),
			name     TEXT NOT NULL,
			UNIQUE(link_idx, name)
		);
	`

func Open() (*DB, error) {
	path := filepath.Join(hstdlib.MustEnv("STATE_DIRECTORY"), "subsrv.db")
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(migration); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}
