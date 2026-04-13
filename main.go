package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"

	"github.com/sojebsikder/tunnel/internal/client"
	"github.com/sojebsikder/tunnel/internal/server"
)

const (
	appName = "Stunnel"
	version = "0.0.2"
)

func main() {
	listen := flag.String("listen", ":8080", "listen address")

	serverURL := flag.String("server", "localhost:8080", "server address")
	localURL := flag.String("url", "http://localhost:3000", "local app url")
	sub := flag.String("subdomain", "demo", "subdomain")
	flag.Parse()

	args := flag.Args()
	if len(args) > 0 {
		switch args[0] {
		case "version":
			fmt.Printf("%s version: %s\n", appName, version)
		case "help":
			fmt.Printf("%s is a tunneling tool.\n\n", appName)
			fmt.Printf("(c) sojebsikder <sojebsikder@gmail.com>\n\n")
			flag.Usage()
		case "start-server":
			b := server.NewBroker()

			// Start QUIC server for agents in a goroutine
			go func() {
				log.Println("QUIC server listening for agents on", *listen)
				b.StartQUICServer(*listen, generateTLSConfig())
			}()

			// Start HTTP server for public traffic
			http.HandleFunc("/", b.HandlePublic)
			log.Println("HTTP server listening for public on :80") // Or another port
			log.Fatal(http.ListenAndServe(":80", nil))
		case "start-agent":
			log.Println("agent starting:", *serverURL, *localURL, *sub)
			client.RunAgent(*serverURL, *localURL, *sub)
		default:
			fmt.Printf("Unknown command: %s\n", args[0])
			fmt.Println("Available commands: download")
			os.Exit(1)
		}
	}
}

// set up a self-signed certificate for QUIC
func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-tunnel-proto"},
	}
}
