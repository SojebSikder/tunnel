package server

import (
	"encoding/base64"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sojebsikder/tunnel/internal"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Broker struct {
	mu     sync.RWMutex
	agents map[string]*Agent
}

func NewBroker() *Broker {
	return &Broker{
		agents: map[string]*Agent{},
	}
}

func (b *Broker) registerAgent(sub string, conn *websocket.Conn) *Agent {
	b.mu.Lock()
	defer b.mu.Unlock()

	if old, ok := b.agents[sub]; ok {
		old.Close()
	}

	a := NewAgent(sub, conn)
	b.agents[sub] = a
	return a
}

func (b *Broker) getAgent(sub string) (*Agent, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	a, ok := b.agents[sub]
	return a, ok
}

func (b *Broker) HandlePublic(w http.ResponseWriter, r *http.Request) {
	sub := strings.Split(r.Host, ".")[0]

	a, ok := b.getAgent(sub)
	if !ok {
		http.Error(w, "Tunnel offline", 502)
		return
	}

	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		http.Error(w, "Tunnel closed", 502)
		return
	}
	a.mu.Unlock()

	id := uuid.New().String()
	raw, _ := io.ReadAll(r.Body)
	r.Body.Close()

	reqMsg := internal.Message{
		ID:      id,
		Type:    "request",
		Method:  r.Method,
		Path:    r.URL.RequestURI(),
		Headers: r.Header,
		BodyB64: base64.StdEncoding.EncodeToString(raw),
	}

	respCh := make(chan internal.Message, 1)

	a.mu.Lock()
	a.pending[id] = respCh
	a.mu.Unlock()

	select {
	case a.Send <- reqMsg:
	default:
		a.mu.Lock()
		delete(a.pending, id)
		a.mu.Unlock()
		http.Error(w, "Agent overloaded", 503)
		return
	}

	select {
	case resp := <-respCh:
		for k, v := range resp.Headers {
			if len(v) > 0 {
				w.Header().Set(k, v[0])
			}
		}
		if resp.Status == 0 {
			resp.Status = 502
		}
		body, _ := base64.StdEncoding.DecodeString(resp.BodyB64)
		w.WriteHeader(resp.Status)
		w.Write(body)

	case <-time.After(30 * time.Second):
		a.mu.Lock()
		delete(a.pending, id)
		a.mu.Unlock()
		http.Error(w, "Timeout", 504)
	}
}

func (b *Broker) HandleWS(w http.ResponseWriter, r *http.Request) {
	// if r.Header.Get("Authorization") != "Bearer "+b.apiKey {
	// 	http.Error(w, "unauthorized", 401)
	// 	return
	// }

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	var reg internal.Message
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err := conn.ReadJSON(&reg); err != nil || reg.Type != "register" {
		conn.Close()
		return
	}

	agent := b.registerAgent(reg.Path, conn)
	log.Println("agent registered:", reg.Path)

	go agent.runWriteLoop()
	go agent.runReadLoop()
}
