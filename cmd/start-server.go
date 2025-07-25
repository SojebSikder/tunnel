package cmd

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/sojebsikder/tunnel/internal/server"
)

func StartServer(args []string) {
	fs := flag.NewFlagSet("start-server", flag.ExitOnError)
	port := fs.String("port", "7000", "Specify url")
	fs.Parse(args)

	http.HandleFunc("/_tunnel", server.HandleWebSocket)
	http.HandleFunc("/", server.HandlePublicRequest)

	fmt.Println("Tunnel Server listening on :" + *port)
	http.ListenAndServe(":"+*port, nil)
}
