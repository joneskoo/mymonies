package handler

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/joneskoo/mymonies/database"
)

func New(db database.Database) http.Handler {
	mux := http.NewServeMux()
	h := handler{db, mux}
	mux.HandleFunc("/", h.accounts)
	mux.HandleFunc("/transactions", h.listTransactions)
	mux.HandleFunc("/transactions/", h.updateTag)
	return &h
}

type handler struct {
	db database.Database
	*http.ServeMux
}

func (h handler) accounts(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "accounts.html", h)
}

func (h handler) updateTag(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		log.Printf("updateTag: failed to convert id %q to integer", r.FormValue("id"))
		http.Error(w, "Form value of id could not be parsed as integer", http.StatusBadRequest)
		return
	}
	tag := r.FormValue("tag")

	if err := h.db.SetRecordTag(id, tag); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h handler) listTransactions(w http.ResponseWriter, r *http.Request) {
	account := r.FormValue("account")
	month := r.FormValue("month")
	q := r.FormValue("q")
	if month == "" {
		// Default: previous month
		month = time.Now().AddDate(0, -1, 0).Format("2006-01")
	}

	transactions := h.db.Transactions()
	if account != "" {
		transactions = transactions.Account(account)
	}
	if month != "" {
		transactions = transactions.Month(month)
	}
	if q != "" {
		transactions = transactions.Search(q)
	}
	records, err := transactions.Records()
	if err != nil {
		log.Printf("Error fetching transactions: %v", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	tags, err := transactions.SumTransactionsByTag()
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
	}{account, records, tags, month}
	h.render(w, r, "list.html", data)
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
		"accounts": h.db.ListAccounts,
		"date":     func(t time.Time) string { return t.Format("2006-01-02") },
		"tags":     h.db.ListTags,
	}
}
