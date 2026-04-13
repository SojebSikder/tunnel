package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"log"
	"math/big"
	"net/http"

	"github.com/sojebsikder/tunnel/internal/client"
	"github.com/sojebsikder/tunnel/internal/server"
)

func main() {
	mode := flag.String("mode", "server", "server or agent")
	listen := flag.String("listen", ":8080", "listen address")

	serverURL := flag.String("server", "localhost:8080", "server address")
	localURL := flag.String("url", "http://localhost:3000", "local app url")
	sub := flag.String("subdomain", "demo", "subdomain")
	flag.Parse()

	if *mode == "server" {
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
	} else {
		log.Println("agent starting:", *serverURL, *localURL, *sub)
		client.RunAgent(*serverURL, *localURL, *sub)
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
