package server

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	"github.com/sojebsikder/tunnel/internal"
)

const (
	AllowedMinPort = 5000
	AllowedMaxPort = 6000
)

type Broker struct {
	mu     sync.RWMutex
	agents map[string]*quic.Conn
}

func NewBroker() *Broker {
	return &Broker{
		agents: map[string]*quic.Conn{},
	}
}

func (b *Broker) HandlePublic(w http.ResponseWriter, r *http.Request) {
	sub := strings.Split(r.Host, ".")[0]

	b.mu.RLock()
	conn, ok := b.agents[sub]
	b.mu.RUnlock()

	if !ok {
		http.Error(w, "Tunnel offline", 502)
		return
	}

	// open a new stream for this specific request
	stream, err := conn.OpenStreamSync(context.Background())
	if err != nil {
		http.Error(w, "Failed to open stream", 502)
		return
	}
	defer stream.Close()

	raw, _ := io.ReadAll(r.Body)
	reqMsg := internal.Message{
		ID:      uuid.New().String(),
		Type:    "request",
		Method:  r.Method,
		Path:    r.URL.RequestURI(),
		Headers: r.Header,
		BodyB64: base64.StdEncoding.EncodeToString(raw),
	}

	// Send request over stream
	json.NewEncoder(stream).Encode(reqMsg)

	// read response from same stream
	var resp internal.Message
	if err := json.NewDecoder(stream).Decode(&resp); err != nil {
		http.Error(w, "Agent read error", 502)
		return
	}

	for k, v := range resp.Headers {
		if len(v) > 0 {
			w.Header().Set(k, v[0])
		}
	}
	w.WriteHeader(resp.Status)
	body, _ := base64.StdEncoding.DecodeString(resp.BodyB64)
	w.Write(body)
}

func (b *Broker) StartQUICServer(addr string, tlsConfig *tls.Config) {
	listener, err := quic.ListenAddr(addr, tlsConfig, &quic.Config{MaxIdleTimeout: internal.IdleTimeout})
	if err != nil {
		log.Fatalf("Failed to start QUIC listener: %v", err)
	}

	for {
		conn, _ := listener.Accept(context.Background())
		go b.handleAgentConn(conn)
	}
}

func (b *Broker) handleAgentConn(conn *quic.Conn) {
	// first stream is for registration
	stream, _ := conn.AcceptStream(context.Background())
	var reg internal.Message
	json.NewDecoder(stream).Decode(&reg)

	if reg.TCPPort != "" {
		var port int
		fmt.Sscanf(reg.TCPPort, ":%d", &port) // extract port from ":<port>"

		if port < AllowedMinPort || port > AllowedMaxPort {
			log.Printf("Agent %s requested forbidden port: %d", reg.Path, port)
			conn.CloseWithError(0x02, "Port not allowed")
			return
		}
	}

	b.mu.Lock()
	b.agents[reg.Path] = conn
	b.mu.Unlock()
	log.Println("Agent registered:", reg.Path)

	if reg.TCPPort != "" {
		go b.StartTCPProxy(reg.TCPPort, reg.Path)
	}
}

// handle TCP proxy
func (b *Broker) StartTCPProxy(port string, subdomain string) {
	listener, _ := net.Listen("tcp", port)
	log.Printf("TCP Proxy listener on %s for agent %s", port, subdomain)

	for {
		conn, _ := listener.Accept()

		b.mu.RLock()
		agentConn, ok := b.agents[subdomain]
		b.mu.RUnlock()

		if !ok {
			conn.Close()
			continue
		}

		// open a raw QUIC stream for TCP connection
		stream, _ := agentConn.OpenStreamSync(context.Background())

		// bidirectional copy between TCP and QUIC streams
		go func() {
			defer stream.Close()
			defer conn.Close()
			go io.Copy(stream, conn)
			io.Copy(conn, stream)
		}()
	}
}
