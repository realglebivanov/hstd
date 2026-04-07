package repo

import (
	"database/sql"
	"errors"
	"maps"
	"slices"

	"github.com/jmoiron/sqlx"
	"github.com/realglebivanov/hstd/hstdlib/xrayconf"
)

func (d *DB) SyncConns(cfgs map[string][]*xrayconf.Config) error {
	tx, err := d.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	prevActiveID, err := deleteXrayConns(cfgs, tx)
	if err != nil {
		return err
	}

	if err := insertXrayConns(cfgs, tx); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		INSERT OR IGNORE INTO active (id, conn_id)
		SELECT 1, COALESCE(
			(SELECT id FROM conns WHERE id = ?),
			(SELECT id FROM conns ORDER BY RANDOM() LIMIT 1)
		)`, prevActiveID); err != nil {
		return err
	}

	return tx.Commit()
}

func deleteXrayConns(cfgs map[string][]*xrayconf.Config, tx *sqlx.Tx) (string, error) {
	var prevActiveID string
	err := tx.QueryRow(`SELECT conn_id FROM active LIMIT 1`).Scan(&prevActiveID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	subIds := slices.Collect(maps.Keys(cfgs))
	query, args, err := sqlx.In(`DELETE FROM conns WHERE sub_id IN (?)`, subIds)
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(query, args...); err != nil {
		return "", err
	}

	return prevActiveID, nil
}

func insertXrayConns(cfgs map[string][]*xrayconf.Config, tx *sqlx.Tx) error {
	var conns []*XrayConn
	for subID, configs := range cfgs {
		for _, cfg := range configs {
			if conn, err := NewXrayConn(cfg, subID); err == nil {
				conns = append(conns, conn)
			}
		}
	}
	if len(conns) > 0 {
		if _, err := tx.NamedExec(
			`INSERT OR IGNORE
				INTO conns (id, remark, config, sub_id)
				VALUES (:id, :remark, :config, :sub_id)`, conns); err != nil {
			return err
		}
	}
	return nil
}
