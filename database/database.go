// Package database manages storage for Mymonies state.
package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"

	// Load postgresql driver
	_ "github.com/lib/pq"
)

// Database represents database connection.
type Database struct {
	conn *sql.DB
}

// Import represents one transaction report imported from a file to
// database.
type Import struct {
	ID       int       `json:"-"`
	Filename string    `json:"filename,omitempty"`
	Account  string    `json:"account,omitempty"`
	Records  []*Record `json:"records,omitempty"`
}

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
	Tag             string    `json:"tag,omitempty"`
}

// Tag represents a transaction tag
type Tag struct {
	Name     string
	Patterns pq.StringArray
}

// New opens a new database connection.
func New(conn string) (*Database, error) {
	db, err := sql.Open("postgres", conn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &Database{db}, nil
}

// Close closes the database, releasing any open resources.
func (db *Database) Close() error { return db.conn.Close() }

// CreateTables creates database tables if they do not exist.
func (db *Database) CreateTables() error {
	txn, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()

	for _, q := range createTableSQL {
		_, err := txn.Exec(q)
		if err != nil {
			return err
		}
	}
	return txn.Commit()
}

var createTableSQL = []string{
	`CREATE TABLE IF NOT EXISTS imports (
		id		serial UNIQUE,
		filename	text,
		account		text NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS tags (
		id		serial UNIQUE,
		name		text UNIQUE,
		patterns	text[])`,
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
		tag			text REFERENCES tags(name))`,
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

// ListTags lists the tags stored in the database.
func (db *Database) ListTags() (tags []Tag, err error) {
	rows, err := db.conn.Query("SELECT name, patterns from tags ORDER BY name")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var tag Tag
		if err = rows.Scan(&tag.Name, &tag.Patterns); err != nil {
			return
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

// ListRecordsByAccount lists the records stored in the database for account.
func (db *Database) ListRecordsByAccount(account, month, search string) (records []Record, err error) {
	var startDate, interval string
	if month != "" {
		startDate = month + "-01"
		interval = "1 month"
	}

	query := `SELECT
		records.id, transaction_date, value_date, payment_date, amount,
		payee_payer, records.account, bic, transaction, reference, payer_reference,
		message, card_number, tag
	FROM records, imports
	WHERE
		records.import_id = imports.id AND
		($1 = '' OR imports.account = $1) AND
		(($2 = '' AND $3 = '') OR records.transaction_date BETWEEN $2::date and $2::date + $3::interval) AND
		($4 = '' OR payee_payer ILIKE $4 OR records.account = $4 OR transaction = $4 OR reference = $4 OR payer_reference = $4 OR message = $4)
	ORDER BY transaction_date, records.id`
	rows, err := db.conn.Query(query, account, startDate, interval, search)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var rec Record
		err = rows.Scan(&rec.ID, &rec.TransactionDate, &rec.ValueDate,
			&rec.PaymentDate, &rec.Amount, &rec.PayeePayer, &rec.Account,
			&rec.BIC, &rec.Transaction, &rec.Reference, &rec.PayerReference,
			&rec.Message, &rec.CardNumber, &rec.Tag)
		if err != nil {
			return
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// SumTransactionsByTag lists the records stored in the database for account.
func (db *Database) SumTransactionsByTag(account, month, search string) (map[string]float64, error) {
	var startDate, interval string
	if month != "" {
		startDate = month + "-01"
		interval = "1 month"
	}

	query := `SELECT tag, sum(amount) FROM records, imports
	WHERE
		records.import_id = imports.id AND
		($1 = '' OR imports.account = $1) AND
		(($2 = '' AND $3 = '') OR records.transaction_date BETWEEN $2::date and $2::date + $3::interval) AND
		($4 = '' OR payee_payer ILIKE $4 OR records.account = $4 OR transaction = $4 OR reference = $4 OR payer_reference = $4 OR message = $4)
	GROUP BY 1`
	rows, err := db.conn.Query(query, account, startDate, interval, search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := make(map[string]float64)
	for rows.Next() {
		var tag string
		var amount float64
		if err := rows.Scan(&tag, &amount); err != nil {
			return nil, err
		}
		tags[tag] = amount
	}
	return tags, rows.Err()
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

	// Ensure all tags exist in database
	tags := make(map[string]bool)
	for _, r := range data.Records {
		tags[r.Tag] = true
	}
	for tag := range tags {
		_, err := db.conn.Exec("INSERT INTO tags (name) VALUES ($1) ON CONFLICT DO NOTHING", tag)
		if err != nil {
			return err
		}
	}

	stmt, err := txn.Prepare(pq.CopyIn("records", "import_id", "transaction_date",
		"value_date", "payment_date", "amount", "payee_payer", "account", "bic",
		"transaction", "reference", "payer_reference", "message", "card_number",
		"tag"))
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range data.Records {
		_, err = stmt.Exec(importid, r.TransactionDate, r.ValueDate, r.PaymentDate,
			r.Amount, r.PayeePayer, r.Account, r.BIC, r.Transaction, r.Reference,
			r.PayerReference, r.Message, r.CardNumber, r.Tag)
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
func (db *Database) SetRecordTag(id int, tag string) error {
	_, err := db.conn.Exec("UPDATE records SET tag = $1 WHERE id = $2", tag, id)
	return err
}
