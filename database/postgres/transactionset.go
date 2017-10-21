package postgres

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joneskoo/mymonies/database"
)

// type assertion
var _ database.TransactionSet = transactionSet{}

type transactionSet struct {
	db        *sqlx.DB
	account   string
	search    string
	startDate time.Time
	endDate   time.Time
	err       error
	selectQuery
}

func (q transactionSet) Account(account string) database.TransactionSet {
	q.AndWhere("imports.account = :account")
	q.account = account
	return q
}

func (q transactionSet) Month(month string) database.TransactionSet {
	var startDate, endDate time.Time
	if month != "" {
		var err error
		startDate, err = time.Parse("2006-01", month)
		if err != nil {
			q.err = err
		}
		endDate = startDate.AddDate(0, 1, -1)
		q.AndWhere("records.transaction_date BETWEEN :start AND :end")
		q.startDate = startDate
		q.endDate = endDate
	}
	return q
}
func (q transactionSet) Search(query string) database.TransactionSet {
	q.AndWhere(":search IN (payee_payer, records.account, transaction, reference, payer_reference, message)")
	q.search = query
	return q
}

func (q transactionSet) Records() ([]database.Transaction, error) {
	if q.err != nil {
		return nil, q.err
	}
	q.Columns = []string{"*"}
	q.From = "records LEFT OUTER JOIN imports ON records.import_id = imports.id"
	q.OrderBy = "transaction_date, records.id"
	fmt.Printf("SQL: %v\n", q.SQL())
	rows, err := q.db.NamedQuery(q.SQL(), q.arg())
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

func (q transactionSet) SumTransactionsByTag() (map[string]float64, error) {
	if q.err != nil {
		return nil, q.err
	}
	q.From = "records LEFT OUTER JOIN imports ON records.import_id = imports.id"
	q.Columns = []string{"tag", "sum(amount)"}
	q.GroupBy = "1"
	fmt.Printf("SQL: %v\n", q.SQL())
	rows, err := q.db.NamedQuery(q.SQL(), q.arg())
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

func (q transactionSet) arg() map[string]interface{} {
	return map[string]interface{}{
		"account": q.account,
		"search":  q.search,
		"start":   q.startDate,
		"end":     q.endDate,
	}
}
