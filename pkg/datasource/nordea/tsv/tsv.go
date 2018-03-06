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

	"github.com/joneskoo/mymonies/pkg/datasource"
	"github.com/joneskoo/mymonies/pkg/rpc/mymonies"
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
	var transactions []*mymonies.Transaction
loop:
	for {
		var t *mymonies.Transaction
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
	transactions []*mymonies.Transaction
}

func (f File) Account() string                       { return f.account }
func (f File) FileName() string                      { return filepath.Base(f.filename) }
func (f File) Transactions() []*mymonies.Transaction { return f.transactions }

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

func date(t time.Time) string {
	return t.UTC().Truncate(24 * time.Hour).Format(time.RFC3339)
}

func fromSlice(r []string) (t *mymonies.Transaction, err error) {
	p := new(safeParser)
	t = &mymonies.Transaction{
		TransactionDate: p.date(r[0], "transaction date"),
		ValueDate:       p.date(r[1], "value date"),
		PaymentDate:     p.date(r[2], "payment date"),
		Amount:          p.amount(r[3], "amount"),
		PayeePayer:      r[4],
		Account:         r[5],
		Bic:             r[6],
		Transaction:     r[7],
		Reference:       r[8],
		PayerReference:  r[9],
		Message:         joinMessage(r[10]),
		CardNumber:      r[11],
		// Receipt:         r[12], // receipt column ignored
	}
	if p.err != nil {
		return nil, p.err
	}
	return t, p.err
}

// joinMessage removes "line wrapping" from message field.
//
// If the message is over 35 characters long, there is a space after
// every 35 characters in the Nordea TSV format. It is like it
// was line wrapped, but new lines replaced with space.
//
//	Lorem ipsum dolor sit amet, consec tetur adipiscing elit. Curabitur d
//	                                  ^ extra space
func joinMessage(msg string) string {
	var b bytes.Buffer
	for i := 0; i < len(msg); i += 36 {
		end := i + 35
		if end > len(msg)-1 {
			end = len(msg)
		} else {
			// If format changes and there are no extra spaces, return original.
			if end+1 < len(msg) && msg[end] != ' ' {
				return msg
			}
		}
		// if msg[end] != ' ' {
		// 	return msg
		// }
		b.Write([]byte(msg[i:end]))
	}
	return b.String()
}

type safeParser struct {
	err error
}

func (p *safeParser) date(v, field string) string {
	if p.err != nil {
		return ""
	}
	var t time.Time
	t, p.err = time.ParseInLocation(dateFormat, v, helsinki)
	if p.err != nil {
		p.err = fmt.Errorf("bad %v format: %v", field, p.err)
		return ""
	}
	return t.UTC().Truncate(24 * time.Hour).Format(time.RFC3339)
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
