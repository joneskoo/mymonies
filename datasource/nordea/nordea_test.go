package nordea

import (
	"reflect"
	"testing"
	"time"

	"github.com/joneskoo/mymonies/database"
)

func mustParseRFC3339(value string) time.Time {
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return ts.In(helsinki)
}

func TestFromFile(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name     string
		args     args
		wantFile file
		wantErr  bool
	}{
		{"missing file", args{""}, file{}, true},
		{"invalid format", args{"test_data/invalid.txt"}, file{}, true},

		{
			"valid file",
			args{"test_data/Tapahtumat_FI4612345600007890_20130808_20130808.txt"},
			file{
				filename: "Tapahtumat_FI4612345600007890_20130808_20130808.txt",
				account:  "FI4612345600007890",
				transactions: []*database.Record{
					&database.Record{
						TransactionDate: mustParseRFC3339("2015-03-23T00:00:00+02:00"),
						ValueDate:       mustParseRFC3339("2015-03-22T00:00:00+02:00"),
						PaymentDate:     mustParseRFC3339("2015-03-22T00:00:00+02:00"),
						Amount:          -30.00,
						PayeePayer:      "Payee ry",
						Account:         "FI1012345600007890",
						BIC:             "ASDFFIHHXXX",
						Transaction:     "Itsepalvelu",
						Reference:       "1 27650",
						PayerReference:  "",
						Message:         "",
						CardNumber:      "",
					},
				},
			},
			false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(tt *testing.T) {
			gotFile, err := FromFile(tc.args.filename)
			if err != nil && !tc.wantErr {
				tt.Fatalf("FromFile() error = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil {
				return
			}
			gotNordea := gotFile.(file)
			if gotNordea.account != tc.wantFile.account {
				tt.Fatalf("FromFile() account = %v, want %v", gotNordea.account, tc.wantFile.account)
			}
			if !reflect.DeepEqual(gotNordea.transactions, tc.wantFile.transactions) {
				tt.Fatalf("FromFile() = %+#v, want %+#v", gotNordea.transactions, tc.wantFile.transactions)
			}
		})
	}
}

func Test_fromSlice(t *testing.T) {
	type args struct {
		r []string
	}
	tests := []struct {
		name    string
		args    args
		wantRec database.Record
		wantErr bool
	}{
		{
			"valid",
			args{[]string{"23.03.2015", "22.03.2015", "22.03.2015", "-30,00", "Payee ry", "FI1012345600007890", "ASDFFIHHXXX", "Itsepalvelu", "1 27650", "", "", "", "", ""}},
			database.Record{
				TransactionDate: mustParseRFC3339("2015-03-23T00:00:00+02:00"),
				ValueDate:       mustParseRFC3339("2015-03-22T00:00:00+02:00"),
				PaymentDate:     mustParseRFC3339("2015-03-22T00:00:00+02:00"),
				Amount:          -30.00,
				PayeePayer:      "Payee ry",
				Account:         "FI1012345600007890",
				BIC:             "ASDFFIHHXXX",
				Transaction:     "Itsepalvelu",
				Reference:       "1 27650",
				PayerReference:  "",
				Message:         "",
				CardNumber:      "",
			},
			false,
		},

		// TODO: implement parsing for messages over 35 characters
		// {
		// 	"long message",
		// 	args{[]string{"07.12.2016", "07.12.2016", "05.12.2016", "50,00", "EXAMPLE PERSON NAME", "", "", "Pano", "", "", "Merry xmas and happy new year to yo u and your family. ", "", "", ""}},
		// 	Record{
		// 		TransactionDate: mustParseRFC3339("2016-12-07T00:00:00+02:00"),
		// 		ValueDate:       mustParseRFC3339("2016-12-07T00:00:00+02:00"),
		// 		PaymentDate:     mustParseRFC3339("2016-12-05T00:00:00+02:00"),
		// 		Amount:          50,
		// 		PayeePayer:      "EXAMPLE PERSON NAME",
		// 		Account:         "",
		// 		BIC:             "",
		// 		Transaction:     "Pano",
		// 		Reference:       "",
		// 		PayerReference:  "",
		// 		Message:         "Merry xmas and happy new year to you and your family.",
		// 		CardNumber:      "",
		// 		Receipt:         "",
		// 	},
		// 	false,
		// },

		{
			"missing transaction date",
			args{[]string{"", "22.03.2015", "22.03.2015", "-30,00", "Payee ry", "FI1012345600007890", "ASDFFIHHXXX", "Itsepalvelu", "1 27650", "", "", "", "", ""}},
			database.Record{},
			true,
		},
		{
			"bad value date format",
			args{[]string{"22.03.2015", "22.3.2015", "22.03.2015", "-30,00", "Payee ry", "FI1012345600007890", "ASDFFIHHXXX", "Itsepalvelu", "1 27650", "", "", "", "", ""}},
			database.Record{},
			true,
		},
		{
			"bad payment date format",
			args{[]string{"22.03.2015", "22.03.2015", "22.13.2015", "-30,00", "Payee ry", "FI1012345600007890", "ASDFFIHHXXX", "Itsepalvelu", "1 27650", "", "", "", "", ""}},
			database.Record{},
			true,
		},
		{
			"invalid amount",
			args{[]string{"", "22.03.2015", "22.03.2015", "invalid", "Payee ry", "FI1012345600007890", "ASDFFIHHXXX", "Itsepalvelu", "1 27650", "", "", "", "", ""}},
			database.Record{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRec, err := fromSlice(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("fromSlice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotRec, tt.wantRec) {
				t.Errorf("fromSlice() = %v, want %v", gotRec, tt.wantRec)
			}
		})
	}
}
