package handlers

import (
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/joneskoo/mymonies/database"
)

func New(db *database.Database) http.Handler {
	mux := http.NewServeMux()
	h := handler{db, mux}
	mux.HandleFunc("/", h.accounts)
	mux.HandleFunc("/accounts/", h.list)
	return &h
}

type handler struct {
	db *database.Database
	*http.ServeMux
}

func (h handler) accounts(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "accounts.html", h)
}

func (h handler) list(w http.ResponseWriter, r *http.Request) {
	p := strings.LastIndex(r.URL.Path, "/")
	account := r.URL.Path[p+1:]

	records, err := h.db.ListRecordsByAccount(account)
	if err != nil {
		log.Printf("Error fetching records: %v", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	data := struct {
		Account string
		Records []database.Record
	}{account, records}
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
