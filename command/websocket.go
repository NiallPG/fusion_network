package command

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type WebSocketHub struct {
	clients   map[*websocket.Conn]bool
	clientsMu sync.RWMutex
	upgrader  websocket.Upgrader
}

type ThreatMessage struct {
	Type       string  `json:"type"`
	ID         int     `json:"id"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Level      int     `json:"level"`
	Confidence float64 `json:"confidence"`
	Sensors    int     `json:"sensors"`
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for dev
			},
		},
	}
}

func (h *WebSocketHub) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	h.clientsMu.Lock()
	h.clients[conn] = true
	h.clientsMu.Unlock()

	log.Printf("WebSocket client connected: %s", conn.RemoteAddr())

	// Keep connection alive, handle disconnect
	defer func() {
		h.clientsMu.Lock()
		delete(h.clients, conn)
		h.clientsMu.Unlock()
		conn.Close()
		log.Printf("WebSocket client disconnected: %s", conn.RemoteAddr())
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (h *WebSocketHub) BroadcastThreat(threat *FusedThreat) {
	msg := ThreatMessage{
		Type:       "threat_update",
		ID:         threat.ID,
		X:          threat.X,
		Y:          threat.Y,
		Level:      threat.Level,
		Confidence: threat.Confidence,
		Sensors:    threat.SensorCount,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	for client := range h.clients {
		err := client.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Printf("WebSocket write error: %v", err)
		}
	}
}

func (h *WebSocketHub) Start(port string) {
	http.HandleFunc("/ws", h.HandleConnection)
	log.Printf("WebSocket server listening on %s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("WebSocket server error: %v", err)
	}
}
