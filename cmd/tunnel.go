package cmd

import (
	"crypto/rand"
	"flag"
	"fmt"
	"math/big"

	"github.com/gorilla/websocket"
	"github.com/sojebsikder/tunnel/internal/client"
)

func randomSubdomain() string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	const length = 10 // increase for better uniqueness

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			panic("crypto/rand failed: " + err.Error()) // or handle gracefully
		}
		result[i] = letters[n.Int64()]
	}
	return string(result)
}

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

	subdomain := randomSubdomain()

	ws.WriteJSON(map[string]interface{}{
		"type":      "register",
		"subdomain": subdomain,
	})

	generatedUrl := "http://" + subdomain + "." + *host
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
