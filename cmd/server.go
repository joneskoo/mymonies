package cmd

import (
	"log"
	"net"
	"net/http"
	"os"

	"github.com/joneskoo/mymonies/pkg/middleware"
	"github.com/joneskoo/mymonies/pkg/mymoniesserver"
	"github.com/joneskoo/mymonies/pkg/rpc/mymonies"
	"github.com/spf13/cobra"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Mymonies backend server",
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, _ := cmd.Flags().GetString("conn")
		listen, _ := cmd.Flags().GetString("listen")
		logger := log.New(os.Stdout, "[mymonies] ", log.Lshortfile)
		logger.Println("Listening on http://" + listen)

		server, err := mymoniesserver.New(conn, logger)
		if err != nil {
			return err
		}
		h := handler(server)
		return http.ListenAndServe(listen, h)
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	serverCmd.Flags().String("conn", "database=mymonies", "PostgreSQL connection string")
	serverCmd.Flags().String("listen", defaultListen(), "HTTP server listen address")
}

func defaultListen() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	return net.JoinHostPort("127.0.0.1", port)

}

func handler(server mymonies.Mymonies) http.Handler {
	mux := http.NewServeMux()

	// Twirp RPC handler with prometheus metrics
	// hooks := prometheus.NewServerHooks(nil)
	twirpHandler := mymonies.NewMymoniesServer(server, nil)
	mux.Handle(mymonies.MymoniesPathPrefix, twirpHandler)

	// Prometheus metrics endpoint
	// mux.Handle("/metrics", promhttp.Handler())

	// Static file server
	mux.Handle("/", http.FileServer(http.Dir("frontend")))

	// Apply middlewares
	var h http.Handler = mux
	for _, mw := range middlewares {
		h = mw(h)
	}
	return h
}

var middlewares = []middleware.Middleware{
	middleware.RequestLogger(os.Stdout),
	middleware.SetResponseHeader("Cache-Control", "no-cache"),
}
