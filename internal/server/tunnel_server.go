package server

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

var clients = make(map[string]*websocket.Conn)

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade to WebSocket", http.StatusInternalServerError)
		return
	}

	var reg map[string]interface{}
	if err := conn.ReadJSON(&reg); err != nil || reg["type"] != "register" {
		conn.Close()
		return
	}

	subdomain := reg["subdomain"].(string)
	clients[subdomain] = conn
	fmt.Printf("Client registered with subdomain: %s\n", subdomain)
}

func HandlePublicRequest(w http.ResponseWriter, r *http.Request) {
	host := r.Host // e.g., app1.tunnel.com
	subdomain := strings.Split(host, ".")[0]

	conn, ok := clients[subdomain]
	if !ok {
		http.Error(w, "Tunnel not connected", http.StatusBadGateway)
		return
	}

	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	req := map[string]interface{}{
		"id":      "req-123",
		"type":    "request",
		"method":  r.Method,
		"path":    r.URL.Path,
		"headers": r.Header,
		"body":    string(body),
	}
	conn.WriteJSON(req)

	var resp map[string]interface{}
	conn.ReadJSON(&resp)

	statusCode := 502
	if raw, ok := resp["status"]; ok {
		if s, ok := raw.(float64); ok {
			statusCode = int(s)
		}
	}
	w.WriteHeader(statusCode)

	if hdrs, ok := resp["headers"].(map[string]interface{}); ok {
		for k, v := range hdrs {
			if str, ok := v.(string); ok {
				w.Header().Set(k, str)
			}
		}
	}

	w.Write([]byte(resp["body"].(string)))
}
