package internal

import (
	"time"
)

type Message struct {
	ID      string              `json:"id"`
	Type    string              `json:"type"` // register, request, response
	Method  string              `json:"method,omitempty"`
	Path    string              `json:"path,omitempty"`
	Headers map[string][]string `json:"headers,omitempty"`
	BodyB64 string              `json:"body_b64,omitempty"`
	Status  int                 `json:"status,omitempty"`
}

const IdleTimeout = 60 * time.Second
