// Package database implements a mymonies database interface with backed by PostgreSQL
package database

import (
	"fmt"
	"strings"

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

var createTableSQL = []string{
	importsCreateTableSQL,
	tagsCreateTableSQL,
	recordsCreateTableSQL,
	patternsCreateTableSQL,
}

// CreateTables creates any missing database tables.
// This is safe to call multiple times.
func (db *Postgres) CreateTables() error {
	txn, err := db.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()

	for _, q := range createTableSQL {
		if _, err := txn.Exec(q); err != nil {
			return err
		}
	}
	return txn.Commit()
}
