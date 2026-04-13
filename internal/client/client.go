package client

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/sojebsikder/tunnel/internal"
)

func RunAgent(serverAddr, localURL, subdomain string) {
	// NextProtos must match the server side
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"quic-tunnel-proto"},
	}

	quicConf := &quic.Config{
		MaxIdleTimeout:  internal.IdleTimeout,
		KeepAlivePeriod: 10 * time.Second,
	}

	for {
		log.Printf("Connecting to QUIC server at %s...", serverAddr)

		ctx := context.Background()
		conn, err := quic.DialAddr(ctx, serverAddr, tlsConf, quicConf)
		if err != nil {
			log.Printf("Dial failed: %v. Retrying in 5s...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Printf("Connected! Registering subdomain: %s", subdomain)

		// open a control stream to register the subdomain
		regStream, err := conn.OpenStreamSync(ctx)
		if err != nil {
			conn.CloseWithError(0, "failed to open reg stream")
			continue
		}

		regMsg := internal.Message{
			Type: "register",
			Path: subdomain,
		}

		if err := json.NewEncoder(regStream).Encode(regMsg); err != nil {
			log.Printf("Registration failed: %v", err)
			regStream.Close()
			continue
		}
		regStream.Close()

		// accept incoming streams from the server
		// each stream represents one HTTP request from the public internet
		for {
			stream, err := conn.AcceptStream(ctx)
			if err != nil {
				log.Printf("Connection lost: %v", err)
				break
			}

			// handle stream
			go handleRequestStream(stream, localURL)
		}

		conn.CloseWithError(0, "reconnecting")
	}
}

func handleRequestStream(stream *quic.Stream, localURL string) {
	defer stream.Close()

	// read the request message from the server
	var msg internal.Message
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&msg); err != nil {
		log.Printf("Failed to decode request from stream: %v", err)
		return
	}

	if msg.Type != "request" {
		return
	}

	// forward the request to the local server
	targetURL := strings.TrimRight(localURL, "/") + msg.Path

	var bodyReader io.Reader
	if msg.BodyB64 != "" {
		body, _ := base64.StdEncoding.DecodeString(msg.BodyB64)
		bodyReader = strings.NewReader(string(body))
	}

	req, err := http.NewRequest(msg.Method, targetURL, bodyReader)
	if err != nil {
		sendErrorResponse(stream, msg.ID, 500)
		return
	}

	// copy headers
	for k, v := range msg.Headers {
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Local request failed: %v", err)
		sendErrorResponse(stream, msg.ID, 502)
		return
	}
	defer resp.Body.Close()

	// prepare and send the response back over the same stream
	respBody, _ := io.ReadAll(resp.Body)

	respMsg := internal.Message{
		ID:      msg.ID,
		Type:    "response",
		Status:  resp.StatusCode,
		Headers: resp.Header,
		BodyB64: base64.StdEncoding.EncodeToString(respBody),
	}

	if err := json.NewEncoder(stream).Encode(respMsg); err != nil {
		log.Printf("Failed to send response back to server: %v", err)
	}
}

func sendErrorResponse(stream *quic.Stream, id string, status int) {
	errResp := internal.Message{
		ID:     id,
		Type:   "response",
		Status: status,
	}
	json.NewEncoder(stream).Encode(errResp)
}
