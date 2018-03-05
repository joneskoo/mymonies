// This file contains the implementation of mymonies rpc methods.

package mymoniesserver

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/twitchtv/twirp"

	"github.com/joneskoo/mymonies/pkg/mymoniesserver/database"
	pb "github.com/joneskoo/mymonies/pkg/rpc/mymonies"
)

// AddImport stores new transaction records.
func (s *server) AddImport(_ context.Context, req *pb.AddImportReq) (*pb.AddImportResp, error) {
	if err := validateAddImportReq(req); err != nil {
		return nil, err
	}
	txn, err := s.DB.Begin()
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	defer txn.Rollback()

	var importid int
	const insertImport = "INSERT INTO imports (filename, account) VALUES ($1, $2) RETURNING id"
	if err := txn.QueryRow(insertImport, req.FileName, req.Account).Scan(&importid); err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	stmt, err := txn.Prepare(pq.CopyIn("records", "import_id", "transaction_date",
		"value_date", "payment_date", "amount", "payee_payer", "account", "bic",
		"transaction", "reference", "payer_reference", "message", "card_number"))
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	defer stmt.Close()
	for _, r := range req.Transactions {
		_, err = stmt.Exec(importid, r.TransactionDate, r.ValueDate, r.PaymentDate,
			r.Amount, r.PayeePayer, r.Account, r.Bic, r.Transaction, r.Reference,
			r.PayerReference, r.Message, r.CardNumber)
		if err != nil {
			return nil, twirp.InternalErrorWith(err)
		}
	}
	if _, err := stmt.Exec(); err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	if err := stmt.Close(); err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	err = txn.Commit()
	return &pb.AddImportResp{}, err
}

func validateAddImportReq(req *pb.AddImportReq) error {
	if req.Account == "" {
		return twirp.RequiredArgumentError("account")
	}
	if req.FileName == "" {
		return twirp.RequiredArgumentError("file_name")
	}
	if len(req.Transactions) == 0 {
		return twirp.RequiredArgumentError("transactions")
	}

	var importErr error
	// must be RFC 3339 format date, with zero time UTC
	mustRFC3339Date := func(argument, timestr string) time.Time {
		t, err := time.Parse(time.RFC3339, timestr)
		if err != nil {
			importErr = twirp.InvalidArgumentError(argument, "must be RFC 3339 timestamp")
		}
		if hour, min, sec := t.Clock(); hour != 0 || min != 0 || sec != 0 {
			e := fmt.Sprintf("time must be zero UTC, was %02d:%02d:%02d", hour, min, sec)
			importErr = twirp.InvalidArgumentError(argument, e)
		}
		return t
	}
	for _, t := range req.Transactions {
		mustRFC3339Date("transaction_date", t.TransactionDate)
		mustRFC3339Date("value_date", t.ValueDate)
		mustRFC3339Date("payment_date", t.PaymentDate)
	}
	return importErr
}

// AddPattern stores a new pattern to tag transactions on import.
func (s *server) AddPattern(ctx context.Context, req *pb.AddPatternReq) (*pb.AddPatternResp, error) {
	if req.Pattern == nil {
		return nil, twirp.RequiredArgumentError("pattern")
	}
	p := req.Pattern

	// List affected transaction ids
	resp, err := s.ListTransactions(ctx, &pb.ListTransactionsReq{Filter: &pb.TransactionFilter{
		Account: p.Account,
		Query:   p.Query,
	}})
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	ids := make([]string, len(resp.Transactions))
	for _, tx := range resp.Transactions {
		ids = append(ids, tx.Id)
	}

	txn, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer txn.Rollback()
	_, err = txn.Exec("INSERT INTO patterns (account, query, tag_id) VALUES ($1, $2, $3)",
		p.Account, p.Query, p.TagId)
	if err != nil {
		return nil, err
	}
	if len(ids) > 0 {
		_, err = txn.Exec("UPDATE records SET tag_id = $1 WHERE id IN ("+strings.Join(ids, ",")+") AND tag_id IS NULL", p.TagId)
		if err != nil {
			return nil, err
		}
	}
	err = txn.Commit()
	return &pb.AddPatternResp{}, err
}

// ListAccounts lists accounts in the database.
func (s *server) ListAccounts(context.Context, *pb.ListAccountsReq) (*pb.ListAccountsResp, error) {
	resp := &pb.ListAccountsResp{Accounts: []*pb.Account{}}
	err := s.DB.Select(&resp.Accounts, "SELECT DISTINCT account AS number from imports")
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	return resp, nil
}

// ListTags lists the transaction tags in the database.
func (s *server) ListTags(context.Context, *pb.ListTagsReq) (*pb.ListTagsResp, error) {
	tags := make([]*pb.Tag, 0)
	rows, err := s.DB.Queryx("SELECT * from tags ORDER BY name")
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	defer rows.Close()
	for rows.Next() {
		var t pb.Tag
		err := rows.StructScan(&t)
		if err != nil {
			return nil, twirp.InternalErrorWith(err)
		}
		tags = append(tags, &t)
	}
	return &pb.ListTagsResp{
		Tags: tags,
	}, nil
}

// ListTransactions lists transactions. Optionally a filter can be provided.
func (s *server) ListTransactions(_ context.Context, req *pb.ListTransactionsReq) (*pb.ListTransactionsResp, error) {
	filter, err := filterFromRequest(req)
	if err != nil {
		return nil, err
	}

	records, err := s.DB.Transactions(*filter)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	return transactionsResponse(records), nil
}

// UpdateTag sets the transaction tag id.
func (s *server) UpdateTag(_ context.Context, req *pb.UpdateTagReq) (*pb.UpdateTagResp, error) {
	txID, err := strconv.Atoi(req.TransactionId)
	if err != nil {
		return nil, twirp.InvalidArgumentError("transaction_id", err.Error())
	}
	tagID, err := strconv.Atoi(req.TagId)
	if err != nil {
		return nil, twirp.InvalidArgumentError("tag_id", err.Error())
	}
	if err := s.DB.SetRecordTag(txID, tagID); err != nil {
		return nil, err
	}
	return &pb.UpdateTagResp{}, nil
}

func filterFromRequest(req *pb.ListTransactionsReq) (*database.TransactionFilter, error) {
	if req.Filter == nil {
		return nil, twirp.RequiredArgumentError("filter")
	}
	var id int
	if req.Filter.Id != "" {
		var err error
		id, err = strconv.Atoi(req.Filter.Id)
		if err != nil && len(req.Filter.Id) > 0 {
			return nil, twirp.InvalidArgumentError("id", err.Error())
		}
	}
	return &database.TransactionFilter{
		Id:      id,
		Account: req.Filter.Account,
		Month:   req.Filter.Month,
		Query:   req.Filter.Query,
	}, nil
}

func transactionsResponse(data []database.Transaction) *pb.ListTransactionsResp {
	rfc3339 := func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.Format(time.RFC3339)
	}
	id := func(i int) string {
		if i == 0 {
			return ""
		}
		return strconv.Itoa(i)
	}
	transactions := make([]*pb.Transaction, len(data))
	for i, t := range data {
		transactions[i] = &pb.Transaction{
			Id:              strconv.Itoa(t.ID),
			TransactionDate: rfc3339(t.TransactionDate),
			ValueDate:       rfc3339(t.ValueDate),
			PaymentDate:     rfc3339(t.PaymentDate),
			Amount:          t.Amount,
			PayeePayer:      t.PayeePayer,
			Account:         t.Account,
			Bic:             t.BIC,
			Transaction:     t.Transaction,
			Reference:       t.Reference,
			PayerReference:  t.PayerReference,
			Message:         t.Message,
			CardNumber:      t.CardNumber,
			TagId:           id(t.TagID),
			ImportId:        id(t.ImportID),
		}
	}
	return &pb.ListTransactionsResp{Transactions: transactions}
}
