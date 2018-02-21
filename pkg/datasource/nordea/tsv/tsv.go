// Package tsv implements Nordea bank TSV transaction record data source.
package tsv

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joneskoo/mymonies/pkg/database"
	"github.com/joneskoo/mymonies/pkg/datasource"
)

// FromFile loads transaction records from a Nordea TSV file.
func FromFile(filename string) (datasource.File, error) {
	lineEnd := []byte("\n\r\n")

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("could not read file: %v", err)
	}

	// The first line contains the account number.
	pos := bytes.Index(data, lineEnd)
	if pos == -1 {
		return nil, fmt.Errorf("unknown format, could not find line break LF CR LF")
	}
	accountLine, body := data[:pos], data[pos+len(lineEnd):]
	parts := bytes.SplitN(accountLine, []byte{'\t'}, 2)
	account := string(parts[1])

	// Read transaction records after account number.
	r := csv.NewReader(bytes.NewReader(body))
	r.Comma = '\t'
	r.FieldsPerRecord = 14
	_, _ = r.Read() // ignore first line
	transactions := []*database.Transaction{}
	for {
		r, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		rec, err := fromSlice(r)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, &rec)
	}
	return file{filename, account, transactions}, nil
}

type file struct {
	filename     string
	account      string
	transactions []*database.Transaction
}

func (f file) Account() string                       { return f.account }
func (f file) FileName() string                      { return filepath.Base(f.filename) }
func (f file) Transactions() []*database.Transaction { return f.transactions }

var fields = []string{
	"Kirjauspäivä",
	"Arvopäivä",
	"Maksupäivä",
	"Määrä",
	"Saaja/Maksaja",
	"Tilinumero",
	"BIC",
	"Tapahtuma",
	"Viite",
	"Maksajan viite",
	"Viesti",
	"Kortinnumero",
	"Kuitti",
}

const dateFormat = "02.01.2006"

var helsinki *time.Location

func fromSlice(r []string) (rec database.Transaction, err error) {
	td, err := time.ParseInLocation(dateFormat, r[0], helsinki)
	if err != nil {
		return rec, fmt.Errorf("bad transaction date format: %v", err)
	}
	vd, err := time.ParseInLocation(dateFormat, r[1], helsinki)
	if err != nil {
		return rec, fmt.Errorf("bad value date format: %v", err)
	}
	pd, err := time.ParseInLocation(dateFormat, r[2], helsinki)
	if err != nil {
		return rec, fmt.Errorf("bad payment date format: %v", err)
	}
	amount, err := strconv.ParseFloat(strings.Replace(r[3], ",", ".", 1), 64)
	if err != nil {
		return rec, fmt.Errorf("bad amount format: %v", err)
	}
	rec = database.Transaction{
		TransactionDate: td,
		ValueDate:       vd,
		PaymentDate:     pd,
		Amount:          amount,
		PayeePayer:      r[4],
		Account:         r[5],
		BIC:             r[6],
		Transaction:     r[7],
		Reference:       r[8],
		PayerReference:  r[9],
		Message:         r[10],
		CardNumber:      r[11],
		// Receipt:         r[12], // receipt column ignored
	}
	return rec, err
}

func init() {
	var err error
	helsinki, err = time.LoadLocation("Europe/Helsinki")
	if err != nil {
		panic(err)
	}
}
