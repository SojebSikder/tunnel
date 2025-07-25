package server

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}
var clientConn *websocket.Conn

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade to WebSocket", http.StatusInternalServerError)
		return
	}
	fmt.Println("Client connected")
	clientConn = conn
}

func HandlePublicRequest(w http.ResponseWriter, r *http.Request) {
	if clientConn == nil {
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
	clientConn.WriteJSON(req)

	// Wait for response
	var resp map[string]interface{}
	clientConn.ReadJSON(&resp)

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
