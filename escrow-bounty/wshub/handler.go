package wshub

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/grayfalcon666/escrow-bounty/token"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, validate Origin header
		return true
	},
}

// Handler is an HTTP handler that upgrades WebSocket connections.
type Handler struct {
	hub       *Hub
	tokenMaker *token.JWTMaker
}

// NewHandler creates a new WebSocket handler.
func NewHandler(hub *Hub, tokenMaker *token.JWTMaker) *Handler {
	return &Handler{
		hub:       hub,
		tokenMaker: tokenMaker,
	}
}

// ServeHTTP handles WebSocket upgrade requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract token from query parameter: ?token=Bearer xxx
	tokenStr := r.URL.Query().Get("token")
	tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
	tokenStr = strings.TrimSpace(tokenStr)

	log.Printf("WS: incoming connection from %s, token present: %v", r.RemoteAddr, tokenStr != "")

	if tokenStr == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	payload, err := h.tokenMaker.VerifyToken(tokenStr)
	if err != nil {
		log.Printf("WS: token verification failed: %v", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	log.Printf("WS: user %s authenticated successfully", payload.Username)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS: failed to upgrade connection: %v", err)
		return
	}

	client := NewClient(h.hub, conn, payload.Username)
	h.hub.register <- client

	log.Printf("WS: client registered for user %s", payload.Username)
	go client.writePump()
	go client.readPump()
}
