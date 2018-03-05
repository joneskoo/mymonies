// Package database implements a mymonies database interface with backed by PostgreSQL
package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	// Load postgresql driver
	_ "github.com/lib/pq"
)

// Connect opens a new PostgreSQL database connection and verifies connection
// by pinging the server.
func Connect(conn string) (*Postgres, error) {
	db, err := sqlx.Connect("postgres", conn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}
	// use JSON field name as database field name
	db.Mapper = reflectx.NewMapperFunc("json", strings.ToLower)
	return &Postgres{db}, nil
}

// Postgres represents a database connection and implements data models.
type Postgres struct {
	*sqlx.DB
}

// Close closes the connection to the database.
// Database connection is normally called only at program exit.
func (db *Postgres) Close() error { return db.Close() }

func logQuery(query string, start time.Time) {
	log.Printf("SQL: %v (%v)\n", query, time.Now().Sub(start))
}

func (db *Postgres) Exec(query string, arg ...interface{}) (sql.Result, error) {
	defer logQuery(query, time.Now())
	return db.DB.Exec(query, arg...)
}

func (db *Postgres) Query(query string, arg ...interface{}) (*sql.Rows, error) {
	defer logQuery(query, time.Now())
	return db.DB.Query(query, arg...)
}

func (db *Postgres) QueryRow(query string, arg ...interface{}) *sql.Row {
	defer logQuery(query, time.Now())
	return db.DB.QueryRow(query, arg...)
}

func (db *Postgres) Select(dest interface{}, query string, args ...interface{}) error {
	defer logQuery(query, time.Now())
	return db.DB.Select(dest, query, args...)
}

func (db *Postgres) Get(dest interface{}, query string, args ...interface{}) error {
	defer logQuery(query, time.Now())
	return db.DB.Get(dest, query, args...)
}

func (db *Postgres) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	defer logQuery(query, time.Now())
	return db.DB.Queryx(query, args...)
}

func (db *Postgres) QueryRowsx(query string, arg ...interface{}) *sqlx.Row {
	defer logQuery(query, time.Now())
	return db.DB.QueryRowx(query, arg...)
}

func (db *Postgres) NamedQuery(query string, arg interface{}) (*sqlx.Rows, error) {
	defer logQuery(query, time.Now())
	return db.DB.NamedQuery(query, arg)
}

func (db *Postgres) NamedExec(query string, arg interface{}) (sql.Result, error) {
	defer logQuery(query, time.Now())
	return db.DB.NamedExec(query, arg)
}

// CreateTables creates any missing database tables.
// This is safe to call multiple times.
func (db *Postgres) CreateTables() error {
	txn, err := db.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()

	for _, t := range tables {
		if _, err := txn.Exec(t.create); err != nil {
			return err
		}
	}
	return txn.Commit()
}

// DropTables deletes any mymonies tables from database.
// This is permanent cannot be undone.
func (db *Postgres) DropTables() error {
	txn, err := db.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()

	for i := len(tables) - 1; i >= 0; i-- {
		t := tables[i]
		if _, err := txn.Exec(t.drop); err != nil {
			return err
		}
	}
	return txn.Commit()
}
