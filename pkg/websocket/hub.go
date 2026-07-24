package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

func getJakartaLoc() *time.Location {
	return shared.JakartaLocation()
}

const (
	writeWait             = 10 * time.Second
	pongWait              = 60 * time.Second
	pingPeriod            = (pongWait * 9) / 10
	maxMessageSize        = 512
	maxConnectionsPerUser = 5
	connRateLimit         = 2
	rateLimiterCleanupInt = 10 * time.Minute
	rateLimiterIdleTTL    = 30 * time.Minute
)

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	if origin == "http://localhost:5173" ||
		origin == "http://localhost:9095" ||
		origin == "http://127.0.0.1:5173" ||
		origin == "http://127.0.0.1:9095" {
		return true
	}
	allowedOrigin := os.Getenv("CORS_ORIGIN")
	if allowedOrigin != "" && origin == allowedOrigin {
		return true
	}
	return false
}

func newUpgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     checkOrigin,
	}
}

type EventType string

const (
	EventStockUpdate   EventType = "stock_update"
	EventSaleCreated   EventType = "sale_created"
	EventLowStockAlert EventType = "low_stock_alert"
	EventProductUpdate EventType = "product_updated"
	EventUserOnline    EventType = "user_online_count"
)

type Event struct {
	Type      EventType       `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
	StoreID   *int            `json:"store_id,omitempty"`
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte

	userID  int
	role    string
	storeID *int
	isAdmin bool

	ctx    context.Context
	cancel context.CancelFunc
}

type TokenValidator interface {
	ValidateToken(tokenString string) (*Claims, error)
}

type Claims struct {
	ID       int
	Role     string
	StoreID  *int
	Username string
}

type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type rateLimiter struct {
	limiters map[string]*rateLimiterEntry
	mu       sync.RWMutex
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		limiters: make(map[string]*rateLimiterEntry),
	}
}

func (rl *rateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.limiters[ip]
	if !exists {
		limiter := rate.NewLimiter(rate.Every(time.Second/connRateLimit), 1)
		entry = &rateLimiterEntry{limiter: limiter, lastSeen: time.Now()}
		rl.limiters[ip] = entry
	} else {
		entry.lastSeen = time.Now()
	}
	return entry.limiter
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	threshold := time.Now().Add(-rateLimiterIdleTTL)
	for ip, entry := range rl.limiters {
		if entry.lastSeen.Before(threshold) {
			delete(rl.limiters, ip)
		}
	}
}

type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan Event
	mutex      sync.RWMutex

	userConnections map[int]int
	userConnMu      sync.RWMutex

	rateLimiter *rateLimiter

	authService TokenValidator

	done chan struct{}
	wg   sync.WaitGroup
}

func NewHub(authService TokenValidator) *Hub {
	return &Hub{
		register:        make(chan *Client, 100),
		unregister:      make(chan *Client, 100),
		broadcast:       make(chan Event, 1000),
		clients:         make(map[*Client]bool),
		userConnections: make(map[int]int),
		rateLimiter:     newRateLimiter(),
		authService:     authService,
		done:            make(chan struct{}),
	}
}

func (h *Hub) Run() {
	h.wg.Add(1)
	defer h.wg.Done()

	cleanupTicker := time.NewTicker(rateLimiterCleanupInt)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-cleanupTicker.C:
			h.rateLimiter.cleanup()
		case <-h.done:
			h.mutex.Lock()
			h.userConnMu.Lock()
			for client := range h.clients {
				if client.cancel != nil {
					client.cancel()
				}
				close(client.send)
				if client.conn != nil {
					client.conn.Close()
				}
				delete(h.clients, client)
			}
			h.userConnections = make(map[int]int)
			h.userConnMu.Unlock()
			h.mutex.Unlock()
			return
		case client := <-h.register:
			h.mutex.Lock()

			h.userConnMu.Lock()
			count := h.userConnections[client.userID]
			if count >= maxConnectionsPerUser {
				h.userConnMu.Unlock()
				h.mutex.Unlock()
				select {
				case client.send <- []byte(`{"type":"error","payload":"Too many connections"}`):
				default:
				}
				client.conn.Close()
				continue
			}
			h.userConnections[client.userID] = count + 1
			h.userConnMu.Unlock()

			h.clients[client] = true
			h.mutex.Unlock()

			h.broadcastUserCount()
			slog.Info("WebSocket client registered", "total", len(h.clients), "user_id", client.userID, "role", client.role)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)

				h.userConnMu.Lock()
				if count := h.userConnections[client.userID]; count > 0 {
					h.userConnections[client.userID] = count - 1
				}
				h.userConnMu.Unlock()
			}
			h.mutex.Unlock()

			if client.cancel != nil {
				client.cancel()
			}
			h.broadcastUserCount()
			slog.Info("WebSocket client unregistered", "total", len(h.clients))

		case event := <-h.broadcast:
			data, err := json.Marshal(event)
			if err != nil {
				slog.Error("error marshaling event", "error", err)
				continue
			}

			h.mutex.RLock()
			recipients := make([]*Client, 0, len(h.clients))
			for client := range h.clients {
				if h.ShouldReceiveEvent(client, &event) {
					recipients = append(recipients, client)
				}
			}
			h.mutex.RUnlock()

			for _, client := range recipients {
				select {
				case client.send <- data:
				default:
					go func(c *Client) {
						select {
						case h.unregister <- c:
						default:
							slog.Warn("unregister channel full, dropping client", "user_id", c.userID)
						}
					}(client)
				}
			}
		}
	}
}

func (h *Hub) ShouldReceiveEvent(client *Client, event *Event) bool {
	if event.StoreID != nil && client.storeID != nil {
		if *event.StoreID != *client.storeID && !client.isAdmin {
			return false
		}
	}
	return true
}

func (h *Hub) Broadcast(event Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().In(getJakartaLoc())
	}
	select {
	case h.broadcast <- event:
	default:
		slog.Warn("Broadcast channel full, dropping event")
	}
}

func (h *Hub) broadcastUserCount() {
	h.mutex.RLock()
	count := len(h.clients)
	clients := make([]*Client, 0, count)
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mutex.RUnlock()

	payload, _ := json.Marshal(struct {
		Count int `json:"count"`
	}{
		Count: count,
	})
	event := Event{
		Type:      EventUserOnline,
		Timestamp: time.Now().In(getJakartaLoc()),
		Payload:   payload,
	}
	data, _ := json.Marshal(event)

	for _, client := range clients {
		select {
		case client.send <- data:
		default:
		}
	}
}

type authMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

func ServeWebSocket(hub *Hub, c *gin.Context) {
	clientIP := c.Request.RemoteAddr
	if host, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
		clientIP = host
	}

	if !hub.rateLimiter.getLimiter(clientIP).Allow() {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many connection attempts"})
		return
	}

	if !strings.Contains(c.Request.Host, "localhost") && c.Request.Header.Get("X-Forwarded-Proto") != "https" {
		slog.Warn("WebSocket connection not using HTTPS", "ip", clientIP)
	}

	conn, err := newUpgrader().Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Warn("WebSocket upgrade error", "ip", clientIP, "error", err)
		return
	}

	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		slog.Warn("WebSocket set auth deadline error", "error", err)
		conn.Close()
		return
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		slog.Warn("WebSocket auth message read error", "ip", clientIP, "error", err)
		conn.Close()
		return
	}

	var authMsg authMessage
	if err := json.Unmarshal(msg, &authMsg); err != nil || authMsg.Type != "auth" || authMsg.Token == "" {
		slog.Warn("WebSocket invalid auth message format", "ip", clientIP)
		conn.Close()
		return
	}

	claims, err := hub.authService.ValidateToken(authMsg.Token)
	if err != nil {
		slog.Warn("WebSocket auth failed", "ip", clientIP, "error", err)
		conn.Close()
		return
	}

	slog.Info("WebSocket auth OK", "user", claims.ID, "role", claims.Role, "store", claims.StoreID, "ip", clientIP)

	var storeID *int
	if claims.StoreID != nil {
		sid := *claims.StoreID
		storeID = &sid
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		hub:     hub,
		conn:    conn,
		send:    make(chan []byte, 256),
		userID:  claims.ID,
		role:    claims.Role,
		storeID: storeID,
		isAdmin: claims.Role == "superadmin" || claims.Role == "admin",
		ctx:     ctx,
		cancel:  cancel,
	}

	conn.SetReadLimit(maxMessageSize)
	if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		slog.Warn("WebSocket set read deadline error", "error", err)
	}
	conn.SetPongHandler(func(string) error {
		if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			slog.Warn("WebSocket set read deadline (pong) error", "error", err)
		}
		return nil
	})

	hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		select {
		case c.hub.unregister <- c:
		default:
			c.conn.Close()
		}
	}()

	go func() {
		<-c.ctx.Done()
		_ = c.conn.SetReadDeadline(time.Now())
	}()

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Warn("WebSocket read error", "user", c.userID, "error", err)
			}
			return
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				slog.Warn("WebSocket set write deadline error", "user", c.userID, "error", err)
				return
			}
			if !ok {
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					slog.Warn("WebSocket write close message error", "user", c.userID, "error", err)
				}
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				return
			}
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				slog.Warn("WebSocket set write deadline (ticker) error", "user", c.userID, "error", err)
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type StockUpdateEvent struct {
	ID       int    `json:"id"`
	SKU      string `json:"sku"`
	Stock    int    `json:"stock"`
	LowStock bool   `json:"low_stock"`
	StoreID  *int   `json:"-"`
}

func BroadcastStockUpdate(hub *Hub, event StockUpdateEvent) {
	if hub == nil {
		return
	}
	payload, _ := json.Marshal(event)
	hub.Broadcast(Event{
		Type:    EventStockUpdate,
		Payload: payload,
		StoreID: event.StoreID,
	})
}

type SaleCreatedEvent struct {
	ID      int    `json:"id"`
	Invoice string `json:"invoice"`
	Total   int    `json:"total"`
	Items   int    `json:"items"`
	StoreID *int   `json:"-"`
}

func BroadcastSaleCreated(hub *Hub, event SaleCreatedEvent) {
	if hub == nil {
		return
	}
	payload, _ := json.Marshal(event)
	hub.Broadcast(Event{
		Type:    EventSaleCreated,
		Payload: payload,
		StoreID: event.StoreID,
	})
}

type ProductUpdateEvent struct {
	ID      int    `json:"id"`
	SKU     string `json:"sku"`
	Stock   int    `json:"stock"`
	Price   int    `json:"price"`
	StoreID *int   `json:"-"`
}

func BroadcastProductUpdate(hub *Hub, event ProductUpdateEvent) {
	if hub == nil {
		return
	}
	payload, _ := json.Marshal(event)
	hub.Broadcast(Event{
		Type:    EventProductUpdate,
		Payload: payload,
		StoreID: event.StoreID,
	})
}

type LowStockAlertEvent struct {
	ID      int    `json:"id"`
	SKU     string `json:"sku"`
	Name    string `json:"name"`
	Stock   int    `json:"stock"`
	StoreID *int   `json:"-"`
}

func BroadcastLowStockAlert(hub *Hub, event LowStockAlertEvent) {
	if hub == nil {
		return
	}
	payload, _ := json.Marshal(event)
	hub.Broadcast(Event{
		Type:    EventLowStockAlert,
		Payload: payload,
		StoreID: event.StoreID,
	})
}

func (h *Hub) Shutdown() {
	close(h.done)
	h.wg.Wait()
}
