// Package postgres implements a mymonies database interface with backed by
// PostgreSQL database.
package postgres

import (
	"fmt"
	"strings"
	"time"

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

func (db *postgres) ListTags() ([]database.Tag, error) {
	var tags []database.Tag
	rows, err := db.Queryx("SELECT id, name, patterns from tags ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var t database.Tag
		err := rows.Scan(&t.ID, &t.Name, (*pq.StringArray)(&t.Patterns))
		if err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (db *postgres) Transactions() database.TransactionSet {
	return &selectQuery{db: db.DB}
}

// AddImport saves data into postgres atomically.
// If import fails, all changes are rolled back.
func (db *postgres) AddImport(data database.Import) error {
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

	// Ensure all tags exist in postgres
	tags := make(map[string]bool)
	for _, r := range data.Transactions {
		tags[r.Tag] = true
	}
	for tag := range tags {
		_, err := db.Exec("INSERT INTO tags (name) VALUES ($1) ON CONFLICT DO NOTHING", tag)
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

	for _, r := range data.Transactions {
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
func (db *postgres) SetRecordTag(id int, tag string) error {
	_, err := db.Exec("UPDATE records SET tag = $1 WHERE id = $2", tag, id)
	return err
}

type selectQuery struct {
	db        *sqlx.DB
	columns   []string
	from      string
	where     []string
	groupBy   string
	orderBy   string
	account   string
	search    string
	startDate time.Time
	endDate   time.Time
	err       error
}

// type assertion
var _ database.TransactionSet = selectQuery{}

func (q selectQuery) Account(account string) database.TransactionSet {
	q.andWhere("imports.account = :account")
	q.account = account
	return q
}

func (q selectQuery) Month(month string) database.TransactionSet {
	var startDate, endDate time.Time
	if month != "" {
		var err error
		startDate, err = time.Parse("2006-01", month)
		if err != nil {
			q.err = err
		}
		endDate = startDate.AddDate(0, 1, -1)
		q.andWhere("records.transaction_date BETWEEN :start AND :end")
		q.startDate = startDate
		q.endDate = endDate
	}
	return q
}
func (q selectQuery) Search(query string) database.TransactionSet {
	q.andWhere(":search IN (payee_payer, records.account, transaction, reference, payer_reference, message)")
	q.search = query
	return q
}

func (q selectQuery) Records() ([]database.Transaction, error) {
	if q.err != nil {
		return nil, q.err
	}
	q.columns = []string{"*"}
	q.from = "records LEFT OUTER JOIN imports ON records.import_id = imports.id"
	q.orderBy = "transaction_date, records.id"
	fmt.Printf("SQL: %v\n", q.sql())
	rows, err := q.db.NamedQuery(q.sql(), q.arg())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []database.Transaction
	for rows.Next() {
		var t database.Transaction
		_ = rows.StructScan(&t)
		records = append(records, t)
	}
	return records, rows.Err()

}

func (q selectQuery) SumTransactionsByTag() (map[string]float64, error) {
	if q.err != nil {
		return nil, q.err
	}
	q.from = "records LEFT OUTER JOIN imports ON records.import_id = imports.id"
	q.columns = []string{"tag", "sum(amount)"}
	q.groupBy = "1"
	fmt.Printf("SQL: %v\n", q.sql())
	rows, err := q.db.NamedQuery(q.sql(), q.arg())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := make(map[string]float64)
	for rows.Next() {
		var tag string
		var amount float64
		_ = rows.Scan(&tag, &amount)
		tags[tag] = amount
	}
	return tags, rows.Err()

}

func (q selectQuery) arg() map[string]interface{} {
	return map[string]interface{}{
		"account": q.account,
		"search":  q.search,
		"start":   q.startDate,
		"end":     q.endDate,
	}
}
func (q selectQuery) sql() string {
	columns := "SELECT " + strings.Join(q.columns, ", ")
	var from, where, groupBy, orderBy string
	if len(q.from) > 0 {
		from = " FROM " + q.from
	}
	if len(q.where) > 0 {
		where = " WHERE " + strings.Join(q.where, " AND ")
	}
	if len(q.groupBy) > 0 {
		groupBy = " GROUP BY " + q.groupBy
	}
	if len(q.orderBy) > 0 {
		orderBy = " ORDER BY " + q.orderBy
	}
	return strings.Join([]string{columns, from, where, groupBy, orderBy}, "")
}

func (q *selectQuery) andWhere(cond string) {
	q.where = append(q.where, cond)
}
