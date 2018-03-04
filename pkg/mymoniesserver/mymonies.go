// This file contains the implementation of mymonies rpc methods.

package mymoniesserver

import (
	"context"
	"strconv"
	"time"

	"github.com/twitchtv/twirp"

	"github.com/joneskoo/mymonies/pkg/mymoniesserver/database"
	pb "github.com/joneskoo/mymonies/pkg/rpc/mymonies"
)

// AddImport stores new transaction records.
func (s *server) AddImport(_ context.Context, req *pb.AddImportReq) (*pb.AddImportResp, error) {
	var importErr error
	mustRFC3339 := func(argument, timestr string) time.Time {
		t, err := time.Parse(time.RFC3339, timestr)
		if err != nil {
			importErr = twirp.InvalidArgumentError(argument, "must be RFC 3339 timestamp")
		}
		return t
	}
	var transactions []*database.Transaction
	for _, t := range req.Transactions {
		transactions = append(transactions, &database.Transaction{
			TransactionDate: mustRFC3339("transaction_date", t.TransactionDate),
			ValueDate:       mustRFC3339("value_date", t.ValueDate),
			PaymentDate:     mustRFC3339("payment_date", t.PaymentDate),
			Amount:          t.Amount,
			PayeePayer:      t.PayeePayer,
			Account:         t.Account,
			BIC:             t.Bic,
			Transaction:     t.Transaction,
			Reference:       t.Reference,
			PayerReference:  t.PayerReference,
			Message:         t.Message,
			CardNumber:      t.CardNumber,
			// TagID:           t.TagId,
		})
		if importErr != nil {
			return nil, importErr
		}
	}
	s.DB.AddImport(req.FileName, req.Account, transactions)
	return &pb.AddImportResp{}, nil
}

// AddPattern stores a new pattern to tag transactions on import.
func (s *server) AddPattern(_ context.Context, req *pb.AddPatternReq) (*pb.AddPatternResp, error) {
	p := req.Pattern
	tagID, err := strconv.Atoi(p.TagId)
	if err != nil {
		return nil, twirp.InvalidArgumentError("tag_id", "invalid tag id value")
	}
	if err := s.DB.AddPattern(p.Account, p.Query, tagID); err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	return &pb.AddPatternResp{}, nil
}

// ListAccounts lists accounts in the database.
func (s *server) ListAccounts(context.Context, *pb.ListAccountsReq) (*pb.ListAccountsResp, error) {
	accounts, err := s.DB.ListAccounts()
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	return &pb.ListAccountsResp{
		Accounts: convertAccounts(accounts),
	}, nil
}

// ListTags lists the transaction tags in the database.
func (s *server) ListTags(context.Context, *pb.ListTagsReq) (*pb.ListTagsResp, error) {
	tags, err := s.DB.ListTags()
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	// tags, err := s.DB.SumTransactionsByTag(filter)
	// if err != nil {
	// 	return nil, err
	// }

	return &pb.ListTagsResp{
		Tags: convertTags(tags),
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
		return &database.TransactionFilter{}, nil
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
	transactions := make([]*pb.Transaction, len(data))
	for i, t := range data {
		transactions[i] = &pb.Transaction{
			Id:              strconv.Itoa(t.ID),
			TransactionDate: t.TransactionDate.Format(time.RFC3339),
			ValueDate:       t.ValueDate.Format(time.RFC3339),
			PaymentDate:     t.PaymentDate.Format(time.RFC3339),
			Amount:          t.Amount,
			PayeePayer:      t.PayeePayer,
			Account:         t.Account,
			Bic:             t.BIC,
			Transaction:     t.Transaction,
			Reference:       t.Reference,
			PayerReference:  t.PayerReference,
			Message:         t.Message,
			CardNumber:      t.CardNumber,
			TagId:           strconv.Itoa(t.TagID),
			ImportId:        strconv.Itoa(t.ImportID),
		}
	}
	return &pb.ListTransactionsResp{Transactions: transactions}
}

func convertTags(in []database.Tag) []*pb.Tag {
	out := make([]*pb.Tag, len(in))
	for i, t := range in {
		out[i] = &pb.Tag{
			Id:   strconv.Itoa(t.ID),
			Name: t.Name,
			// Patterns: queries,
		}
	}
	return out
}

func convertAccounts(in []string) []*pb.Account {
	out := make([]*pb.Account, len(in))
	for i, t := range in {
		out[i] = &pb.Account{
			Number: t,
		}
	}
	return out
}
