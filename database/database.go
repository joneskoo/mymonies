// Package database manages storage for Mymonies state.
package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq" // postgresql driver
)

// Database represents database connection.
type Database struct {
	conn *sql.DB
}

// Import represents one transaction report imported from a file to
// database.
type Import struct {
	ID       int      `json:"-"`
	Filename string   `json:"filename,omitempty"`
	Account  string   `json:"account,omitempty"`
	Records  []Record `json:"records,omitempty"`
}

const createImports = `CREATE TABLE IF NOT EXISTS imports (
	id SERIAL UNIQUE,
	filename TEXT,
	account TEXT NOT NULL)`

// Record represents one account transaction record.
type Record struct {
	ID              int       `json:"-"`
	TransactionDate time.Time `json:"transaction_date,omitempty"`
	ValueDate       time.Time `json:"value_date,omitempty"`
	PaymentDate     time.Time `json:"payment_date,omitempty"`
	Amount          float64   `json:"amount,omitempty"`
	PayeePayer      string    `json:"payee_payer,omitempty"`
	Account         string    `json:"account,omitempty"`
	BIC             string    `json:"bic,omitempty"`
	Transaction     string    `json:"transaction,omitempty"`
	Reference       string    `json:"reference,omitempty"`
	PayerReference  string    `json:"payer_reference,omitempty"`
	Message         string    `json:"message,omitempty"`
	CardNumber      string    `json:"card_number,omitempty"`
}

const createRecords = `CREATE TABLE IF NOT EXISTS records (
	id SERIAL UNIQUE,
	import_id INT REFERENCES imports(id) ON DELETE CASCADE,
	transaction_date DATE ,
	value_date DATE,
	payment_date DATE,
	amount DOUBLE PRECISION,
	payee_payer TEXT,
	account TEXT,
	bic TEXT,
	transaction TEXT,
	reference TEXT,
	payer_reference TEXT,
	message TEXT,
	card_number TEXT,
	tag TEXT REFERENCES tags(name))`

// Tag represents a transaction tag
type Tag struct {
	Name string
}

const createTags = `CREATE TABLE IF NOT EXISTS tags (
	id SERIAL UNIQUE,
	name TEXT UNIQUE)`

// New creates a mymonies database connection.
func New(conn string) (*Database, error) {
	db, err := sql.Open("postgres", conn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	return &Database{db}, nil
}

// CreateTables creates database tables if they do not exist.
func (db *Database) CreateTables() error {
	txn, err := db.conn.Begin()
	if err != nil {
		return err
	}

	queries := []string{createImports, createTags, createRecords}
	for _, q := range queries {
		_, err := txn.Exec(q)
		if err != nil {
			return err
		}
	}

	err = txn.Commit()
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

// ListAccounts lists the accounts with data stored in the database.
func (db *Database) ListAccounts() (accounts []string, err error) {
	rows, err := db.conn.Query("SELECT DISTINCT account from imports")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var account string
		if err = rows.Scan(&account); err != nil {
			return
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

// AddImport saves data into database atomically.
// If import fails, all changes are rolled back.
func (db *Database) AddImport(data Import) error {
	txn, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()

	var importid int
	const insertImport = "INSERT INTO imports (filename, account) VALUES ($1, $2) RETURNING id"
	if err := db.conn.QueryRow(insertImport, data.Filename, data.Account).Scan(&importid); err != nil {
		return err
	}

	stmt, err := txn.Prepare(pq.CopyIn("records", "import_id", "transaction_date",
		"value_date", "payment_date", "amount", "payee_payer", "account", "bic",
		"transaction", "reference", "payer_reference", "message", "card_number"))
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range data.Records {
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