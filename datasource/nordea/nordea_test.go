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

func TestFromTsv(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name     string
		args     args
		wantFile File
		wantErr  bool
	}{
		{"missing file", args{}, File{}, true},
		{"invalid format", args{"test_data/invalid.txt"}, File{}, true},

		{
			"valid file",
			args{"test_data/Tapahtumat_FI4612345600007890_20130808_20130808.txt"},
			File{
				Account: "FI4612345600007890",
				Records: []*database.Record{
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFile, err := FromTsv(tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromTsv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotFile.Account, tt.wantFile.Account) {
				t.Errorf("FromTsv() Account = %v, want %v", gotFile.Account, tt.wantFile.Account)
			}
			if !reflect.DeepEqual(gotFile.Records, tt.wantFile.Records) {
				t.Errorf("FromTsv() = %+#v, want %+#v", gotFile.Records, tt.wantFile.Records)
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
