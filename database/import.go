package database

import "github.com/lib/pq"

// Import represents one transaction report imported from a file to
// database.
type Import struct {
	ID       int    `json:"id"`
	Filename string `json:"filename,omitempty"`
	Account  string `json:"account,omitempty"`
}

var importsCreateTableSQL = `
CREATE TABLE IF NOT EXISTS imports (
	id		serial UNIQUE,
	filename	text,
	account		text NOT NULL)`

// Import gets tag details from database by id.
func (db *Postgres) Import(id int) (Import, error) {
	t := Import{}
	err := db.QueryRowx("SELECT * from imports WHERE id = $1", id).StructScan(&t)
	return t, err
}

// ListAccounts lists the accounts for which transaction records are available.
func (db *Postgres) ListAccounts() ([]string, error) {
	var accounts []string
	err := db.Select(&accounts, "SELECT DISTINCT account from imports")
	return accounts, err
}

// AddImport saves data into postgres atomically.
// If import fails, all changes are rolled back.
func (db *Postgres) AddImport(filename, account string, transactions []*Transaction) error {
	txn, err := db.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()

	var importid int
	const insertImport = "INSERT INTO imports (filename, account) VALUES ($1, $2) RETURNING id"
	if err := db.QueryRow(insertImport, filename, account).Scan(&importid); err != nil {
		return err
	}

	stmt, err := txn.Prepare(pq.CopyIn("records", "import_id", "transaction_date",
		"value_date", "payment_date", "amount", "payee_payer", "account", "bic",
		"transaction", "reference", "payer_reference", "message", "card_number"))
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range transactions {
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
