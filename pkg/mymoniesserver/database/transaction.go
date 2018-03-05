package database

import (
	"database/sql"
	"time"
)

// Transaction represents one account transaction record.
type Transaction struct {
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
	TagID           int       `json:"tag_id"`
	ImportID        int       `json:"import_id"`
}

// SetRecordTag updates the Record Tag for record id to value tag.
func (db *Postgres) SetRecordTag(id int, tag int) error {
	tagID := sql.NullInt64{Int64: int64(tag), Valid: tag > 0}
	_, err := db.Exec("UPDATE records SET tag_id = $1 WHERE id = $2", tagID, id)
	return err
}
