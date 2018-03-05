package mymoniesserver

import (
	"github.com/joneskoo/mymonies/pkg/mymoniesserver/database"
	"github.com/joneskoo/mymonies/pkg/rpc/mymonies"
)

func New(conn string, logger Logger) (mymonies.Mymonies, error) {
	db, err := database.Connect(conn)
	if err != nil {
		return nil, err
	}
	logger.Println("Connected to database")
	if err := db.CreateTables(); err != nil {
		return nil, err
	}
	server := &server{DB: db, logger: logger}
	return server, nil
}

type server struct {
	DB *database.Postgres

	logger Logger
}

// Logger is a logging interface compatible with log.Logger.
type Logger interface {
	Println(args ...interface{})
}
