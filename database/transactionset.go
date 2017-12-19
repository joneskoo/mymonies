package database

import (
	"time"
)

// Transactions returns all transactions, optionally limited by filters.
func (db *Postgres) Transactions(filters ...TransactionFilter) ([]Transaction, error) {
	query := &selectQuery{
		Columns: []string{"records.*"},
		From: `records
			LEFT OUTER JOIN imports ON records.import_id = imports.id`,
		OrderBy: "transaction_date DESC, records.id",
		args:    make(map[string]interface{}),
	}
	for _, f := range filters {
		f(query)
	}

	if query.err != nil {
		return nil, query.err
	}

	rows, err := db.NamedQuery(query.SQL(), query.args)
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
	query := &selectQuery{
		Columns: []string{"coalesce(tags.name, '?')", "sum(amount)"},
		From: `records
			LEFT OUTER JOIN imports ON records.import_id = imports.id
			LEFT OUTER JOIN tags ON records.tag_id = tags.id`,
		GroupBy: "1",
		OrderBy: "abs(sum(amount)) DESC",

		args: make(map[string]interface{}),
	}
	for _, f := range filters {
		f(query)
	}

	if query.err != nil {
		return nil, query.err
	}
	rows, err := db.NamedQuery(query.SQL(), query.args)
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

// TransactionFilter is an option that limits records returned by the query.
type TransactionFilter func(*selectQuery)

func noOpFilter(*selectQuery) {}

// Id filters to only include a specific transaction by id.
func Id(id int) TransactionFilter {
	if id == 0 {
		return noOpFilter
	}
	return func(t *selectQuery) {
		t.AndWhere("records.id = :record_id")
		t.args["record_id"] = id
	}
}

// Account filters to only include transactions from account.
func Account(account string) TransactionFilter {
	if account == "" {
		return noOpFilter
	}
	return func(t *selectQuery) {
		t.AndWhere("imports.account = :account")
		t.args["account"] = account
	}
}

// Month filters to only include transactions during a given month.
func Month(month string) TransactionFilter {
	if month == "" {
		return noOpFilter
	}
	return func(t *selectQuery) {
		var startDate, endDate time.Time
		if month != "" {
			var err error
			startDate, err = time.Parse("2006-01", month)
			if err != nil {
				t.err = err
			}
			endDate = startDate.AddDate(0, 1, -1)
			t.AndWhere("records.transaction_date BETWEEN :start AND :end")
			t.args["start"] = startDate
			t.args["end"] = endDate
		}
	}
}

// Search filters to only include transactions containing the
// query exactly as a value of a field.
func Search(query string) TransactionFilter {
	if query == "" {
		return noOpFilter
	}
	return func(t *selectQuery) {
		t.AndWhere(":search IN (payee_payer, records.account, transaction, reference, payer_reference, message)")
		t.args["search"] = query
	}
}
