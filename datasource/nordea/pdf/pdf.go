package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joneskoo/mymonies/database"
	"github.com/joneskoo/mymonies/datasource"

	"rsc.io/pdf"
)

func main() {
	f, err := FromFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAILED: %v", err)
		os.Exit(2)
	}
	fmt.Printf("Account: %q File: %q\n", f.Account(), f.FileName())
	for _, tx := range f.Transactions() {
		fmt.Printf("%v %12v %-40v %8.2f\n", tx.TransactionDate.Format("02.01.2006"), tx.Transaction, tx.PayeePayer, tx.Amount)
	}
}

var transactionPattern = regexp.MustCompile(`^(?P<Date>\d+\.(\d+).) +(?P<InterestDate>\d+\.(\d+).) +(?P<Transaction>[^ ]{12}) +(?P<Payee>.*[^ ]) +(?P<Amount>\d+\.\d\d-?) *$`)
var accountPattern = regexp.MustCompile(`^(?P<Account>\d{16}/[A-ZÅÄÖ ]*[A-ZÅÄÖ]) *$`)
var totalPattern = regexp.MustCompile(`^ *LASKUN LOPPUSALDO YHTEENSÄ *(?P<DueDate>\d\d\.\d\d\.\d\d) *(?P<BillAmount>\d+\.\d\d) *$`)

type bill struct {
	file         string
	account      string
	transactions []*database.Transaction
}

func (b bill) FileName() string                      { return filepath.Base(b.file) }
func (b bill) Account() string                       { return b.account }
func (b bill) Transactions() []*database.Transaction { return b.transactions }

// FromFile loads transaction records from a Nordea TSV file.
func FromFile(filename string) (datasource.File, error) {
	lines, err := extractText(filename)
	if err != nil {
		return nil, err
	}
	var account string
	var total, sum float64
	var transactions []*database.Transaction
	var due time.Time
	for _, line := range lines {
		match := accountPattern.FindStringSubmatch(line)
		if match != nil {
			account = match[1]
			account = "************" + account[12:]
			continue
		}

		match = totalPattern.FindStringSubmatch(line)
		if match != nil {
			var err error
			total, err = strconv.ParseFloat(match[2], 64)
			if err != nil {
				return nil, err
			}
			due, err = time.Parse("02.01.06", match[1])
			if err != nil {
				return nil, err
			}

			continue
		}

		match = transactionPattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		result := make(map[string]string)
		for i, name := range transactionPattern.SubexpNames() {
			result[name] = match[i]
		}
		if strings.HasSuffix(result["Amount"], "-") {
			result["Amount"] = "-" + result["Amount"][:len(result["Amount"])-1]
		}
		amount, err := strconv.ParseFloat(result["Amount"], 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse amount %q on line %q", result["Amount"], line)
		}

		txdate, err := time.Parse("02.01.", result["Date"])
		if err != nil {
			return nil, err
		}
		txdate = txdate.AddDate(due.Year(), 0, 0)
		valdate, err := time.Parse("02.01.", result["InterestDate"])
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, &database.Transaction{
			TransactionDate: txdate,
			ValueDate:       valdate,
			Transaction:     result["Transaction"],
			PayeePayer:      result["Payee"],
			Amount:          amount,
		})
		sum += amount
	}
	for _, tx := range transactions {
		// fixYear fixes timestamp t be in the year window before reference
		fixYear := func(t time.Time, reference time.Time) time.Time {
			t = t.AddDate(reference.Year()-t.Year(), 0, 0)
			if t.After(reference) {
				t = t.AddDate(-1, 0, 0)
			}
			return t
		}
		tx.TransactionDate = fixYear(tx.TransactionDate, due)
		tx.ValueDate = fixYear(tx.ValueDate, due)
	}
	sum = math.Trunc(sum*100) / 100
	if total != sum {
		return nil, fmt.Errorf("sum of transaction amounts %.2f != bill total %.2f", sum, total)
	}
	return bill{account: account, file: filename, transactions: transactions}, nil
}

func extractText(file string) ([]string, error) {
	reader, err := pdf.Open(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %v", err)
	}

	var lines []string
	for i := 1; i < reader.NumPage()+1; i++ {
		lines = append(lines, fmt.Sprintf("Page %d", i))
		page := reader.Page(i)

		data := make(map[float64][]pdf.Text)

		for _, t := range page.Content().Text {
			data[t.Y] = append(data[t.Y], t)
		}

		// Find text lines that have text starting at position where
		// we know transactions have text data.
		lineItemLines := make(map[float64]bool)
		for _, texts := range data {
			for _, t := range texts {
				if t.X == 44.4 {
					lineItemLines[t.Y] = true
				}
			}
		}
		var sortedLines []float64
		for line := range lineItemLines {
			sortedLines = append(sortedLines, line)
		}
		sort.Sort(sort.Reverse(sort.Float64Slice(sortedLines)))

		for _, line := range sortedLines {
			var s string
			for _, l := range data[line] {
				s += l.S
			}
			lines = append(lines, s)
		}
	}
	return lines, nil
}
