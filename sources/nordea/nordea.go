package nordea

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
)

type File struct {
	Account string
	Records []Record
}

func FromTsv(filename string) (file File, err error) {
	lineEnd := []byte("\n\r\n")

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return file, fmt.Errorf("could not read file: %v", err)
	}
	pos := bytes.Index(data, lineEnd)
	if pos == -1 {
		return file, fmt.Errorf("unknown format, could not find line break LF CR LF")
	}
	account, body := data[:pos], data[pos+len(lineEnd):]
	parts := bytes.SplitN(account, []byte{'\t'}, 2)
	file.Account = string(parts[1])
	r := csv.NewReader(bytes.NewReader(body))
	r.Comma = '\t'
	r.FieldsPerRecord = 14
	_, _ = r.Read() // ignore first line
	for {
		r, err := r.Read()
		if err == io.EOF {
			return file, nil
		}
		if err != nil {
			return file, err
		}
		rec, err := fromSlice(r)
		if err != nil {
			return file, err
		}
		file.Records = append(file.Records, rec)
	}
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

type Record struct {
	TransactionDate time.Time `json:"transaction_date"`
	ValueDate       time.Time `json:"value_date"`
	PaymentDate     time.Time `json:"payment_date"`
	Amount          float64   `json:"amount"`
	PayeePayer      string    `json:"payee_payer"`
	Account         string    `json:"account"`
	BIC             string    `json:"bic"`
	Transaction     string    `json:"transaction"`
	Reference       string    `json:"reference"`
	PayerReference  string    `json:"payer_reference"`
	Message         string    `json:"message"`
	CardNumber      string    `json:"card_number"`
	Receipt         string    `json:"receipt"`
}

func (r Record) String() string {
	return fmt.Sprintf("%s %-30s %8.2f", r.TransactionDate.Format(dateFormat), r.PayeePayer, r.Amount)
}

const dateFormat = "02.01.2006"

var helsinki *time.Location

func fromSlice(r []string) (rec Record, err error) {
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
	rec = Record{
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
		Receipt:         r[12],
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
