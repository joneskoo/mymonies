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
)

// FromFile loads transaction records from a Nordea TSV file.
func FromFile(filename string) (*File, error) {
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
	var transactions []*Transaction
loop:
	for {
		var t *Transaction
		r, err := r.Read()
		switch err {
		case io.EOF:
			break loop
		case nil:
			t, err = fromSlice(r)
		}
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}
	return &File{filename, account, transactions}, nil
}

type File struct {
	filename     string
	account      string
	transactions []*Transaction
}

func (f File) Account() string              { return f.account }
func (f File) FileName() string             { return filepath.Base(f.filename) }
func (f File) Transactions() []*Transaction { return f.transactions }

type Transaction struct {
	ID              string
	TransactionDate time.Time
	ValueDate       time.Time
	PaymentDate     time.Time
	Amount          float64
	PayeePayer      string
	Account         string
	BIC             string
	Transaction     string
	Reference       string
	PayerReference  string
	Message         string
	CardNumber      string
}

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

func fromSlice(r []string) (t *Transaction, err error) {
	p := new(safeParser)
	t = &Transaction{
		TransactionDate: p.date(r[0], "transaction date"),
		ValueDate:       p.date(r[1], "value date"),
		PaymentDate:     p.date(r[2], "payment date"),
		Amount:          p.amount(r[3], "amount"),
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
	if p.err != nil {
		return nil, p.err
	}
	return t, p.err
}

type safeParser struct {
	err error
}

func (p *safeParser) date(v, field string) time.Time {
	if p.err != nil {
		return time.Time{}
	}
	var t time.Time
	t, p.err = time.ParseInLocation(dateFormat, v, helsinki)
	if p.err != nil {
		p.err = fmt.Errorf("bad %v format: %v", field, p.err)
		return time.Time{}
	}
	return t
}

func (p *safeParser) amount(v, field string) (a float64) {
	if p.err != nil {
		return
	}
	a, p.err = strconv.ParseFloat(strings.Replace(v, ",", ".", 1), 64)
	if p.err != nil {
		p.err = fmt.Errorf("bad %v format: %v", field, p.err)
	}
	return
}

var helsinki *time.Location

func init() {
	var err error
	helsinki, err = time.LoadLocation("Europe/Helsinki")
	if err != nil {
		panic(err)
	}
}
