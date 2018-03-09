package mymoniesserver

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"
	"time"

	"github.com/joneskoo/mymonies/pkg/mymoniesserver/database"
	pb "github.com/joneskoo/mymonies/pkg/rpc/mymonies"
)

var (
	update           bool
	testDatabaseConn string
)

func init() {
	flag.BoolVar(&update, "update", false, "update and overwrite golden test data files")
	flag.StringVar(&testDatabaseConn, "mymonies-test-db", "database=mymonies_test", "Database to use for mymonies unit tests")
	flag.Parse()
}

func readFixture(t *testing.T, f string) []byte {
	if f == "" {
		return nil
	}
	// FIXME: use filepath for cross-platform compatibility
	b, err := ioutil.ReadFile(f)
	if err != nil {
		t.Fatalf("failed to read fixture %q", f)
	}
	return b
}

func jsonFixture(t *testing.T, f string, dst interface{}) {
	b := readFixture(t, f)
	if len(b) == 0 {
		return
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	d.Decode(dst)
}

func writeFixture(t *testing.T, f string, got interface{}) {
	t.Logf("wrote updated fixture: %v", f)
	b, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("failed to encode json: %v", err)
	}
	if err := ioutil.WriteFile(f, b, 0644); err != nil {
		t.Fatalf("failed to write fixture update: %v", err)
	}
}

func newServer(t *testing.T, sqlFixture string) *server {
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
	query := string(readFixture(t, sqlFixture))
	if _, err := db.Exec(query); err != nil {
		t.Fatal(err)
	}
	logger := make(mockLogger, 0)
	return &server{DB: db, logger: logger}
}

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
			s := newServer(t, "")
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
		sql     string
		req     *pb.AddPatternReq
		want    *pb.AddPatternResp
		wantErr bool
	}{
		{
			name: "valid",
			sql:  "testdata/data.sql",
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
			s := newServer(t, tt.sql)
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
		sql     string
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
			s := newServer(t, tt.sql)
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
		sql     string
		req     *pb.ListTagsReq
		want    *pb.ListTagsResp
		wantErr bool
	}{
		{
			name: "valid",
			sql:  "testdata/data.sql",
			req:  &pb.ListTagsReq{},
			want: &pb.ListTagsResp{Tags: []*pb.Tag{
				&pb.Tag{Id: "1", Name: "example"},
				&pb.Tag{Id: "2", Name: "example2"},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t, tt.sql)
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
		sql     string
		req     string
		want    string
		wantErr bool
	}{
		{
			name: "valid all transactions",
			sql:  "testdata/list-transactions/data.sql",
			req:  "testdata/list-transactions/valid-all-transactions/req.json",
			want: "testdata/list-transactions/valid-all-transactions/want.json",
		},
		{
			name:    "missing-filter",
			req:     "testdata/list-transactions/missing-filter/req.json",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t, tt.sql)
			req := &pb.ListTransactionsReq{}
			want := &pb.ListTransactionsResp{}
			jsonFixture(t, tt.req, req)
			jsonFixture(t, tt.want, want)
			got, err := s.ListTransactions(context.Background(), req)
			if update && tt.want != "" {
				writeFixture(t, tt.want, got)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("server.ListTransactions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, want) {
				t.Errorf("server.ListTransactions() = %v, want %v", got, want)
			}
		})
	}
}

func Test_server_UpdateTag(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		req     string
		want    string
		wantErr bool
	}{
		{
			name: "valid",
			sql:  "testdata/update-tag/data.sql",
			req:  "testdata/update-tag/valid/req.json",
			want: "testdata/update-tag/valid/want.json",
		},
		{
			name:    "missing transaction id",
			sql:     "testdata/update-tag/data.sql",
			req:     "testdata/update-tag/missing-transaction-id/req.json",
			wantErr: true,
		},
		{
			name:    "malformed tag id",
			sql:     "testdata/update-tag/data.sql",
			req:     "testdata/update-tag/malformed-tag-id/req.json",
			wantErr: true,
		},
		{
			name:    "malformed transaction id",
			sql:     "testdata/update-tag/data.sql",
			req:     "testdata/update-tag/malformed-transaction-id/req.json",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newServer(t, tt.sql)
			req := &pb.UpdateTagReq{}
			want := &pb.UpdateTagResp{}
			jsonFixture(t, tt.req, req)
			jsonFixture(t, tt.want, want)
			got, err := s.UpdateTag(context.Background(), req)
			if update && tt.want != "" {
				writeFixture(t, tt.want, got)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("server.UpdateTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, want) {
				t.Errorf("server.UpdateTag() = %#v, want %#v", got, want)
			}
		})
	}
}
