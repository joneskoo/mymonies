// Package database manages storage for Mymonies state.
package database

import (
	"time"

	"github.com/lib/pq"

	// Load postgresql driver
	_ "github.com/lib/pq"
)

// Database represents the storage for mymonies data.
type Database interface {
	// Close closes the connection to the database.
	// Database connection is normally called only at program exit.
	Close() error

	// CreateTables creates any missing database tables.
	// This is safe to call multiple times.
	CreateTables() error

	// ListAccounts lists the accounts for which transaction records are available.
	ListAccounts() ([]string, error)

	// ListTags lists the tags configured.
	ListTags() ([]Tag, error)

	// ListRecordsByAccount lists transactions. Any of the parameters may be empty
	// to include all records.
	ListRecordsByAccount(account, month, search string) ([]Record, error)

	// SumTransactionsByTag returns the sum of transaction amounts by tag.
	SumTransactionsByTag(account, month, search string) (map[string]float64, error)

	// AddImport saves the transaction data to database.
	AddImport(data Import) error

	// SetRecordTag updates the tag field of a transaction.
	SetRecordTag(id int, tag string) error
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
	ID              int       `json:"id"`
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
