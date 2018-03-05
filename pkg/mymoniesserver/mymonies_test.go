package mymoniesserver

import (
	"context"
	"flag"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/joneskoo/mymonies/pkg/mymoniesserver/database"
	pb "github.com/joneskoo/mymonies/pkg/rpc/mymonies"
)

var testDatabaseConn string

func init() {
	flag.StringVar(&testDatabaseConn, "mymonies-test-db", "database=mymonies_test", "Database to use for mymonies unit tests")
	flag.Parse()
}

func newServer(t *testing.T, fixtures ...fixture) *server {
	db, err := database.Connect(testDatabaseConn)
	if err != nil {
		t.Fatal("connect to test database failed:", err)
	}
	if err := db.DropTables(); err != nil {
		t.Fatal("db.DropTables() returned error:", err)
	}
	if err := db.CreateTables(); err != nil {
		t.Fatal("db.CreateTables() returned error:", err)
	}
	for _, f := range fixtures {
		if _, err := db.Exec(string(f)); err != nil {
			t.Fatal(err)
		}
	}
	logger := make(mockLogger, 0)
	return &server{DB: db, logger: logger}
}

type fixture string

var (
	fixtureTags fixture = `
		INSERT INTO tags (name) VALUES ('example');
		INSERT INTO tags (name) VALUES ('example2');
	`
	fixtureTransactions fixture = `
		INSERT INTO imports (filename, account) VALUES ('asdf', 'foo');
		INSERT INTO records (import_id, transaction_date, value_date, payment_date, amount) VALUES (1, '2018-03-01'::date, '2018-03-02'::date, '2018-03-03'::date, 10);
	`
)

type mockLogger []string

func (l mockLogger) Println(args ...interface{}) {
	l = append(l, fmt.Sprint(args...))
}

func Test_server_AddImport(t *testing.T) {
	tests := []struct {
		name    string
		req     *pb.AddImportReq
		want    *pb.AddImportResp
		wantErr bool
	}{
		{
			name: "valid",
			req: &pb.AddImportReq{
				Account:  "example",
				FileName: "data.txt",
				Transactions: []*pb.Transaction{
					&pb.Transaction{
						Amount:          10.0,
						PaymentDate:     today,
						TransactionDate: today,
						ValueDate:       today,
					},
				},
			},
			want: &pb.AddImportResp{},
		},
		{
			name: "missing-account",
			req: &pb.AddImportReq{
				Account:  "",
				FileName: "data.txt",
				Transactions: []*pb.Transaction{
					&pb.Transaction{
						Amount:          10.0,
						PaymentDate:     today,
						TransactionDate: today,
						ValueDate:       today,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing-file-name",
			req: &pb.AddImportReq{
				Account:  "example",
				FileName: "",
				Transactions: []*pb.Transaction{
					&pb.Transaction{
						Amount:          10.0,
						PaymentDate:     today,
						TransactionDate: today,
						ValueDate:       today,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing-transactions",
			req: &pb.AddImportReq{
				Account:      "example",
				FileName:     "foo.txt",
				Transactions: []*pb.Transaction{},
			},
			wantErr: true,
		},
		{
			name: "invalid-payment-date",
			req: &pb.AddImportReq{
				Account:  "example",
				FileName: "data.txt",
				Transactions: []*pb.Transaction{
					&pb.Transaction{
						Amount:          10.0,
						PaymentDate:     "1.2.2015",
						TransactionDate: today,
						ValueDate:       today,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "value-date-time-not-zero",
			req: &pb.AddImportReq{
				Account:      "example",
				FileName:     "data.txt",
				Transactions: []*pb.Transaction{exampleTransaction},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t)
			got, err := s.AddImport(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("server.AddImport() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("server.AddImport() = %v, want %v", got, tt.want)
			}
		})
	}
}

var (
	today              = date(time.Now())
	exampleTransaction = &pb.Transaction{
		Amount:          10.0,
		PaymentDate:     today,
		TransactionDate: "1985-04-12T23:20:50.52Z",
		ValueDate:       today,
	}
)

func date(t time.Time) string {
	return t.UTC().Truncate(24 * time.Hour).Format(time.RFC3339)
}

func Test_server_AddPattern(t *testing.T) {
	tests := []struct {
		name    string
		req     *pb.AddPatternReq
		want    *pb.AddPatternResp
		wantErr bool
	}{
		{
			name: "valid",
			req: &pb.AddPatternReq{
				Pattern: &pb.Pattern{
					Account: "example",
					Query:   "",
					TagId:   "1",
				},
			},
			want: &pb.AddPatternResp{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t, fixtureTags)
			got, err := s.AddPattern(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("server.AddPattern() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("server.AddPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_server_ListAccounts(t *testing.T) {
	tests := []struct {
		name    string
		req     *pb.ListAccountsReq
		want    *pb.ListAccountsResp
		wantErr bool
	}{
		{
			name: "valid",
			req:  &pb.ListAccountsReq{},
			want: &pb.ListAccountsResp{
				Accounts: []*pb.Account{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t)
			got, err := s.ListAccounts(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("server.ListAccounts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("server.ListAccounts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_server_ListTags(t *testing.T) {
	tests := []struct {
		name    string
		req     *pb.ListTagsReq
		want    *pb.ListTagsResp
		wantErr bool
	}{
		{
			name: "valid",
			req:  &pb.ListTagsReq{},
			want: &pb.ListTagsResp{Tags: []*pb.Tag{
				&pb.Tag{Id: "1", Name: "example"},
				&pb.Tag{Id: "2", Name: "example2"},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t, fixtureTags)
			got, err := s.ListTags(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("server.ListTags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("server.ListTags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_server_ListTransactions(t *testing.T) {
	tests := []struct {
		name    string
		req     *pb.ListTransactionsReq
		want    *pb.ListTransactionsResp
		wantErr bool
	}{
		{
			name: "valid all transactions",
			req:  &pb.ListTransactionsReq{Filter: &pb.TransactionFilter{}},
			want: &pb.ListTransactionsResp{
				Transactions: []*pb.Transaction{
					{
						Id:              "1",
						Amount:          10,
						ImportId:        "1",
						TransactionDate: "2018-03-01T00:00:00Z",
						ValueDate:       "2018-03-02T00:00:00Z",
						PaymentDate:     "2018-03-03T00:00:00Z",
					},
				},
			},
		},
		{
			name:    "missing-filter",
			req:     &pb.ListTransactionsReq{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t, fixtureTransactions)
			got, err := s.ListTransactions(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("server.ListTransactions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("server.ListTransactions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_server_UpdateTag(t *testing.T) {
	tests := []struct {
		name    string
		req     *pb.UpdateTagReq
		want    *pb.UpdateTagResp
		wantErr bool
	}{
		{
			name: "valid",
			req:  &pb.UpdateTagReq{TagId: "1", TransactionId: "1"},
			want: &pb.UpdateTagResp{},
		},
		{
			name:    "missing transaction id",
			req:     &pb.UpdateTagReq{TagId: "1"},
			wantErr: true,
		},
		{
			name:    "malformed tag id",
			req:     &pb.UpdateTagReq{TagId: "x", TransactionId: "1"},
			wantErr: true,
		},
		{
			name:    "malformed transaction id",
			req:     &pb.UpdateTagReq{TagId: "1", TransactionId: "x"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t, fixtureTags, fixtureTransactions)
			got, err := s.UpdateTag(context.Background(), tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("server.UpdateTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("server.UpdateTag() = %v, want %v", got, tt.want)
			}
		})
	}
}
