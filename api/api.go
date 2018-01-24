package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/joneskoo/mymonies/database"
	"github.com/mholt/binding"
)

type apiResponse struct {
	// Data is the API response data.
	Data interface{} `json:"data,omitempty"`

	// Error is set when the API request failed.
	Error *errorResponse `json:"error,omitempty"`
}

func (res apiResponse) WriteTo(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")

	code := http.StatusOK
	if res.Error != nil {
		code = res.Error.HTTPCode()
	} else if res.Data == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.WriteHeader(code)

	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Printf("could not encode JSON: %v", err)
	}
}

func New(db *database.Postgres) http.Handler {
	mux := http.NewServeMux()

	// Methods without parameters
	mux.HandleFunc("/accounts", apiMethod(db, handlerFunc(accounts)))
	mux.HandleFunc("/tags", apiMethod(db, handlerFunc(tags)))

	// Methods with parameters (FieldMapper)
	mux.HandleFunc("/transactions", apiMethod(db, new(transactions)))
	mux.HandleFunc("/update-tag", apiMethod(db, new(updateTag)))
	return mux
}

func apiMethod(db *database.Postgres, h handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse input if handler is FieldMapper
		if mapper, ok := h.(binding.FieldMapper); ok {
			if err := binding.Bind(r, mapper); err != nil {
				apiResponse{Error: &badParametersError}.WriteTo(w)
				return
			}
		}

		data, err := h.Handle(db)
		if err != nil {
			// FIXME: handle not found error
			apiResponse{Error: &internalServerError}.WriteTo(w)
			return
		}
		apiResponse{Data: data}.WriteTo(w)
	}
}

// handler is the interface API methods implement.
//
// If the method takes input, handler may implement binding.FieldMapper
// interface to map the request to it, e.g. fields of struct.
type handler interface {
	// Handle uses database to produce data or error.
	Handle(db *database.Postgres) (interface{}, error)
}

// handlerFunc is an adapter that implements handler interface for a regular function.
type handlerFunc func(db *database.Postgres) (interface{}, error)

func (f handlerFunc) Handle(db *database.Postgres) (interface{}, error) { return f(db) }

func accounts(db *database.Postgres) (interface{}, error) {
	return db.ListAccounts()
}

func tags(db *database.Postgres) (interface{}, error) {
	tags, err := db.ListTags()
	if err != nil {
		log.Printf("Error fetching tags: %v", err)
		return nil, err
	}

	patterns, err := db.ListPatterns(-1)
	if err != nil {
		log.Printf("Error fetching patterns: %v", err)
		return nil, err
	}
	type patternData struct {
		Account string `json:"account"`
		Query   string `json:"query"`
	}
	type tagData struct {
		TagID    int           `json:"tag_id"`
		TagName  string        `json:"tag_name"`
		Patterns []patternData `json:"patterns"`
	}
	data := make([]tagData, 0)

	for _, t := range tags {
		queries := make([]patternData, 0)
		for _, p := range patterns {
			if t.ID == p.TagID {
				queries = append(queries, patternData{
					Query:   p.Query,
					Account: p.Account,
				})
			}
		}
		data = append(data, tagData{
			TagID:    t.ID,
			TagName:  t.Name,
			Patterns: queries,
		})

	}
	return data, nil
}

// transactions returns transaction data with search query
type transactions struct {
	database.TransactionFilter
}

func (h *transactions) FieldMap(*http.Request) binding.FieldMap {
	return binding.FieldMap{
		&h.Id:      "id",
		&h.Account: "account",
		&h.Month:   "month",
		&h.Query:   "q",
	}
}

func (h transactions) Handle(db *database.Postgres) (interface{}, error) {
	records, err := db.Transactions(h.TransactionFilter)

	if err != nil {
		log.Printf("Error fetching transactions: %v", err)
		return nil, err
	}

	tags, err := db.SumTransactionsByTag(h.TransactionFilter)
	if err != nil {
		log.Printf("Error fetching tags: %v", err)
		return nil, err
	}

	type data struct {
		Transactions []database.Transaction `json:"transactions"`
		Tags         []database.TagAmount   `json:"tags"`
	}
	return data{records, tags}, nil
}

type updateTag struct {
	id  int
	tag int
}

func (h *updateTag) FieldMap(*http.Request) binding.FieldMap {
	return binding.FieldMap{
		&h.id:  binding.Field{Form: "id", Required: true},
		&h.tag: binding.Field{Form: "tag_id"},
	}
}

func (h updateTag) Handle(db *database.Postgres) (interface{}, error) {
	if err := db.SetRecordTag(h.id, h.tag); err != nil {
		log.Printf("SetRecordTag(%d, %d) failed: %v", h.id, h.tag, err)
		return nil, err
	}
	return struct{}{}, nil
}

type addPattern struct {
	account string
	query   string
	tagID   int
}

func (h *addPattern) FieldMap(*http.Request) binding.FieldMap {
	return binding.FieldMap{
		&h.account: binding.Field{Form: "account", Required: true},
		&h.query:   binding.Field{Form: "query", Required: true},
		&h.tagID:   binding.Field{Form: "tag_id", Required: true},
	}
}

func (h *addPattern) Handler(db *database.Postgres) (interface{}, error) {
	if err := db.AddPattern(h.account, h.query, h.tagID); err != nil {
		log.Printf("AddPattern(%v, %v, %v) failed: %v", h.account, h.query, h.tagID, err)
		return nil, err
	}
	return struct{}{}, nil
}
