// Package postgres implements a mymonies database interface with backed by
// PostgreSQL database.
package postgres

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/joneskoo/mymonies/database"
	"github.com/lib/pq"
	// Load postgresql driver
	_ "github.com/lib/pq"
)

// New opens a new PostgreSQL database connection and verifies connection
// by pinging the server.
func New(conn string) (database.Database, error) {
	db, err := sqlx.Connect("postgres", conn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	// use JSON field name as database field name
	db.Mapper = reflectx.NewMapperFunc("json", strings.ToLower)
	return &postgres{db}, nil
}

// type assertion
var _ database.Database = &postgres{}

type postgres struct {
	*sqlx.DB
}

var createTableSQL = []string{
	`CREATE TABLE IF NOT EXISTS imports (
		id		serial UNIQUE,
		filename	text,
		account		text NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS tags (
		id		serial UNIQUE,
		name		text UNIQUE)`,
	`CREATE TABLE IF NOT EXISTS records (
		id			serial UNIQUE,
		import_id		int REFERENCES imports(id) ON DELETE CASCADE,
		transaction_date	date ,
		value_date		date,
		payment_date		date,
		amount			double precision,
		payee_payer		text,
		account			text,
		bic			text,
		transaction		text,
		reference		text,
		payer_reference		text,
		message			text,
		card_number		text,
		tag_id			int REFERENCES tags(id))`,
	`CREATE TABLE IF NOT EXISTS patterns (
		id serial		UNIQUE,
		tag_id			int REFERENCES tags(id),
		account			text NOT NULL,
		query			text NOT NULL)`,
}

func (db *postgres) Close() error { return db.Close() }

func (db *postgres) CreateTables() error {
	txn, err := db.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()

	for _, q := range createTableSQL {
		if _, err := txn.Exec(q); err != nil {
			return err
		}
	}
	return txn.Commit()
}

func (db *postgres) ListAccounts() ([]string, error) {
	var accounts []string
	err := db.Select(&accounts, "SELECT DISTINCT account from imports")
	return accounts, err
}

func (db *postgres) Tag(id int) (database.Tag, error) {
	t := database.Tag{}
	err := db.QueryRowx("SELECT id, name from tags WHERE id = $1", id).StructScan(&t)
	return t, err
}

func (db *postgres) Import(id int) (database.Import, error) {
	t := database.Import{}
	err := db.QueryRowx("SELECT * from imports WHERE id = $1", id).StructScan(&t)
	return t, err
}

func (db *postgres) ListTags() ([]database.Tag, error) {
	var tags []database.Tag
	rows, err := db.Queryx("SELECT * from tags ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var t database.Tag
		err := rows.StructScan(&t)
		if err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (db *postgres) ListPatterns() ([]database.Pattern, error) {
	var tags []database.Pattern
	rows, err := db.Queryx("SELECT * from patterns ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p database.Pattern
		err := rows.StructScan(&p)
		if err != nil {
			return nil, err
		}
		tags = append(tags, p)
	}
	return tags, rows.Err()
}

func (db *postgres) Transactions() database.TransactionSet {
	return &transactionSet{db: db.DB}
}

// AddImport saves data into postgres atomically.
// If import fails, all changes are rolled back.
func (db *postgres) AddImport(data database.ImportTransactions) error {
	txn, err := db.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()

	var importid int
	const insertImport = "INSERT INTO imports (filename, account) VALUES ($1, $2) RETURNING id"
	if err := db.QueryRow(insertImport, data.Filename, data.Account).Scan(&importid); err != nil {
		return err
	}

	stmt, err := txn.Prepare(pq.CopyIn("records", "import_id", "transaction_date",
		"value_date", "payment_date", "amount", "payee_payer", "account", "bic",
		"transaction", "reference", "payer_reference", "message", "card_number"))
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range data.Transactions {
		_, err = stmt.Exec(importid, r.TransactionDate, r.ValueDate, r.PaymentDate,
			r.Amount, r.PayeePayer, r.Account, r.BIC, r.Transaction, r.Reference,
			r.PayerReference, r.Message, r.CardNumber)
		if err != nil {
			return err
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	err = stmt.Close()
	if err != nil {
		return err
	}

	return txn.Commit()
}

// SetRecordTag updates the Record Tag for record id to value tag.
func (db *postgres) SetRecordTag(id int, tag int) error {
	tagID := sql.NullInt64{Int64: int64(tag), Valid: tag > 0}
	_, err := db.Exec("UPDATE records SET tag_id = $1 WHERE id = $2", tagID, id)
	return err
}

// AddPattern saves a new Pattern
func (db *postgres) AddPattern(account, query string, tagID int) error {
	records, err := db.Transactions().Account(account).Search(query).Records()
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
