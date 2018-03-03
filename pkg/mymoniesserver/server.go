package mymoniesserver

// Server implements mymonies RPC interface.
import (
	"net/http"

	"github.com/joneskoo/mymonies/pkg/mymoniesserver/database"
)

type Server struct {
	DB *database.Postgres

	logger Logger
}

func New(conn string, logger Logger) (http.Handler, error) {
	db, err := database.Connect(conn)
	if err != nil {
		return nil, err
	}
	logger.Println("Connected to database")
	if err := db.CreateTables(); err != nil {
		return nil, err
	}
	server := &Server{DB: db, logger: logger}
	return server.handler(), nil
}

// Logger is a logging interface compatible with log.Logger.
type Logger interface {
	Println(args ...interface{})
}
