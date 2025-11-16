package client

import (
	"crypto/tls"
	"encoding/base64"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sojebsikder/tunnel/internal"
)

// parseURL parses the server URL and returns the host
func parseURL(serverURL string) string {
	u, err := url.Parse(serverURL)
	if err != nil {
		log.Fatalf("failed to parse server URL: %v", err)
	}
	return u.Host
}

func RunAgent(serverURL, localURL, subdomain string) {
	backoff := 1.0

	for {
		log.Println("connecting to server:", serverURL)

		dialer := websocket.Dialer{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}

		conn, _, err := dialer.Dial(serverURL, http.Header{
			// "Authorization": []string{"Bearer " + token},
		})
		if err != nil {
			wait := time.Duration(math.Min(30, backoff)) * time.Second
			backoff *= 1.5
			time.Sleep(wait)
			continue
		}

		backoff = 1.0

		host := parseURL(serverURL)
		log.Println("connected to server:", serverURL)

		// Send register
		conn.WriteJSON(internal.Message{Type: "register", Path: subdomain})
		log.Printf("registered to server: %s.%s", subdomain, host)

		send := make(chan internal.Message, 128)
		done := make(chan struct{})

		// Writer
		go func() {
			ticker := time.NewTicker(internal.PingPeriod)
			defer ticker.Stop()

			for {
				select {
				case msg := <-send:
					if err := conn.WriteJSON(msg); err != nil {
						conn.Close()
						close(done)
						return
					}
				case <-ticker.C:
					if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
						conn.Close()
						close(done)
						return
					}
				}
			}
		}()

		// Reader
		go func() {
			defer close(done)
			for {
				var msg internal.Message
				if err := conn.ReadJSON(&msg); err != nil {
					return
				}
				if msg.Type == "request" {
					go handleAgentRequest(localURL, msg, send)
				}
			}
		}()

		<-done
		conn.Close()
	}
}

func handleAgentRequest(localURL string, msg internal.Message, send chan internal.Message) {
	url := strings.TrimRight(localURL, "/") + msg.Path

	var body []byte
	if msg.BodyB64 != "" {
		body, _ = base64.StdEncoding.DecodeString(msg.BodyB64)
	}

	req, _ := http.NewRequest(msg.Method, url, io.NopCloser(strings.NewReader(string(body))))
	req.ContentLength = int64(len(body))

	for k, v := range msg.Headers {
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		send <- internal.Message{ID: msg.ID, Type: "response", Status: 502}
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	send <- internal.Message{
		ID:      msg.ID,
		Type:    "response",
		Status:  resp.StatusCode,
		Headers: resp.Header,
		BodyB64: base64.StdEncoding.EncodeToString(respBody),
	}
}
