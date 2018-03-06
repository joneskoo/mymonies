package datasource

import "github.com/joneskoo/mymonies/pkg/rpc/mymonies"

type File interface {
	// FileName returns the name of the file that is contained or empty string
	// if file name is not available.
	FileName() string

	// Account is the IBAN account number or masked credit card number
	// identifying the account the records are from.
	Account() string

	// Transactions returns the transaction records from the file.
	Transactions() []*mymonies.Transaction
}
