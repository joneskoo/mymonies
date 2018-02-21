// Package datasource declares the interface for mymonies data sources.
package datasource

import "github.com/joneskoo/mymonies/pkg/database"

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
	Transactions() []*database.Transaction
}
