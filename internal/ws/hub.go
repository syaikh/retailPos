package ws

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: checkOrigin,
}

type Event struct {
	Type      string `json:"type"`
	Payload   any    `json:"payload"`
	StoreID   string `json:"store_id,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

type Message struct {
	Client   *websocket.Conn
	Event    Event
	Response chan []byte
}

type Register struct {
	Client   *websocket.Conn
	StoreID  string
	UserID   string
	Response chan error
}

type Unregister struct {
	Client *websocket.Conn
}

type StatsRequest struct {
	Response chan Stats
}

type Stats struct {
	ClientCount int32 `json:"client_count"`
}

type Hub struct {
	clients        map[*websocket.Conn]clientInfo
	register       chan Register
	unregister     chan Unregister
	broadcast      chan Message
	stats          chan StatsRequest
	eventWhitelist map[string]bool
	clientStore    map[*websocket.Conn]string
	clientUserID   map[*websocket.Conn]string
	clientCount    atomic.Int32
}

type clientInfo struct {
	storeID string
	userID  string
}

func NewHub() *Hub {
	whitelist := map[string]bool{
		"sale.created":  true,
		"stock.updated": true,
	}

	return &Hub{
		clients:        make(map[*websocket.Conn]clientInfo),
		register:       make(chan Register),
		unregister:     make(chan Unregister),
		broadcast:      make(chan Message, 256),
		stats:          make(chan StatsRequest, 8),
		eventWhitelist: whitelist,
		clientStore:    make(map[*websocket.Conn]string),
		clientUserID:   make(map[*websocket.Conn]string),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case reg := <-h.register:
			conn := reg.Client
			h.clients[conn] = clientInfo{
				storeID: reg.StoreID,
				userID:  reg.UserID,
			}
			h.clientStore[conn] = reg.StoreID
			h.clientUserID[conn] = reg.UserID
			h.clientCount.Add(1)

			log.Printf("[WS] client registered: store=%s user=%s total=%d",
				reg.StoreID, reg.UserID, h.clientCount.Load())
			reg.Response <- nil

		case reg := <-h.unregister:
			conn := reg.Client
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				delete(h.clientStore, conn)
				delete(h.clientUserID, conn)
				h.clientCount.Add(-1)
				log.Printf("[WS] client disconnected. total=%d", h.clientCount.Load())
			}

		case msg := <-h.broadcast:
			if !h.eventWhitelist[msg.Event.Type] {
				log.Printf("[WS] event not in whitelist: %s", msg.Event.Type)
				continue
			}

			targetStoreID := msg.Event.StoreID
			for client, info := range h.clients {
				if targetStoreID != "" && info.storeID != targetStoreID {
					continue
				}

				if err := client.WriteJSON(msg.Event); err != nil {
					log.Printf("[WS] write error: %v", err)
					client.Close()
					h.unregister <- Unregister{Client: client}
				}
			}

		case req := <-h.stats:
			req.Response <- Stats{ClientCount: h.clientCount.Load()}
		}
	}
}

func (h *Hub) Broadcast(storeID, eventType string, payload any) error {
	if !h.eventWhitelist[eventType] {
		log.Printf("[WS] broadcast blocked (not whitelisted): %s", eventType)
		return nil
	}

	msg := Message{
		Event: Event{
			Type:      eventType,
			Payload:   payload,
			StoreID:   storeID,
			Timestamp: time.Now().Unix(),
		},
	}

	select {
	case h.broadcast <- msg:
	default:
		log.Printf("[WS] broadcast channel full, dropping event: %s", eventType)
	}
	return nil
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Debug mode bypass
	if os.Getenv("GIN_MODE") == "debug" {
		h.handleWebSocket(w, r, "", "")
		return
	}

	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		authHeader := r.Header.Get("Authorization")
		tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
	}
	// Fallback to session_token cookie
	if tokenStr == "" {
		if cookie, err := r.Cookie("session_token"); err == nil && cookie.Value != "" {
			tokenStr = cookie.Value
		}
	}

	if tokenStr == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	claims, err := validateToken(tokenStr)
	if err != nil {
		http.Error(w, "invalid token: "+err.Error(), http.StatusUnauthorized)
		return
	}

	storeID := r.URL.Query().Get("store_id")
	deviceID := r.URL.Query().Get("device_id")
	_ = deviceID // unused for now
	userID := fmt.Sprintf("%v", claims["user_id"])

	h.handleWebSocket(w, r, storeID, userID)
}

func validateToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

func checkOrigin(r *http.Request) bool {
	if os.Getenv("GIN_MODE") == "debug" {
		return true
	}
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000,http://localhost:5173"
	}
	allowed := strings.Split(allowedOrigins, ",")
	origin := r.Header.Get("Origin")
	for _, a := range allowed {
		if strings.TrimSpace(a) == origin {
			return true
		}
	}
	log.Printf("[WS] origin rejected: %s", origin)
	return false
}

func (h *Hub) handleWebSocket(w http.ResponseWriter, r *http.Request, storeID, userID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] upgrade error: %v", err)
		return
	}

	conn.SetReadLimit(1 << 20)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(60 * time.Second))

	reg := Register{
		Client:   conn,
		StoreID:  storeID,
		UserID:   userID,
		Response: make(chan error, 1),
	}

	h.register <- reg
	<-reg.Response

	pingTicker := time.NewTicker(54 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-r.Context().Done():
			h.unregister <- Unregister{Client: conn}
			conn.Close()
			return

		case <-pingTicker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				h.unregister <- Unregister{Client: conn}
				conn.Close()
				return
			}

		default:
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			_, _, err := conn.ReadMessage()
			if err != nil {
				h.unregister <- Unregister{Client: conn}
				conn.Close()
				return
			}
		}
	}
}

func clientKey(conn *websocket.Conn) string {
	if addr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		return strconv.Itoa(addr.Port)
	}
	return "unknown"
}
