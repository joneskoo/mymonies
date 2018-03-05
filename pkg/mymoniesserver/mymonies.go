// This file contains the implementation of mymonies rpc methods.

package mymoniesserver

import (
	"context"
	"database/sql"
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
	query, args, err := transactionFilterQuery(req.Filter)
	if err != nil {
		return nil, err
	}

	rows, err := s.DB.NamedQuery(query, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transactions := make([]*pb.Transaction, 0)
	for rows.Next() {
		var t pb.Transaction
		_ = rows.StructScan(&t)
		transactions = append(transactions, &t)
	}
	if rows.Err() != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	return &pb.ListTransactionsResp{Transactions: transactions}, nil
}

// UpdateTag sets the transaction tag id.
func (s *server) UpdateTag(_ context.Context, req *pb.UpdateTagReq) (*pb.UpdateTagResp, error) {
	if req.TransactionId == "" {
		return nil, twirp.RequiredArgumentError("transaction_id")
	}

	_, err := strconv.ParseInt(req.TagId, 10, 64)
	if req.TagId != "" && err != nil {
		return nil, twirp.InvalidArgumentError("tag_id", err.Error())
	}

	_, err = s.DB.Exec("UPDATE records SET tag_id = $1 WHERE id = $2",
		sql.NullString{String: req.TagId, Valid: req.TagId == ""},
		req.TransactionId)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	return &pb.UpdateTagResp{}, nil
}

func transactionFilterQuery(tf *pb.TransactionFilter) (string, map[string]interface{}, error) {
	if tf == nil {
		return "", nil, twirp.RequiredArgumentError("filter")
	}

	query := &database.SelectQuery{
		Columns: []string{"records.*"},
		From: `records
			LEFT OUTER JOIN imports ON records.import_id = imports.id`,
		OrderBy: "transaction_date DESC, records.id",
	}
	args := make(map[string]interface{})

	if tf.Id != "" {
		query.AndWhere("records.id = :record_id")
		args["record_id"] = tf.Id
	}

	if tf.Account != "" {
		query.AndWhere("imports.account = :account")
		args["account"] = tf.Account
	}

	if tf.Month != "" {
		var startDate, endDate time.Time
		var err error
		startDate, err = time.Parse("2006-01", tf.Month)
		if err != nil {
			return "", nil, twirp.InvalidArgumentError("month", err.Error())
		}
		endDate = startDate.AddDate(0, 1, -1)
		query.AndWhere("records.transaction_date BETWEEN :start AND :end")
		args["start"] = startDate
		args["end"] = endDate
	}

	if tf.Query != "" {
		query.AndWhere(":search IN (payee_payer, records.account, transaction, reference, payer_reference, message)")
		args["search"] = tf.Query
	}

	return query.SQL(), args, nil
}
