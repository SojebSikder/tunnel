package client

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

func HandleRequest(ws *websocket.Conn, msg map[string]interface{}, url string) {
	req, _ := http.NewRequest(
		msg["method"].(string),
		url+msg["path"].(string),
		bytes.NewBufferString(msg["body"].(string)),
	)

	if hdrs, ok := msg["headers"].(map[string]interface{}); ok {
		for k, v := range hdrs {
			if vv, ok := v.([]interface{}); ok && len(vv) > 0 {
				if val, ok := vv[0].(string); ok {
					req.Header.Set(k, val)
				}
			}
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		ws.WriteJSON(map[string]interface{}{
			"id":     msg["id"],
			"type":   "response",
			"status": 502,
			"body":   "Local server error",
		})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	headers := map[string]string{}
	for k, v := range resp.Header {
		headers[k] = v[0]
	}

	ws.WriteJSON(map[string]interface{}{
		"id":      msg["id"],
		"type":    "response",
		"status":  resp.StatusCode,
		"headers": headers,
		"body":    string(body),
	})
}
