package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/sojebsikder/tunnel/internal/client"
	"github.com/sojebsikder/tunnel/internal/server"
)

func main() {
	mode := flag.String("mode", "server", "server or agent")
	listen := flag.String("listen", ":8080", "listen address")
	serverURL := flag.String("server", "ws://localhost:8080/ws", "server url")
	localURL := flag.String("url", "http://localhost:3000", "local app url")
	sub := flag.String("subdomain", "demo", "subdomain")
	flag.Parse()

	if *mode == "server" {
		b := server.NewBroker()
		http.HandleFunc("/ws", b.HandleWS)
		http.HandleFunc("/", b.HandlePublic)
		log.Println("server listening on", *listen)
		log.Fatal(http.ListenAndServe(*listen, nil))
	}

	log.Println("agent starting:", *serverURL, *localURL, *sub)
	client.RunAgent(*serverURL, *localURL, *sub)
}
