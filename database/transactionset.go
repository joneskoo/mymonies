package database

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// TransactionSet is a filterable set of transactions.
type TransactionSet struct {
	db        *sqlx.DB
	recordID  int
	account   string
	search    string
	startDate time.Time
	endDate   time.Time
	err       error
	selectQuery
}

func (t *TransactionSet) arg() map[string]interface{} {
	return map[string]interface{}{
		"record_id": t.recordID,
		"account":   t.account,
		"search":    t.search,
		"start":     t.startDate,
		"end":       t.endDate,
	}
}

// Id filters to only include a specific transaction by id.
func (t *TransactionSet) Id(id int) *TransactionSet {
	t.AndWhere("records.id = :record_id")
	t.recordID = id
	return t
}

// Account filters to only include transactions from account.
func (t *TransactionSet) Account(account string) *TransactionSet {
	t.AndWhere("imports.account = :account")
	t.account = account
	return t
}

// Month filters to only include transactions during a given month.
func (t *TransactionSet) Month(month string) *TransactionSet {
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
	return t
}

// Search filters to only include transactions containing the
// query exactly as a value of a field.
func (t *TransactionSet) Search(query string) *TransactionSet {
	t.AndWhere(":search IN (payee_payer, records.account, transaction, reference, payer_reference, message)")
	t.search = query
	return t
}

// Records executes the query and returns matching transactions.
func (t *TransactionSet) Records() ([]Transaction, error) {
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

	var records []Transaction
	for rows.Next() {
		var t Transaction
		_ = rows.StructScan(&t)
		records = append(records, t)
	}
	return records, rows.Err()

}

// SumTransactionsByTag executes the query and returns total amounts of
// transactions by tag.
func (t *TransactionSet) SumTransactionsByTag() (map[string]float64, error) {
	if t.err != nil {
		return nil, t.err
	}
	t.Columns = []string{"coalesce(tags.name, '?')", "sum(abs(amount))"}
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
