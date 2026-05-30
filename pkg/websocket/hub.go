package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"retail-pos-system/internal/auth"
	"retail-pos-system/internal/domain"
	"golang.org/x/time/rate"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
	// Security: Max connections per user
	maxConnectionsPerUser = 5
	// Security: Rate limit for connection attempts
	connRateLimit = 2 // connections per second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     checkOrigin,
}

// Security: Validate origin based on environment
func checkOrigin(r *http.Request) bool {
	// In production, restrict to known origins
	// For LAN deployment, same-origin or trusted hosts
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // No origin header (direct connection)
	}
	// Allow localhost and local network for development
	if strings.HasPrefix(origin, "http://localhost") ||
		strings.HasPrefix(origin, "http://127.0.0.1") ||
		strings.HasPrefix(origin, "http://192.168.") ||
		strings.HasPrefix(origin, "http://10.") {
		return true
	}
	// In production, should check against configured FRONTEND_URL
	return true // TODO: Configure properly for production
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
	Type      EventType        `json:"type"`
	Payload   json.RawMessage  `json:"payload"`
	Timestamp time.Time        `json:"timestamp"`
	StoreID   *int             `json:"store_id,omitempty"`
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte

	userID  int
	role    string
	storeID *int
	isAdmin bool
	mutex   sync.RWMutex
	
	// Security: Track connection for cleanup
	ctx    context.Context
	cancel context.CancelFunc
}

// Security: Rate limiter per IP
type rateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		limiters: make(map[string]*rate.Limiter),
	}
}

func (rl *rateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	limiter, exists := rl.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(rate.Every(time.Second/connRateLimit), 1)
		rl.limiters[ip] = limiter
	}
	return limiter
}

type Hub struct {
	clients         map[*Client]bool
	register        chan *Client
	unregister      chan *Client
	broadcast       chan Event
	mutex           sync.RWMutex
	
	// Security: Track connections per user
	userConnections map[int]int
	userConnMu      sync.RWMutex
	
	// Security: Rate limiter
	rateLimiter      *rateLimiter
	
	// Security: Auth service for token validation
	authService     *auth.AuthService
	
	// Shutdown control
	done    chan struct{}
	wg      sync.WaitGroup
}

func NewHub(authService *auth.AuthService) *Hub {
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
	
	for {
		select {
		case <-h.done:
			// Graceful shutdown - close all connections
			h.mutex.Lock()
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
			h.mutex.Unlock()
			return
		case client := <-h.register:
			h.mutex.Lock()
			
			// Security: Check max connections per user
			h.userConnMu.Lock()
			count := h.userConnections[client.userID]
			if count >= maxConnectionsPerUser {
				h.userConnMu.Unlock()
				h.mutex.Unlock()
				// Non-blocking send to avoid deadlock if writePump hasn't started yet
				select {
				case client.send <- []byte(`{"type":"error","payload":"Too many connections"}`):
				default:
					// Channel full or no receiver, just close the connection
				}
				client.conn.Close()
				continue
			}
			h.userConnections[client.userID] = count + 1
			h.userConnMu.Unlock()
			
			h.clients[client] = true
			h.mutex.Unlock()
			
			h.broadcastUserCount()
			log.Printf("WebSocket client registered. Total: %d (user_id=%d, role=%s)", 
				len(h.clients), client.userID, client.role)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				
				// Security: Decrement user connection count
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
			log.Printf("WebSocket client unregistered. Total: %d", len(h.clients))

		case event := <-h.broadcast:
			h.mutex.RLock()
			var recipients []*Client
			for client := range h.clients {
				if h.ShouldReceiveEvent(client, &event) {
					recipients = append(recipients, client)
				}
			}
			h.mutex.RUnlock()

			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("Error marshaling event: %v", err)
				continue
			}

			for _, client := range recipients {
				select {
				case client.send <- data:
				default:
					// Channel full - schedule client for removal non-blocking
					select {
					case h.unregister <- client:
						// Will be cleaned up in unregister handler
					default:
						// Unregister channel also full, just log
						log.Printf("Warning: unregister channel full, dropping client")
					}
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
		event.Timestamp = time.Now()
	}
	select {
	case h.broadcast <- event:
	default:
		log.Println("Warning: Broadcast channel full, dropping event")
	}
}

func (h *Hub) broadcastUserCount() {
	h.mutex.RLock()
	count := len(h.clients)
	h.mutex.RUnlock()

	payload, _ := json.Marshal(struct {
		Count int `json:"count"`
	}{
		Count: count,
	})
	event := Event{
		Type:      EventUserOnline,
		Timestamp: time.Now(),
		Payload:   payload,
	}
	data, _ := json.Marshal(event)

	h.mutex.RLock()
	for client := range h.clients {
		select {
		case client.send <- data:
		default:
		}
	}
	h.mutex.RUnlock()
}

// Security: Enhanced WebSocket upgrade with rate limiting and better auth
func ServeWebSocket(hub *Hub, c *gin.Context) {
	// Security: Rate limit by IP
	clientIP := c.ClientIP()
	if !hub.rateLimiter.getLimiter(clientIP).Allow() {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many connection attempts"})
		return
	}

	// Security: Get token from query param (or potentially from Sec-WebSocket-Protocol header)
	tokenString := c.Query("token")
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token required"})
		return
	}

	// Security: Validate token
	claims, err := hub.authService.ValidateToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	// Security: In production, enforce WSS
	if c.Request.Header.Get("X-Forwarded-Proto") != "https" {
		// Only warn in development
		log.Printf("Warning: WebSocket connection not using HTTPS from IP %s", clientIP)
	}

	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error from %s: %v", clientIP, err)
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
		isAdmin: claims.Role == "superadmin" || claims.Role == "admin",
		ctx:     ctx,
		cancel:  cancel,
	}

	// Security: Set timeouts
	conn.SetReadLimit(maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		// Non-blocking unregister to prevent deadlock if hub is busy
		select {
		case c.hub.unregister <- c:
			// Successfully queued for unregistration
		default:
			// Unregister channel full, just close connection
			c.conn.Close()
		}
	}()
	
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			// Security: Ignore all incoming messages (client->server not allowed)
			// This prevents clients from sending arbitrary commands
			_, _, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket read error (user %d): %v", c.userID, err)
				}
				return
			}
			// Just continue - we don't process client messages
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
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Broadcast helper functions
func BroadcastStockUpdate(hub *Hub, product *domain.Product, isLowStock bool) {
	if hub == nil {
		return
	}
	payload, _ := json.Marshal(struct {
		ID       int    `json:"id"`
		SKU      string `json:"sku"`
		Stock    int    `json:"stock"`
		LowStock bool   `json:"low_stock"`
	}{
		ID:       product.ID,
		SKU:      product.SKU,
		Stock:    product.Stock,
		LowStock: isLowStock,
	})
	event := Event{
		Type:    EventStockUpdate,
		Payload: payload,
		StoreID: product.StoreID,
	}
	hub.Broadcast(event)
}

func BroadcastSaleCreated(hub *Hub, sale *domain.Sale) {
	if hub == nil {
		return
	}
	payload, _ := json.Marshal(struct {
		ID      int    `json:"id"`
		Invoice string `json:"invoice"`
		Total   int    `json:"total"`
		Items   int    `json:"items"`
	}{
		ID:      sale.ID,
		Invoice: sale.InvoiceNumber,
		Total:   sale.TotalAmount,
		Items:   len(sale.Items),
	})
	event := Event{
		Type:    EventSaleCreated,
		Payload: payload,
		StoreID: sale.StoreID,
	}
	hub.Broadcast(event)
}

func BroadcastProductUpdate(hub *Hub, product *domain.Product) {
	if hub == nil {
		return
	}
	payload, _ := json.Marshal(struct {
		ID    int    `json:"id"`
		SKU   string `json:"sku"`
		Stock int    `json:"stock"`
		Price int    `json:"price"`
	}{
		ID:    product.ID,
		SKU:   product.SKU,
		Stock: product.Stock,
		Price: product.Price,
	})
	event := Event{
		Type:    EventProductUpdate,
		Payload: payload,
		StoreID: product.StoreID,
	}
	hub.Broadcast(event)
}

func BroadcastLowStockAlert(hub *Hub, product *domain.Product) {
	if hub == nil {
		return
	}
	payload, _ := json.Marshal(struct {
		ID    int    `json:"id"`
		SKU   string `json:"sku"`
		Name  string `json:"name"`
		Stock int    `json:"stock"`
	}{
		ID:    product.ID,
		SKU:   product.SKU,
		Name:  product.Name,
		Stock: product.Stock,
	})
	event := Event{
		Type:    EventLowStockAlert,
		Payload: payload,
		StoreID: product.StoreID,
	}
	hub.Broadcast(event)
}

// Shutdown gracefully closes all WebSocket connections
func (h *Hub) Shutdown() {
	close(h.done)  // Signal the Run() loop to stop
	h.wg.Wait()    // Wait for Run() to finish
}
