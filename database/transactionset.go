package database

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Records executes the query and returns matching transactions.
func (db *Postgres) Transactions(filters ...TransactionFilter) ([]Transaction, error) {
	t := &transactionSet{db: db.DB}
	for _, f := range filters {
		f(t)
	}

	if t.err != nil {
		return nil, t.err
	}

	t.Columns = []string{"records.*"}
	t.From = "records LEFT OUTER JOIN imports ON records.import_id = imports.id"
	t.OrderBy = "transaction_date DESC, records.id"
	fmt.Printf("SQL: %v\n", t.SQL())
	rows, err := t.db.NamedQuery(t.SQL(), t.arg())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]Transaction, 0)
	for rows.Next() {
		var t Transaction
		_ = rows.StructScan(&t)
		records = append(records, t)
	}
	return records, rows.Err()
}

// SumTransactionsByTag executes the query and returns total amounts of
// transactions by tag.
func (db *Postgres) SumTransactionsByTag(filters ...TransactionFilter) (map[string]float64, error) {
	t := &transactionSet{db: db.DB}
	for _, f := range filters {
		f(t)
	}

	if t.err != nil {
		return nil, t.err
	}
	t.Columns = []string{"coalesce(tags.name, '?')", "sum(amount)"}
	t.From = `records
		LEFT OUTER JOIN imports ON records.import_id = imports.id
		LEFT OUTER JOIN tags ON records.tag_id = tags.id`
	t.GroupBy = "1"
	t.OrderBy = "abs(sum(amount)) DESC"
	fmt.Printf("SQL: %v\n", t.SQL())
	rows, err := t.db.NamedQuery(t.SQL(), t.arg())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := make(map[string]float64)
	for rows.Next() {
		var tag string
		var amount float64
		_ = rows.Scan(&tag, &amount)
		if tag == "" {
			tag = "?"
		}
		tags[tag] = amount
	}
	return tags, rows.Err()
}

type TransactionFilter func(*transactionSet)

// transactionSet is a filterable set of transactions.
type transactionSet struct {
	db        *sqlx.DB
	recordID  int
	account   string
	search    string
	startDate time.Time
	endDate   time.Time
	err       error
	selectQuery
}

func (t *transactionSet) arg() map[string]interface{} {
	return map[string]interface{}{
		"record_id": t.recordID,
		"account":   t.account,
		"search":    t.search,
		"start":     t.startDate,
		"end":       t.endDate,
	}
}

func noop() TransactionFilter {
	return func(*transactionSet) {}
}

// Id filters to only include a specific transaction by id.
func Id(id int) TransactionFilter {
	if id == 0 {
		return noop()
	}
	return func(t *transactionSet) {
		t.AndWhere("records.id = :record_id")
		t.recordID = id
	}
}

// Account filters to only include transactions from account.
func Account(account string) TransactionFilter {
	if account == "" {
		return noop()
	}
	return func(t *transactionSet) {
		t.AndWhere("imports.account = :account")
		t.account = account
	}
}

// Month filters to only include transactions during a given month.
func Month(month string) TransactionFilter {
	if month == "" {
		return noop()
	}
	return func(t *transactionSet) {
		var startDate, endDate time.Time
		if month != "" {
			var err error
			startDate, err = time.Parse("2006-01", month)
			if err != nil {
				t.err = err
			}
			endDate = startDate.AddDate(0, 1, -1)
			t.AndWhere("records.transaction_date BETWEEN :start AND :end")
			t.startDate = startDate
			t.endDate = endDate
		}
	}
}

// Search filters to only include transactions containing the
// query exactly as a value of a field.
func Search(query string) TransactionFilter {
	if query == "" {
		return noop()
	}
	return func(t *transactionSet) {
		t.AndWhere(":search IN (payee_payer, records.account, transaction, reference, payer_reference, message)")
		t.search = query
	}
}
