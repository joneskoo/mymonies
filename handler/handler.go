package handler

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/joneskoo/mymonies/database"
	"github.com/mholt/binding"
)

func New(db *database.Postgres) http.Handler {
	mux := http.NewServeMux()
	h := handler{db, mux}
	mux.HandleFunc("/", h.accounts)
	mux.HandleFunc("/transactions", h.listTransactions)
	mux.HandleFunc("/transactions/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.transactionDetails(w, r)
		case http.MethodPost:
			h.updateTag(w, r)
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	})
	mux.HandleFunc("/tags/", h.tagDetails)
	mux.HandleFunc("/patterns/", h.addPattern)
	return &h
}

type handler struct {
	db *database.Postgres
	*http.ServeMux
}

func (h handler) PreviousMonth() string {
	return time.Now().AddDate(0, -1, 0).Format("2006-01")
}

func (h handler) accounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.db.ListAccounts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tags, err := h.db.ListTags()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	patterns, err := h.db.ListPatterns(-1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	patternRows := make([][]string, 0)
	for _, p := range patterns {
		var tag string
		for _, t := range tags {
			if t.ID == p.TagID {
				tag = t.Name
			}
		}
		patternRows = append(patternRows, []string{p.Account, p.Query, tag})
	}

	h.render(w, r, "accounts.html", map[string]interface{}{
		"PreviousMonth": h.PreviousMonth(),
		"Accounts":      accounts,
		"Tags":          tags,
		"Patterns":      patternRows,
	})
}

func (h handler) tagDetails(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/tags/"))
	if err != nil {
		http.Error(w, "", http.StatusNotFound)
		return
	}

	tag, err := h.db.Tag(id)
	if err != nil {
		log.Printf("failed to get tag %v: %v", id, err)
		http.Error(w, "Not found.", http.StatusNotFound)
		return
	}
	patterns, err := h.db.ListPatterns(id)
	if err != nil {
		log.Printf("failed to get patterns for tag %v: %v", id, err)
		http.Error(w, "Not found.", http.StatusInternalServerError)
		return
	}

	ctx := map[string]interface{}{
		"ID":       tag.ID,
		"Name":     tag.Name,
		"Patterns": patterns,
	}
	h.render(w, r, "tag.html", ctx)
	return
}

type updateTagRequest struct {
	id  int
	tag int
}

func (utr *updateTagRequest) FieldMap(*http.Request) binding.FieldMap {
	return binding.FieldMap{
		&utr.id:  "id",
		&utr.tag: "tag_id",
	}
}

func (h handler) updateTag(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")

	utr := new(updateTagRequest)
	err := binding.Bind(r, utr)
	if err != nil {
		log.Printf("bad request updating tag: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.db.SetRecordTag(utr.id, utr.tag); err != nil {
		log.Printf("SetRecordTag(%d, %d) failed: %v", utr.id, utr.tag, err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type addPatternRequest struct {
	Account string
	Query   string
	TagID   int
}

func (apr *addPatternRequest) FieldMap(*http.Request) binding.FieldMap {
	return binding.FieldMap{
		&apr.Account: binding.Field{Form: "account", Required: true},
		&apr.Query:   binding.Field{Form: "query", Required: true},
		&apr.TagID:   binding.Field{Form: "tag_id", Required: true},
	}
}

func (h handler) addPattern(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")

	apr := new(addPatternRequest)
	err := binding.Bind(r, apr)
	if err != nil {
		log.Printf("bad request updating tag: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.db.AddPattern(apr.Account, apr.Query, apr.TagID); err != nil {
		log.Printf("AddPattern(%v, %v, %v) failed: %v", apr.Account, apr.Query, apr.TagID, err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h handler) transactionDetails(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/transactions/"))
	if err != nil {
		http.Error(w, "", http.StatusNotFound)
		return
	}

	transactions, err := h.db.Transactions(database.Id(id))
	if err != nil {
		log.Printf("failed to get transaction: %v", err)
		http.Error(w, "", http.StatusNotFound)
		return
	}
	h.render(w, r, "transaction_detail.html", transactions[0])
}

type listTransactionsRequest struct {
	account string
	month   string
	query   string
}

func (ltr *listTransactionsRequest) FieldMap(*http.Request) binding.FieldMap {
	return binding.FieldMap{
		&ltr.account: "account",
		&ltr.month:   "month",
		&ltr.query:   "q",
	}
}

func (h handler) listTransactions(w http.ResponseWriter, r *http.Request) {
	ltr := new(listTransactionsRequest)
	if err := binding.Bind(r, ltr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	records, err := h.db.Transactions(
		database.Account(ltr.account),
		database.Month(ltr.month),
		database.Search(ltr.query))

	if err != nil {
		log.Printf("Error fetching transactions: %v", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	tags, err := h.db.SumTransactionsByTag(
		database.Account(ltr.account),
		database.Month(ltr.month),
		database.Search(ltr.query))

	if err != nil {
		log.Printf("Error fetching tags: %v", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	data := struct {
		Account      string
		Transactions []database.Transaction
		Tags         map[string]float64
		Month        string
		Query        string
	}{ltr.account, records, tags, ltr.month, ltr.query}
	h.render(w, r, "transaction_list.html", data)
}

func (h handler) render(w http.ResponseWriter, r *http.Request, templateFile string, data interface{}) {
	tmpl, err := template.New(templateFile).Funcs(h.funcMap()).ParseFiles("templates/" + templateFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Failed to render template: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h handler) funcMap() template.FuncMap {
	return template.FuncMap{
		"date":   func(t time.Time) string { return t.Format("2006-01-02") },
		"tags":   h.db.ListTags,
		"tag":    h.db.Tag,
		"import": h.db.Import,
	}
}
