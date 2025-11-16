package server

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sojebsikder/tunnel/internal"
)

type Agent struct {
	Subdomain string
	Conn      *websocket.Conn
	Send      chan internal.Message // writer owns this

	mu      sync.Mutex
	pending map[string]chan internal.Message
	closed  bool
}

func NewAgent(sub string, conn *websocket.Conn) *Agent {
	return &Agent{
		Subdomain: sub,
		Conn:      conn,
		Send:      make(chan internal.Message, 64),
		pending:   make(map[string]chan internal.Message),
	}
}

func (a *Agent) Close() {
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		return
	}
	a.closed = true

	_ = a.Conn.Close()

	for id, ch := range a.pending {
		close(ch)
		delete(a.pending, id)
	}
	a.mu.Unlock()
}

func (a *Agent) runReadLoop() {
	a.Conn.SetReadLimit(10 << 20)
	a.Conn.SetReadDeadline(time.Now().Add(internal.PongWait))
	a.Conn.SetPongHandler(func(string) error {
		a.Conn.SetReadDeadline(time.Now().Add(internal.PongWait))
		return nil
	})

	for {
		var msg internal.Message
		if err := a.Conn.ReadJSON(&msg); err != nil {
			log.Printf("read error for %s: %v", a.Subdomain, err)
			a.Close()
			return
		}

		if msg.Type == "response" {
			a.mu.Lock()
			ch, ok := a.pending[msg.ID]
			if ok {
				ch <- msg
				delete(a.pending, msg.ID)
			}
			a.mu.Unlock()
			continue
		}

		log.Printf("unexpected message from agent %s: %s", a.Subdomain, msg.Type)
	}
}

func (a *Agent) runWriteLoop() {
	ticker := time.NewTicker(internal.PingPeriod)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-a.Send:
			if !ok {
				return
			}
			if err := a.Conn.WriteJSON(msg); err != nil {
				log.Printf("write error %s: %v", a.Subdomain, err)
				a.Close()
				return
			}

		case <-ticker.C:
			if err := a.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("ping error %s: %v", a.Subdomain, err)
				a.Close()
				return
			}
		}
	}
}
