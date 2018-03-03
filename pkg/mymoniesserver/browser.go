package mymoniesserver

import (
	"context"
	"strconv"
	"time"

	"github.com/twitchtv/twirp"

	"github.com/joneskoo/mymonies/pkg/mymoniesserver/database"
	pb "github.com/joneskoo/mymonies/pkg/rpc/mymonies"
)

// AddPattern stores a new pattern to tag transactions on import.
func (s *Server) AddPattern(_ context.Context, req *pb.AddPatternReq) (*pb.AddPatternResp, error) {
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
func (s *Server) ListAccounts(context.Context, *pb.ListAccountsReq) (*pb.ListAccountsResp, error) {
	accounts, err := s.DB.ListAccounts()
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	return &pb.ListAccountsResp{
		Accounts: convertAccounts(accounts),
	}, nil
}

// ListTags lists the transaction tags in the database.
func (s *Server) ListTags(context.Context, *pb.ListTagsReq) (*pb.ListTagsResp, error) {
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
func (s *Server) ListTransactions(_ context.Context, req *pb.ListTransactionsReq) (*pb.ListTransactionsResp, error) {
	filter, err := filterFromRequest(req)
	if err != nil {
		return nil, err
	}

	records, err := s.DB.Transactions(filter)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	return transactionsResponse(records), nil
}

// UpdateTag sets the transaction tag id.
func (s *Server) UpdateTag(_ context.Context, req *pb.UpdateTagReq) (*pb.UpdateTagResp, error) {
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

func filterFromRequest(req *pb.ListTransactionsReq) (database.TransactionFilter, error) {
	filter := database.TransactionFilter{}
	if req.Filter == nil {
		return filter, nil
	}
	if req.Filter.Id != "" {
		id, err := strconv.Atoi(req.Filter.Id)
		if err != nil && len(req.Filter.Id) > 0 {
			return filter, twirp.InvalidArgumentError("id", err.Error())
		}
		filter.Id = id
	}
	filter.Account = req.Filter.Account
	filter.Month = req.Filter.Month
	filter.Query = req.Filter.Query
	return filter, nil
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
