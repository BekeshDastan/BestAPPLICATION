package handler

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	userv1 "github.com/bekesh/social/gen/go/user/v1"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WsEvent is the JSON envelope for all WebSocket messages.
type WsEvent struct {
	Type           string      `json:"type"`
	ConversationID string      `json:"conversation_id,omitempty"`
	UserID         string      `json:"user_id,omitempty"`
	MessageID      string      `json:"message_id,omitempty"`
	Text           string      `json:"text,omitempty"`
	Payload        interface{} `json:"payload,omitempty"`
}

// wsClient represents a single connected browser tab.
type wsClient struct {
	userID string
	conn   *websocket.Conn
	send   chan []byte
}

// Hub routes WS events to the right conversation rooms.
// rooms[convID][userID] → client
type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[string]*wsClient
}

func NewHub() *Hub {
	return &Hub{rooms: make(map[string]map[string]*wsClient)}
}

func (h *Hub) join(convID, userID string, c *wsClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[convID] == nil {
		h.rooms[convID] = make(map[string]*wsClient)
	}
	h.rooms[convID][userID] = c
}

func (h *Hub) leave(convID, userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if room := h.rooms[convID]; room != nil {
		delete(room, userID)
		if len(room) == 0 {
			delete(h.rooms, convID)
		}
	}
}

func (h *Hub) leaveAll(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for convID, room := range h.rooms {
		delete(room, userID)
		if len(room) == 0 {
			delete(h.rooms, convID)
		}
	}
}

// Broadcast sends an event to every client in the conversation room.
func (h *Hub) Broadcast(convID string, ev WsEvent) {
	data, _ := json.Marshal(ev)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.rooms[convID] {
		select {
		case c.send <- data:
		default:
		}
	}
}

// ─── Handler ───────────────────────────────────────────────────────────────

type WsHandler struct {
	user     userv1.UserServiceClient
	hub      *Hub
	upgrader websocket.Upgrader
}

func NewWsHandler(user userv1.UserServiceClient, hub *Hub) *WsHandler {
	return &WsHandler{
		user: user,
		hub:  hub,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(_ *http.Request) bool { return true },
		},
	}
}

func (h *WsHandler) Handle(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}
	resp, err := h.user.ValidateToken(c.Request.Context(), &userv1.ValidateTokenRequest{AccessToken: token})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	userID := resp.UserId

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	client := &wsClient{
		userID: userID,
		conn:   conn,
		send:   make(chan []byte, 128),
	}
	defer h.hub.leaveAll(userID)

	// write pump
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case msg, ok := <-client.send:
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if !ok {
					conn.WriteMessage(websocket.CloseMessage, nil)
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	// read pump
	conn.SetReadLimit(8192)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		var ev WsEvent
		if json.Unmarshal(raw, &ev) != nil {
			continue
		}

		switch ev.Type {
		case "join":
			if ev.ConversationID != "" {
				h.hub.join(ev.ConversationID, userID, client)
			}
		case "leave":
			if ev.ConversationID != "" {
				h.hub.leave(ev.ConversationID, userID)
			}
		case "typing":
			if ev.ConversationID != "" {
				h.hub.Broadcast(ev.ConversationID, WsEvent{
					Type:           "typing",
					ConversationID: ev.ConversationID,
					UserID:         userID,
				})
			}
		}
	}
}
