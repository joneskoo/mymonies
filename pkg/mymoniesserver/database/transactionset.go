package database

import (
	"time"
)

// TransactionFilter is an option that limits records returned by the query.
type TransactionFilter struct {
	Id      int
	Account string
	Month   string
	Query   string
}

func (f *TransactionFilter) Apply(t *selectQuery, args map[string]interface{}) error {
	if f.Id > 0 {
		t.AndWhere("records.id = :record_id")
		args["record_id"] = f.Id
	}

	if f.Account != "" {
		t.AndWhere("imports.account = :account")
		args["account"] = f.Account
	}

	if f.Month != "" {
		var startDate, endDate time.Time
		var err error
		startDate, err = time.Parse("2006-01", f.Month)
		if err != nil {
			return err
		}
		endDate = startDate.AddDate(0, 1, -1)
		t.AndWhere("records.transaction_date BETWEEN :start AND :end")
		args["start"] = startDate
		args["end"] = endDate
	}

	if f.Query != "" {
		t.AndWhere(":search IN (payee_payer, records.account, transaction, reference, payer_reference, message)")
		args["search"] = f.Query
	}
	return nil
}

// Transactions returns all transactions, optionally limited by filters.
func (db *Postgres) Transactions(tf TransactionFilter) ([]Transaction, error) {
	query := &selectQuery{
		Columns: []string{"records.*"},
		From: `records
			LEFT OUTER JOIN imports ON records.import_id = imports.id`,
		OrderBy: "transaction_date DESC, records.id",
	}

	args := make(map[string]interface{})
	if err := tf.Apply(query, args); err != nil {
		return nil, err
	}

	rows, err := db.NamedQuery(query.SQL(), args)
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
