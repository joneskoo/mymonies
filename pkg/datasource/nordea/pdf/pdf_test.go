package pdf

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/joneskoo/mymonies/pkg/datasource"
	"github.com/joneskoo/mymonies/pkg/rpc/mymonies"
)

func TestFromFile(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    datasource.File
		wantErr bool
	}{
		{
			name:    "empty",
			args:    args{filepath.Join("testdata", "empty.pdf")},
			wantErr: true,
		},
		// Commented out because I'm not going to publish my real bills.
		// {
		// 	name: "real example pdf bill",
		// 	args: args{"/tmp/bill.pdf"},
		// 	want: &bill{
		// 		account:      "asdf",
		// 		file:         "/tmp/bill.pdf",
		// 		transactions: nil,
		// 	},
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromFile(tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FromFile() = %v, want %v", got, tt.want)
				for _, t := range got.Transactions() {
					fmt.Printf("%#v\n", t)
				}
			}
		})
	}
}

func Test_parseLines(t *testing.T) {
	type args struct {
		lines []string
	}
	tests := []struct {
		name             string
		args             args
		wantAccount      string
		wantTransactions []*mymonies.Transaction
		wantErr          bool
	}{
		{
			name: "valid extracted text",
			args: args{[]string{
				"Page 1",
				"10.11.                           EDELLISEN LASKUN LOPPUSALDO           123.45             0.00            123.45      ",
				"                                                                                                                      ",
				"30.11. 30.11.  0123456789123456  SUORITUS                              123.45-                            123.45-     ",
				"1234567890123456/HOLDER CARD                                                                                          ",
				"10.11. 02.01.  012765012765      HESBURGER                                                                 13.37      ",
				"19.11. 02.01.  133713371337      IKEA - HAPARANDA                                                          14.29      ",
				"               SEK        138.000                                                                                     ",
				"               KURSSI      9.6571                                                                                     ",
				"02.12. 02.01.  101010010010      ITUNES.COM/BILL          /HYVITYS                                          4.99-     ",
				"                                 KORTTITAPAHTUMAT YHTEENSÄ              22.67                                         ",
				"                                                                                   -------------------------------    ",
				"                                                                                          0.00             22.67      ",
				"                                                                                   ===============================    ",
				"                                 LASKUN LOPPUSALDO YHTEENSÄ      02.01.17                                  22.67      ",
				"                                 VALITSEMANNE LYHENNYSVAPAAT KUUKAUDET -  JA  -                                       ",
				"                                                                                                                      ",
				"Page 2",
				"EURIBOR 3 ON MUUTTUNUT. SOVELLAMME UUTTA KORKOA TÄSTÄ                                                                 ",
				"LASKUSTA ALKAEN.                                                                                                      ",
				"                                                                                                                      ",
				"Sopiva eräpäivä                                                                                                       ",
				"                                                                                                                      ",
				"Lasku erääntyy maksettavaksi valittuna eräpäivänä kuukausittain. Voit halutessa-                                      ",
				"si muuttaa eräpäivän (valinnainen 1-31 päivä) ePalvelussa,Nordean verkkopankissa                                      ",
				"tai soittamalla asiakaspalveluun. Eräpäivän muutoksesta ei veloiteta kulua.                                           ",
			}},
			wantAccount: "1234567890123456/HOLDER CARD",
			wantTransactions: []*mymonies.Transaction{
				&mymonies.Transaction{TransactionDate: "2016-11-10T00:00:00Z", ValueDate: "2017-01-02T00:00:00Z", Amount: 13.37, PayeePayer: "HESBURGER", Transaction: "012765012765"},
				&mymonies.Transaction{TransactionDate: "2016-11-19T00:00:00Z", ValueDate: "2017-01-02T00:00:00Z", Amount: 14.29, PayeePayer: "IKEA - HAPARANDA", Transaction: "133713371337"},
				&mymonies.Transaction{TransactionDate: "2016-12-02T00:00:00Z", ValueDate: "2017-01-02T00:00:00Z", Amount: -4.99, PayeePayer: "ITUNES.COM/BILL          /HYVITYS", Transaction: "101010010010"},
			},
		},

		{
			name: "transaction sum not equal bill total",
			args: args{[]string{
				"Page 1",
				"10.11.                           EDELLISEN LASKUN LOPPUSALDO           123.45             0.00            123.45      ",
				"                                                                                                                      ",
				"30.11. 30.11.  0123456789123456  SUORITUS                              123.45-                            123.45-     ",
				"1234567890123456/HOLDER CARD                                                                                          ",
				"10.11. 02.01.  012765012765      HESBURGER                                                                 13.38      ",
				"19.11. 02.01.  133713371337      IKEA - HAPARANDA                                                          14.29      ",
				"               SEK        138.000                                                                                     ",
				"               KURSSI      9.6571                                                                                     ",
				"02.12. 02.01.  101010010010      ITUNES.COM/BILL          /HYVITYS                                          4.99-     ",
				"                                 KORTTITAPAHTUMAT YHTEENSÄ              22.67                                         ",
				"                                                                                   -------------------------------    ",
				"                                                                                          0.00             22.67      ",
				"                                                                                   ===============================    ",
				"                                 LASKUN LOPPUSALDO YHTEENSÄ      02.01.17                                  22.67      ",
				"                                 VALITSEMANNE LYHENNYSVAPAAT KUUKAUDET -  JA  -                                       ",
				"                                                                                                                      ",
				"Page 2",
				"EURIBOR 3 ON MUUTTUNUT. SOVELLAMME UUTTA KORKOA TÄSTÄ                                                                 ",
				"LASKUSTA ALKAEN.                                                                                                      ",
				"                                                                                                                      ",
				"Sopiva eräpäivä                                                                                                       ",
				"                                                                                                                      ",
				"Lasku erääntyy maksettavaksi valittuna eräpäivänä kuukausittain. Voit halutessa-                                      ",
				"si muuttaa eräpäivän (valinnainen 1-31 päivä) ePalvelussa,Nordean verkkopankissa                                      ",
				"tai soittamalla asiakaspalveluun. Eräpäivän muutoksesta ei veloiteta kulua.                                           ",
			}},
			wantErr: true,
		},

		{
			name: "minimal",
			args: args{[]string{
				"Page 1",
				"1234567890123456/HOLDER CARD                                                                                          ",
				"10.11. 02.01.  012765012765      HESBURGER                                                                 13.38      ",
				"                                 KORTTITAPAHTUMAT YHTEENSÄ              13.38                                         ",
				"                                                                                   -------------------------------    ",
				"                                                                                          0.00             13.38      ",
				"                                                                                   ===============================    ",
				"                                 LASKUN LOPPUSALDO YHTEENSÄ      02.01.17                                  13.38      ",
			}},
			wantAccount: "1234567890123456/HOLDER CARD",
			wantTransactions: []*mymonies.Transaction{
				&mymonies.Transaction{TransactionDate: "2016-11-10T00:00:00Z", ValueDate: "2017-01-02T00:00:00Z", Amount: 13.38, PayeePayer: "HESBURGER", Transaction: "012765012765"},
			},
		},

		{
			name: "malformed payment date",
			args: args{[]string{
				"Page 1",
				"1234567890123456/HOLDER CARD                                                                                          ",
				"10.13. 02.01.  012765012765      HESBURGER                                                                 13.38      ",
				"                                 KORTTITAPAHTUMAT YHTEENSÄ              13.38                                         ",
				"                                                                                   -------------------------------    ",
				"                                                                                          0.00             13.38      ",
				"                                                                                   ===============================    ",
				"                                 LASKUN LOPPUSALDO YHTEENSÄ      02.01.17                                  13.38      ",
			}},
			wantErr: true,
		},

		{
			name: "malformed value date",
			args: args{[]string{
				"Page 1",
				"1234567890123456/HOLDER CARD                                                                                          ",
				"10.11. 00.01.  012765012765      HESBURGER                                                                 13.38      ",
				"                                 KORTTITAPAHTUMAT YHTEENSÄ              13.38                                         ",
				"                                                                                   -------------------------------    ",
				"                                                                                          0.00             13.38      ",
				"                                                                                   ===============================    ",
				"                                 LASKUN LOPPUSALDO YHTEENSÄ      02.01.17                                  13.38      ",
			}},
			wantErr: true,
		},

		// malformed amount is hard to test because regexp is strict about format.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAccount, gotTransactions, err := parseLines(tt.args.lines)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotAccount != tt.wantAccount {
				t.Errorf("parseLines() gotAccount = %v, want %v", gotAccount, tt.wantAccount)
			}
			if !reflect.DeepEqual(gotTransactions, tt.wantTransactions) {
				t.Errorf("parseLines() gotTransactions = %v, want %v", gotTransactions, tt.wantTransactions)
			}
		})
	}
}

func Test_safeParser_amount(t *testing.T) {
	type fields struct {
		err error
	}
	type args struct {
		v     string
		field string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantA   float64
		wantErr bool
	}{
		{
			name:  "valid",
			args:  args{v: "10.00"},
			wantA: 10.0,
		},

		{
			name:    "malformed amount",
			args:    args{v: "10.a"},
			wantA:   0.0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &safeParser{
				err: tt.fields.err,
			}
			if gotA := p.amount(tt.args.v, tt.args.field); gotA != tt.wantA {
				t.Errorf("safeParser.amount() = %v, want %v", gotA, tt.wantA)
			}
			if (p.err != nil) != tt.wantErr {
				t.Errorf("safeParser.amount() set err = %v, wantErr %v", p.err, tt.wantErr)
			}

		})
	}
}

func Test_extractText(t *testing.T) {
	type args struct {
		file string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "empty pdf file",
			args: args{filepath.Join("testdata", "empty.pdf")},
			want: []string{"Page 1"},
		},

		{
			name:    "corrupted pdf file",
			args:    args{filepath.Join("testdata", "corrupted.pdf")},
			wantErr: true,
		},
		// I'm not going to include an actual PDF bill and I don't have
		// a sample file of a credit card bill without personal data.
		// Sorry.
		// {
		// 	name: "real example pdf bill",
		// 	args: args{"/tmp/bill.pdf")},
		// 	want: []string{"Page 1"},
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractText(tt.args.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_parseAmount(t *testing.T) {
	type args struct {
		amount string
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{
			name: "valid",
			args: args{"10.00"},
			want: 10.0,
		},

		{
			name: "negative",
			args: args{"10.00-"},
			want: -10.0,
		},

		{
			name:    "malformed amount",
			args:    args{"10.a"},
			want:    0.0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAmount(tt.args.amount)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAmount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseAmount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fixYear(t *testing.T) {
	type args struct {
		t         time.Time
		reference time.Time
	}
	tests := []struct {
		name string
		args args
		want time.Time
	}{
		{
			name: "same day",
			args: args{
				t:         time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
				reference: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			want: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "dec 31st day before",
			args: args{
				t:         time.Date(1, 12, 31, 0, 0, 0, 0, time.UTC),
				reference: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			want: time.Date(2017, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "next day",
			args: args{
				t:         time.Date(1, 1, 2, 0, 0, 0, 0, time.UTC),
				reference: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			want: time.Date(2017, 1, 2, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fixYear(tt.args.t, tt.args.reference); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fixYear() = %v, want %v", got, tt.want)
			}
		})
	}
}
