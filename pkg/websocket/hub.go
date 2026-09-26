package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"retail-pos-system/internal/permissions"
	"retail-pos-system/internal/shared"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"
)

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

	allowedOrigins := map[string]bool{
		"http://localhost:5173": true,
		"http://localhost:9095": true,
		"http://127.0.0.1:5173": true,
		"http://127.0.0.1:9095": true,
	}
	if allowedOrigins[origin] {
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
	EventPOReceived    EventType = "po_received"
	EventPOCreated     EventType = "po_created"
	EventPOConfirmed   EventType = "po_confirmed"
	EventPOCancelled   EventType = "po_cancelled"
	EventSOCreated     EventType = "so_created"
	EventSOOpened      EventType = "so_opened"
	EventSOSubmitted   EventType = "so_submitted"
	EventSOApproved    EventType = "so_approved"
	EventSOPosted      EventType = "so_posted"
	EventSOClosed      EventType = "so_closed"
	EventSORejected    EventType = "so_rejected"
	EventSORecount     EventType = "so_needs_recount"
	EventSOCancelled   EventType = "so_cancelled"
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
	ip      string

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

	done    chan struct{}
	started chan struct{}
	wg      sync.WaitGroup
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
		started:         make(chan struct{}),
	}
}

func (h *Hub) Run() {
	h.wg.Add(1)
	defer h.wg.Done()
	close(h.started)

	cleanupTicker := time.NewTicker(rateLimiterCleanupInt)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-cleanupTicker.C:
			h.rateLimiter.cleanup()
		case <-h.done:
			h.mutex.Lock()
			h.userConnMu.Lock()
			slog.Debug("WebSocket hub shutting down", "clients", len(h.clients))
			for client := range h.clients {
				if client.cancel != nil {
					client.cancel()
				}
				close(client.send)
				if client.conn != nil {
					_ = client.conn.Close()
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
				slog.Warn("WebSocket connection rejected: per-user limit reached",
					"user_id", client.userID, "ip", client.ip, "limit", maxConnectionsPerUser)
				select {
				case client.send <- []byte(`{"type":"error","payload":"Too many connections"}`):
				default:
				}
				_ = client.conn.Close()
				continue
			}
			h.userConnections[client.userID] = count + 1
			h.userConnMu.Unlock()

			h.clients[client] = true
			connectedCount := len(h.clients)
			h.mutex.Unlock()

			h.broadcastUserCount()
			// Single lifecycle event per connect. The authenticated identity
			// (including the client IP, useful for auth diagnostics) is
			// reported here so the upgrade handler does not have to log it
			// again.
			slog.Info("WebSocket client connected",
				"user_id", client.userID, "role", client.role, "store_id", client.storeID, "ip", client.ip, "total", connectedCount)

		case client := <-h.unregister:
			h.mutex.Lock()
			// A client can reach h.unregister more than once (a slow-client
			// drop and readPump's deferred unregister both push it). Only the
			// first one owns the connection; later ones must not re-announce a
			// disconnect for a slot already accounted for.
			_, registered := h.clients[client]
			if registered {
				delete(h.clients, client)
				close(client.send)

				h.userConnMu.Lock()
				if count := h.userConnections[client.userID]; count > 0 {
					h.userConnections[client.userID] = count - 1
				}
				h.userConnMu.Unlock()
			}
			disconnectedCount := len(h.clients)
			h.mutex.Unlock()

			if client.cancel != nil {
				client.cancel()
			}
			h.broadcastUserCount()
			if registered {
				slog.Info("WebSocket client disconnected",
					"user_id", client.userID, "ip", client.ip, "total", disconnectedCount)
			}

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
		event.Timestamp = time.Now().In(shared.JakartaLocation())
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
		Timestamp: time.Now().In(shared.JakartaLocation()),
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

	// CSRF note: The WebSocket authenticates via a JWT sent as a WebSocket message body
	// after upgrade (not a cookie). CSRF attacks rely on cookie-based auth being sent
	// automatically by the browser. Since the JWT is stored in sessionStorage and sent
	// explicitly as a message, cross-origin requests cannot forge a valid WebSocket session.
	// The Origin header is additionally validated by the gorilla/websocket upgrader via
	// checkOrigin() for defense-in-depth.

	conn, err := newUpgrader().Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Warn("WebSocket upgrade error", "ip", clientIP, "error", err)
		return
	}

	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		slog.Warn("WebSocket set auth deadline error", "error", err)
		_ = conn.Close()
		return
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		slog.Warn("WebSocket auth message read error", "ip", clientIP, "error", err)
		_ = conn.Close()
		return
	}

	var authMsg authMessage
	if err := json.Unmarshal(msg, &authMsg); err != nil || authMsg.Type != "auth" || authMsg.Token == "" {
		slog.Warn("WebSocket invalid auth message format", "ip", clientIP)
		_ = conn.Close()
		return
	}

	claims, err := hub.authService.ValidateToken(authMsg.Token)
	if err != nil {
		slog.Warn("WebSocket auth failed", "ip", clientIP, "error", err)
		_ = conn.Close()
		return
	}

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
		isAdmin: claims.Role == permissions.RoleSuperadmin || claims.Role == permissions.RoleManager,
		ip:      clientIP,
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

	hub.wg.Add(2)
	hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer c.hub.wg.Done()
	defer func() {
		select {
		case c.hub.unregister <- c:
		default:
			_ = c.conn.Close()
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

// isExpectedCloseError reports whether err is a normal end-of-connection
// condition rather than a fault. gorilla returns ErrCloseSent once a close
// frame has already been written — which is exactly what happens when the hub
// unregisters a client during shutdown — and the net package reports
// net.ErrClosed once the socket is gone. Neither is actionable, so a close
// write that fails with either is left unreported; any other error on the
// close-frame write is still surfaced by the caller.
func isExpectedCloseError(err error) bool {
	return errors.Is(err, websocket.ErrCloseSent) || errors.Is(err, net.ErrClosed)
}

func (c *Client) writePump() {
	defer c.hub.wg.Done()
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
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
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil && !isExpectedCloseError(err) {
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

type POReceivedEvent struct {
	POID     int    `json:"po_id"`
	PONumber string `json:"po_number"`
	GRNumber string `json:"gr_number"`
	StoreID  *int   `json:"-"`
}

func BroadcastPOReceived(hub *Hub, event POReceivedEvent) {
	if hub == nil {
		return
	}
	payload, _ := json.Marshal(event)
	hub.Broadcast(Event{
		Type:    EventPOReceived,
		Payload: payload,
		StoreID: event.StoreID,
	})
}

type POCreatedEvent struct {
	POID     int    `json:"po_id"`
	PONumber string `json:"po_number"`
	StoreID  *int   `json:"-"`
}

func BroadcastPOCreated(hub *Hub, event POCreatedEvent) {
	if hub == nil {
		return
	}
	payload, _ := json.Marshal(event)
	hub.Broadcast(Event{
		Type:    EventPOCreated,
		Payload: payload,
		StoreID: event.StoreID,
	})
}

type POConfirmedEvent struct {
	POID     int    `json:"po_id"`
	PONumber string `json:"po_number"`
	StoreID  *int   `json:"-"`
}

func BroadcastPOConfirmed(hub *Hub, event POConfirmedEvent) {
	if hub == nil {
		return
	}
	payload, _ := json.Marshal(event)
	hub.Broadcast(Event{
		Type:    EventPOConfirmed,
		Payload: payload,
		StoreID: event.StoreID,
	})
}

type POCancelledEvent struct {
	POID     int    `json:"po_id"`
	PONumber string `json:"po_number"`
	StoreID  *int   `json:"-"`
}

func BroadcastPOCancelled(hub *Hub, event POCancelledEvent) {
	if hub == nil {
		return
	}
	payload, _ := json.Marshal(event)
	hub.Broadcast(Event{
		Type:    EventPOCancelled,
		Payload: payload,
		StoreID: event.StoreID,
	})
}

type StockOpnameStatusEvent struct {
	SessionID     int    `json:"session_id"`
	SessionNumber string `json:"session_number"`
	Status        string `json:"status"`
	StoreID       *int   `json:"-"`
}

func BroadcastStockOpnameStatus(hub *Hub, eventType EventType, event StockOpnameStatusEvent) {
	if hub == nil {
		return
	}
	payload, _ := json.Marshal(event)
	hub.Broadcast(Event{
		Type:    eventType,
		Payload: payload,
		StoreID: event.StoreID,
	})
}

func (h *Hub) Shutdown() {
	<-h.started
	close(h.done)
	h.wg.Wait()
}
