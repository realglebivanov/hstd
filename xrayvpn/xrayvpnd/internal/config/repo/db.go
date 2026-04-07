package repo

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

const dbPath = "/etc/xrayvpn/state.db"

const migration = `
	CREATE TABLE IF NOT EXISTS subs (
		id  TEXT PRIMARY KEY,
		url TEXT NOT NULL UNIQUE
	);
	CREATE TABLE IF NOT EXISTS conns (
		id     TEXT PRIMARY KEY,
		remark TEXT NOT NULL,
		config TEXT NOT NULL,
		sub_id TEXT REFERENCES subs(id) ON DELETE CASCADE
	);
	CREATE TABLE IF NOT EXISTS active (
		id      INTEGER PRIMARY KEY CHECK (id = 1),
		conn_id TEXT NOT NULL REFERENCES conns(id) ON DELETE CASCADE
	);
`

type DB struct {
	db *sqlx.DB
}

func Open() (*DB, error) {
	db, err := sqlx.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	if _, err = db.Exec(migration); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &DB{db: db}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}
