package cmd

import (
	"flag"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/sojebsikder/tunnel/internal/client"
)

func StartClient(args []string) {
	fs := flag.NewFlagSet("tunnel", flag.ExitOnError)
	url := fs.String("url", "http://localhost:4000", "Specify url")
	host := fs.String("host", "localhost:7000", "Specify url")
	fs.Parse(args)

	ws, _, err := websocket.DefaultDialer.Dial("ws://"+*host+"/_tunnel", nil)
	if err != nil {
		panic(err)
	}
	defer ws.Close()
	generatedUrl := "http://" + *host
	fmt.Println("Connected to Tunnel Server. url: " + generatedUrl)

	for {
		var msg map[string]interface{}
		if err := ws.ReadJSON(&msg); err != nil {
			panic(err)
		}

		if msg["type"] == "request" {
			go client.HandleRequest(ws, msg, *url)
		}
	}
}
