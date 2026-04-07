package repo

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/realglebivanov/hstd/hstdlib/xrayconf"
	_ "modernc.org/sqlite"
)

func (d *DB) GetActiveConfig() (*xrayconf.Config, error) {
	var data string
	err := d.db.QueryRow(`SELECT c.config FROM conns c JOIN active a ON a.conn_id = c.id`).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no active conn selected")
	}
	if err != nil {
		return nil, err
	}
	var cfg xrayconf.Config
	return &cfg, json.Unmarshal([]byte(data), &cfg)
}

func (d *DB) AddConn(cfg *xrayconf.Config) error {
	conn, err := NewXrayConn(cfg, "")
	if err != nil {
		return err
	}
	_, err = d.db.Exec(`INSERT OR IGNORE INTO conns (id, remark, config) VALUES (?, ?, ?)`,
		conn.ID, conn.Remark, conn.Config)
	return err
}

func (d *DB) GetSubs() ([]*XraySub, error) {
	var subs []*XraySub
	return subs, d.db.Select(&subs, `SELECT id, url FROM subs`)
}

func (d *DB) AddSub(url string) error {
	sub := NewXraySub(url)
	_, err := d.db.Exec(`INSERT OR IGNORE INTO subs (id, url) VALUES (?, ?)`, sub.ID, sub.URL)
	return err
}

func (d *DB) RemoveSub(id string) error {
	return expectRows(d.db.Exec(`DELETE FROM subs WHERE id = ?`, id))
}

func (d *DB) GetConns() ([]*XrayConnInfo, error) {
	var conns []*XrayConnInfo
	return conns, d.db.Select(&conns, `
		SELECT c.id, c.remark, c.config, COALESCE(c.sub_id, '') AS sub_id, (a.conn_id IS NOT NULL) AS active
		FROM conns c LEFT JOIN active a ON a.conn_id = c.id`)
}

func (d *DB) RemoveConn(id string) (activeChanged bool, err error) {
	if err := expectRows(d.db.Exec(`DELETE FROM conns WHERE id = ?`, id)); err != nil {
		return false, err
	}
	var hasActive bool
	d.db.QueryRow(`SELECT 1 FROM active LIMIT 1`).Scan(&hasActive)
	return !hasActive, nil
}

func (d *DB) ChooseConn(id string) error {
	_, err := d.db.Exec(
		`INSERT INTO active (id, conn_id) VALUES (1, ?) ON CONFLICT(id) DO UPDATE SET conn_id = ?`,
		id, id)
	return err
}

func expectRows(res sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}
