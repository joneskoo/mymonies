// Package datasource declares the interface for mymonies data sources.
package datasource

import "time"

// File represents set of transaction records for a particular account.
// Depending on the data source the time span may be a monthly statement
// or an arbitrary time period.
type File interface {
	// FileName returns the name of the file that is contained or empty string
	// if file name is not available.
	FileName() string

	// Account is the IBAN account number or masked credit card number
	// identifying the account the records are from.
	Account() string

	// Transactions returns the transaction records from the file.
	Transactions() []Transaction
}

type Transaction interface {
	ID() string
	TransactionDate() time.Time
	ValueDate() time.Time
	PaymentDate() time.Time
	Amount() float64
	PayeePayer() string
	Account() string
	BIC() string
	Transaction() string
	Reference() string
	PayerReference() string
	Message() string
	CardNumber() string
}
