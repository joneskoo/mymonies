package nordea

import (
	"reflect"
	"testing"
	"time"
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
				Records: []Record{
					Record{
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
						Receipt:         "",
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

func TestRecord_String(t *testing.T) {
	type fields struct {
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
		Receipt         string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			"example record",
			fields{
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
				Receipt:         "",
			},
			"23.03.2015 Payee ry                         -30.00",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Record{
				TransactionDate: tt.fields.TransactionDate,
				ValueDate:       tt.fields.ValueDate,
				PaymentDate:     tt.fields.PaymentDate,
				Amount:          tt.fields.Amount,
				PayeePayer:      tt.fields.PayeePayer,
				Account:         tt.fields.Account,
				BIC:             tt.fields.BIC,
				Transaction:     tt.fields.Transaction,
				Reference:       tt.fields.Reference,
				PayerReference:  tt.fields.PayerReference,
				Message:         tt.fields.Message,
				CardNumber:      tt.fields.CardNumber,
				Receipt:         tt.fields.Receipt,
			}
			if got := r.String(); got != tt.want {
				t.Errorf("Record.String() = %v, want %v", got, tt.want)
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
		wantRec Record
		wantErr bool
	}{
		{
			"valid",
			args{[]string{"23.03.2015", "22.03.2015", "22.03.2015", "-30,00", "Payee ry", "FI1012345600007890", "ASDFFIHHXXX", "Itsepalvelu", "1 27650", "", "", "", "", ""}},
			Record{
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
				Receipt:         "",
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
			Record{},
			true,
		},
		{
			"bad value date format",
			args{[]string{"22.03.2015", "22.3.2015", "22.03.2015", "-30,00", "Payee ry", "FI1012345600007890", "ASDFFIHHXXX", "Itsepalvelu", "1 27650", "", "", "", "", ""}},
			Record{},
			true,
		},
		{
			"bad payment date format",
			args{[]string{"22.03.2015", "22.03.2015", "22.13.2015", "-30,00", "Payee ry", "FI1012345600007890", "ASDFFIHHXXX", "Itsepalvelu", "1 27650", "", "", "", "", ""}},
			Record{},
			true,
		},
		{
			"invalid amount",
			args{[]string{"", "22.03.2015", "22.03.2015", "invalid", "Payee ry", "FI1012345600007890", "ASDFFIHHXXX", "Itsepalvelu", "1 27650", "", "", "", "", ""}},
			Record{},
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
