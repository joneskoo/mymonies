package database

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
)

// Pattern represents a rule to match transactions on Account by search pattern
// Query and tags them with TagID.
type Pattern struct {
	ID      int    `json:"id"`
	TagID   int    `json:"tag_id"`
	Account string `json:"account"`
	Query   string `json:"query"`
}

var patternsCreateTableSQL = `
CREATE TABLE IF NOT EXISTS patterns (
	id serial		UNIQUE,
	tag_id			int REFERENCES tags(id),
	account			text NOT NULL,
	query			text NOT NULL)`

// AddPattern adds a new rule to map transactions on account matching query to tagID.
func (db *Postgres) AddPattern(account, query string, tagID int) error {
	records, err := db.Transactions(Account(account), Search(query))
	ids := make([]string, len(records))
	for i, r := range records {
		ids[i] = strconv.Itoa(r.ID)
	}
	fmt.Println(ids)

	txn, err := db.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()

	_, err = txn.Exec("INSERT INTO patterns (account, query, tag_id) VALUES ($1, $2, $3)", account, query, tagID)
	if err != nil {
		return err
	}
	_, err = txn.Exec("UPDATE records SET tag_id = $1 WHERE id IN ("+strings.Join(ids, ",")+") AND tag_id IS NULL", tagID)
	if err != nil {
		return err
	}
	txn.Commit()
	return err
}

// ListPatterns lists the patterns configured.
func (db *Postgres) ListPatterns(tagID int) ([]Pattern, error) {
	var tags []Pattern
	var rows *sqlx.Rows
	var err error
	if tagID > 0 {
		rows, err = db.Queryx("SELECT * from patterns WHERE tag_id = $1 ORDER BY query, account, id", tagID)
	} else {
		rows, err = db.Queryx("SELECT * from patterns ORDER BY account, id")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p Pattern
		err := rows.StructScan(&p)
		if err != nil {
			return nil, err
		}
		tags = append(tags, p)
	}
	return tags, rows.Err()
}
