package db

import (
	"database/sql"

	"github.com/realglebivanov/hstd/hstdlib/sublink"
	_ "modernc.org/sqlite"
)

type SublinkInfo struct {
	Index   int
	Version int
	Comment string
	Enabled bool
	Devices string
}

func (d *DB) IsEnabled(l *sublink.Sublink) (bool, error) {
	row := d.db.QueryRow(
		`SELECT links.enabled FROM links WHERE links.idx = ?`,
		l.Index,
	)

	var isEnabled bool
	err := row.Scan(&isEnabled)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return isEnabled, nil
}

func (d *DB) List(count int) ([]SublinkInfo, error) {
	rows, err := d.db.Query(`
		WITH RECURSIVE g(value) AS (
			SELECT 0
			UNION ALL
			SELECT value + 1 FROM g WHERE value < ?-1
		)
		SELECT
			g.value,
			COALESCE(l.comment, ''),
			COALESCE(l.enabled, 0),
			COALESCE(GROUP_CONCAT(d.name, char(10)), ''),
			COALESCE(l.version, 0)
		FROM g
		LEFT JOIN links l ON l.idx = g.value
		LEFT JOIN devices d ON d.link_idx = g.value AND d.last_seen > unixepoch() - 86400
		GROUP BY g.value
	`, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []SublinkInfo
	for rows.Next() {
		var li SublinkInfo
		if err := rows.Scan(&li.Index, &li.Comment, &li.Enabled, &li.Devices, &li.Version); err != nil {
			return nil, err
		}
		result = append(result, li)
	}
	return result, rows.Err()
}

func (d *DB) TrackDevice(l *sublink.Sublink, name string) (*SublinkInfo, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var li SublinkInfo
	row := tx.QueryRow(
		`INSERT INTO links (idx) VALUES (?)
		ON CONFLICT(idx) DO UPDATE SET version = version + 1
		RETURNING idx, comment, enabled, version`,
		l.Index,
	)
	if err := row.Scan(&li.Index, &li.Comment, &li.Enabled, &li.Version); err != nil {
		return nil, err
	}

	_, err = tx.Exec(
		`INSERT INTO devices (link_idx, name) VALUES (?, ?)
		 ON CONFLICT(link_idx, name) DO UPDATE SET last_seen = unixepoch()`,
		l.Index, name,
	)
	if err != nil {
		return nil, err
	}

	row = tx.QueryRow(`
		SELECT COALESCE(GROUP_CONCAT(name, char(10)), '')
		FROM devices WHERE link_idx = ? AND last_seen > unixepoch() - 86400`,
		l.Index,
	)
	if err := row.Scan(&li.Devices); err != nil {
		return nil, err
	}

	return &li, tx.Commit()
}

func (d *DB) UpdateLink(index int, comment *string, enabled *bool) (*SublinkInfo, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var li SublinkInfo
	row := tx.QueryRow(`
		INSERT INTO links (idx, comment, enabled) VALUES (?, COALESCE(?, ''), COALESCE(?, 0))
		ON CONFLICT(idx) DO UPDATE SET
			comment = COALESCE(?, links.comment),
			enabled = COALESCE(?, links.enabled),
			version = links.version + 1
		RETURNING idx, comment, enabled, version`,
		index, comment, enabled, comment, enabled,
	)
	if err := row.Scan(&li.Index, &li.Comment, &li.Enabled, &li.Version); err != nil {
		return nil, err
	}

	row = tx.QueryRow(`
		SELECT COALESCE(GROUP_CONCAT(name, char(10)), '')
		FROM devices WHERE link_idx = ? AND last_seen > unixepoch() - 86400`,
		index,
	)
	if err := row.Scan(&li.Devices); err != nil {
		return nil, err
	}

	return &li, tx.Commit()
}
