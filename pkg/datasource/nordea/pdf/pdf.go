package pdf

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joneskoo/mymonies/pkg/datasource"

	"github.com/joneskoo/mymonies/pkg/rpc/mymonies"
	"rsc.io/pdf"
)

// FromFile loads Transaction records from a Nordea TSV file.
func FromFile(filename string) (datasource.File, error) {
	var (
		lines []string
		err   error
	)
	lines, err = extractText(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to extract text from PDF: %v", err)
	}
	account, transactions, err := parseLines(lines)
	if err != nil {
		return nil, err
	}
	return &bill{file: filename, account: account, transactions: transactions}, nil
}

func parseLines(lines []string) (account string, transactions []*mymonies.Transaction, err error) {
	var billTotal float64
	var dueDate time.Time
	for _, line := range lines {
		switch {
		case accountPattern.MatchString(line):
			match := accountPattern.FindStringSubmatch(line)
			account = match[1]
		case billTotalPattern.MatchString(line):
			match := billTotalPattern.FindStringSubmatch(line)
			due, _ := match[1], match[2]
			dueDate, err = time.Parse("02.01.06", due)
			if err != nil {
				return "", nil, err
			}
		case paymentsTotalPattern.MatchString(line):
			match := paymentsTotalPattern.FindStringSubmatch(line)
			total := match[1]
			billTotal, err = parseAmount(total)
			if err != nil {
				return "", nil, err
			}
		}
	}
	if account == "" {
		return "", nil, fmt.Errorf("could not find account number header from file")
	}

	p := new(safeParser)
	transactions = make([]*mymonies.Transaction, 0)
	for _, line := range lines {
		if transactionPattern.MatchString(line) {
			match := transactionPattern.FindStringSubmatch(line)
			transactions = append(transactions, &mymonies.Transaction{
				TransactionDate: date(fixYear(p.date(match[1], "transaction date"), dueDate)),
				ValueDate:       date(fixYear(p.date(match[2], "interest date"), dueDate)),
				Transaction:     match[3],
				PayeePayer:      match[4],
				Amount:          p.amount(match[5], "amount"),
			})
		}
	}
	if p.err != nil {
		return "", nil, p.err
	}
	sum := 0.0
	for _, tx := range transactions {
		sum += tx.Amount
	}
	if math.Abs(billTotal-sum) > 0.009 {
		return "", nil, fmt.Errorf("transaction amounts (%.2f) != bill total (%.2f)", sum, billTotal)
	}
	return account, transactions, nil
}

var (
	accountPattern       = regexp.MustCompile(`^(?P<Account>\d{16}/[A-ZÅÄÖ ]*[A-ZÅÄÖ]) *$`)
	billTotalPattern     = regexp.MustCompile(`^ *LASKUN LOPPUSALDO YHTEENSÄ *(?P<DueDate>\d\d\.\d\d\.\d\d) *(?P<BillTotal>[\d ]+\.\d\d) *$`)
	paymentsTotalPattern = regexp.MustCompile(`^ *KORTTITAPAHTUMAT YHTEENSÄ *(?P<PaymentsTotal>[\d ]+\.\d\d) *$`)
	transactionPattern   = regexp.MustCompile(`^(?P<Date>\d+\.\d+\.) +(?P<InterestDate>\d+\.\d+\.) +(?P<Transaction>[^ ]{12}) +(?P<Payee>.*[^ ]) +(?P<Amount>\d+\.\d\d-?) *$`)
)

type bill struct {
	file         string
	account      string
	transactions []*mymonies.Transaction
}

func (b bill) FileName() string { return filepath.Base(b.file) }
func (b bill) Account() string {
	account := "************" + b.account[12:]
	return account
}
func (b bill) Transactions() []*mymonies.Transaction { return b.transactions }

type safeParser struct {
	err error
}

func (p *safeParser) date(v, field string) (t time.Time) {
	if p.err != nil {
		return
	}
	t, p.err = time.Parse("02.01.", v)
	if p.err != nil {
		p.err = fmt.Errorf("bad %v format: %v", field, p.err)
	}
	return
}

func (p *safeParser) amount(v, field string) (a float64) {
	if p.err != nil {
		return
	}
	a, p.err = parseAmount(v)
	if p.err != nil {
		p.err = fmt.Errorf("bad %v format: %v", field, p.err)
	}
	return
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

func parseAmount(amount string) (float64, error) {
	amount = strings.Replace(amount, " ", "", -1)
	if strings.HasSuffix(amount, "-") {
		amount = "-" + amount[:len(amount)-1]
	}
	res, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return 0.0, err
	}
	return res, nil
}

// fixYear fixes timestamp t be in the year window before reference
func fixYear(t time.Time, reference time.Time) time.Time {
	t = t.AddDate(reference.Year()-t.Year(), 0, 0)
	if t.After(reference) {
		t = t.AddDate(-1, 0, 0)
	}
	return t
}

func date(t time.Time) string {
	return t.UTC().Truncate(24 * time.Hour).Format(time.RFC3339)
}
