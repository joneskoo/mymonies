package tsv

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"

	pb "github.com/joneskoo/mymonies/pkg/rpc/mymonies"
)

func mustParseRFC3339(value string) string {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return t.UTC().Truncate(24 * time.Hour).Format(time.RFC3339)
}

func TestFromFile(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name     string
		args     args
		wantFile File
		wantErr  bool
	}{
		{"missing file", args{""}, File{}, true},
		{"invalid format", args{filepath.Join("testdata", "invalid.txt")}, File{}, true},

		{
			"valid file",
			args{filepath.Join("testdata", "Tapahtumat_FI4612345600007890_20130808_20130808.txt")},
			File{
				filename: "Tapahtumat_FI4612345600007890_20130808_20130808.txt",
				account:  "FI4612345600007890",
				transactions: []*pb.Transaction{
					&pb.Transaction{
						TransactionDate: mustParseRFC3339("2015-03-23T00:00:00+02:00"),
						ValueDate:       mustParseRFC3339("2015-03-22T00:00:00+02:00"),
						PaymentDate:     mustParseRFC3339("2015-03-22T00:00:00+02:00"),
						Amount:          -30.00,
						PayeePayer:      "Payee ry",
						Account:         "FI1012345600007890",
						Bic:             "ASDFFIHHXXX",
						Transaction:     "Itsepalvelu",
						Reference:       "1 27650",
					},
				},
			},
			false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(tt *testing.T) {
			got, err := FromFile(tc.args.filename)
			if err != nil && !tc.wantErr {
				tt.Fatalf("FromFile() error = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil {
				return
			}
			if got.Account() != tc.wantFile.account {
				tt.Fatalf("FromFile() account = %v, want %v", got.Account(), tc.wantFile.account)
			}
			if !reflect.DeepEqual(got.Transactions(), tc.wantFile.transactions) {
				tt.Fatalf("FromFile() = %+#v, want %+#v", got.Transactions(), tc.wantFile.transactions)
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
		wantRec *pb.Transaction
		wantErr bool
	}{
		{
			"valid",
			args{[]string{"23.03.2015", "22.03.2015", "22.03.2015", "-30,00", "Payee ry", "FI1012345600007890", "ASDFFIHHXXX", "Itsepalvelu", "1 27650", "", "", "", "", ""}},
			&pb.Transaction{
				TransactionDate: mustParseRFC3339("2015-03-23T00:00:00+02:00"),
				ValueDate:       mustParseRFC3339("2015-03-22T00:00:00+02:00"),
				PaymentDate:     mustParseRFC3339("2015-03-22T00:00:00+02:00"),
				Amount:          -30.00,
				PayeePayer:      "Payee ry",
				Account:         "FI1012345600007890",
				Bic:             "ASDFFIHHXXX",
				Transaction:     "Itsepalvelu",
				Reference:       "1 27650",
			},
			false,
		},

		{
			"long message",
			args{[]string{"07.12.2016", "07.12.2016", "05.12.2016", "50,00", "EXAMPLE PERSON NAME", "", "", "Pano", "", "", "Merry xmas and happy new year to yo u and your family. ", "", "", ""}},
			&pb.Transaction{
				TransactionDate: mustParseRFC3339("2016-12-07T00:00:00+02:00"),
				ValueDate:       mustParseRFC3339("2016-12-07T00:00:00+02:00"),
				PaymentDate:     mustParseRFC3339("2016-12-05T00:00:00+02:00"),
				Amount:          50,
				PayeePayer:      "EXAMPLE PERSON NAME",
				Transaction:     "Pano",
				Message:         "Merry xmas and happy new year to you and your family. ",
			},
			false,
		},

		{
			"missing transaction date",
			args{[]string{"", "22.03.2015", "22.03.2015", "-30,00", "Payee ry", "FI1012345600007890", "ASDFFIHHXXX", "Itsepalvelu", "1 27650", "", "", "", "", ""}},
			&pb.Transaction{},
			true,
		},
		{
			"bad value date format",
			args{[]string{"22.03.2015", "22.3.2015", "22.03.2015", "-30,00", "Payee ry", "FI1012345600007890", "ASDFFIHHXXX", "Itsepalvelu", "1 27650", "", "", "", "", ""}},
			&pb.Transaction{},
			true,
		},
		{
			"bad payment date format",
			args{[]string{"22.03.2015", "22.03.2015", "22.13.2015", "-30,00", "Payee ry", "FI1012345600007890", "ASDFFIHHXXX", "Itsepalvelu", "1 27650", "", "", "", "", ""}},
			&pb.Transaction{},
			true,
		},
		{
			"invalid amount",
			args{[]string{"", "22.03.2015", "22.03.2015", "invalid", "Payee ry", "FI1012345600007890", "ASDFFIHHXXX", "Itsepalvelu", "1 27650", "", "", "", "", ""}},
			&pb.Transaction{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRec, err := fromSlice(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Logf("fromSlice(%v) = %v", tt.args.r, gotRec)
				t.Errorf("fromSlice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(gotRec, tt.wantRec) {
				t.Errorf("fromSlice() = %v, want %v", gotRec, tt.wantRec)
			}
		})
	}
}

func Test_joinMessage(t *testing.T) {
	type args struct {
		msg string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "extra spaces after 35",
			args: args{"Lorem ipsum dolor sit amet, consect etur adipiscing elit. Curabitur dig nissim mi enim, eget condimentum ma uris faucibus vitae."},
			want: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Curabitur dignissim mi enim, eget condimentum mauris faucibus vitae.",
		},
		{
			name: "no extra spaces",
			args: args{"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Curabitur dignissim mi enim, eget condimentum mauris faucibus vitae."},
			want: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Curabitur dignissim mi enim, eget condimentum mauris faucibus vitae.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := joinMessage(tt.args.msg); got != tt.want {
				t.Errorf("joinMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}
