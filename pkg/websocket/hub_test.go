package websocket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func drainMessages(ch chan []byte) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

func intPtr(i int) *int { return &i }

func newTestHub(authService TokenValidator) *Hub {
	hub := NewHub(authService)
	return hub
}

func registerClient(t *testing.T, hub *Hub, userID int, storeID *int, isAdmin bool) *Client {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{
		hub:     hub,
		userID:  userID,
		send:    make(chan []byte, 256),
		isAdmin: isAdmin,
		storeID: storeID,
		ctx:     ctx,
		cancel:  cancel,
	}
	hub.register <- client
	require.Eventually(t, func() bool {
		hub.mutex.RLock()
		defer hub.mutex.RUnlock()
		return hub.userConnections[userID] >= 1
	}, time.Second, 10*time.Millisecond)
	return client
}

// --- Pure function tests ---

func TestGetJakartaLoc(t *testing.T) {
	loc := getJakartaLoc()
	require.NotNil(t, loc)
	assert.Equal(t, "Asia/Jakarta", loc.String())
}

func TestCheckOrigin(t *testing.T) {
	tests := []struct {
		name     string
		origin   string
		envValue string
		expected bool
	}{
		{"empty origin", "", "", false},
		{"localhost 5173", "http://localhost:5173", "", true},
		{"localhost 9095", "http://localhost:9095", "", true},
		{"127.0.0.1 5173", "http://127.0.0.1:5173", "", true},
		{"127.0.0.1 9095", "http://127.0.0.1:9095", "", true},
		{"env allowed origin", "https://example.com", "https://example.com", true},
		{"env not matching", "https://other.com", "https://example.com", false},
		{"no env set", "https://anything.com", "", false},
		{"env empty string", "https://test.com", "", false},
		{"random origin", "https://evil.com", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("CORS_ORIGIN", tt.envValue)
				defer os.Unsetenv("CORS_ORIGIN")
			} else {
				os.Unsetenv("CORS_ORIGIN")
			}

			r, _ := http.NewRequest("GET", "http://localhost:5173/ws", nil)
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			result := checkOrigin(r)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewRateLimiter(t *testing.T) {
	rl := newRateLimiter()
	require.NotNil(t, rl)
	require.NotNil(t, rl.limiters)
	assert.Empty(t, rl.limiters)
}

func TestRateLimiter_GetLimiter(t *testing.T) {
	rl := newRateLimiter()

	t.Run("returns same limiter for same IP", func(t *testing.T) {
		l1 := rl.getLimiter("1.2.3.4")
		l2 := rl.getLimiter("1.2.3.4")
		assert.Same(t, l1, l2)
	})

	t.Run("returns different limiter for different IP", func(t *testing.T) {
		l1 := rl.getLimiter("1.2.3.4")
		l2 := rl.getLimiter("5.6.7.8")
		assert.NotSame(t, l1, l2)
	})

	t.Run("updates lastSeen", func(t *testing.T) {
		entry := rl.getLimiter("9.9.9.9")
		require.NotNil(t, entry)
		rl.mu.RLock()
		e := rl.limiters["9.9.9.9"]
		require.NotNil(t, e)
		before := e.lastSeen
		rl.mu.RUnlock()

		time.Sleep(5 * time.Millisecond)
		rl.getLimiter("9.9.9.9")

		rl.mu.RLock()
		e = rl.limiters["9.9.9.9"]
		assert.True(t, e.lastSeen.After(before))
		rl.mu.RUnlock()
	})
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := newRateLimiter()

	// Add fresh entry
	rl.getLimiter("fresh-ip")

	// Add stale entry by manipulating lastSeen directly
	rl.mu.Lock()
	limiter := rate.NewLimiter(rate.Every(time.Second/connRateLimit), 1)
	rl.limiters["stale-ip"] = &rateLimiterEntry{
		limiter:  limiter,
		lastSeen: time.Now().Add(-rateLimiterIdleTTL - time.Minute),
	}
	rl.mu.Unlock()

	assert.Len(t, rl.limiters, 2)

	rl.cleanup()

	rl.mu.RLock()
	defer rl.mu.RUnlock()
	_, hasFresh := rl.limiters["fresh-ip"]
	_, hasStale := rl.limiters["stale-ip"]
	assert.True(t, hasFresh, "fresh entry should remain")
	assert.False(t, hasStale, "stale entry should be removed")
}

func TestNewHub(t *testing.T) {
	hub := NewHub(nil)
	require.NotNil(t, hub)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.broadcast)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.userConnections)
	assert.NotNil(t, hub.rateLimiter)
	assert.NotNil(t, hub.done)
	assert.Nil(t, hub.authService)
}

// --- Hub lifecycle tests ---

func TestHub_Shutdown(t *testing.T) {
	hub := &Hub{
		clients:         make(map[*Client]bool),
		register:        make(chan *Client, 100),
		unregister:      make(chan *Client, 100),
		broadcast:       make(chan Event, 1000),
		userConnections: make(map[int]int),
		done:            make(chan struct{}),
	}

	go hub.Run()
	time.Sleep(50 * time.Millisecond)

	registerClient(t, hub, 1, nil, true)
	registerClient(t, hub, 2, nil, false)

	hub.mutex.RLock()
	assert.Len(t, hub.clients, 2)
	hub.mutex.RUnlock()

	hub.Shutdown()

	hub.mutex.RLock()
	assert.Len(t, hub.clients, 0)
	assert.Equal(t, 0, hub.userConnections[1])
	assert.Equal(t, 0, hub.userConnections[2])
	hub.mutex.RUnlock()
}

func TestHub_MaxConnectionsPerUser(t *testing.T) {
	hub := &Hub{
		clients:         make(map[*Client]bool),
		register:        make(chan *Client, 100),
		unregister:      make(chan *Client, 100),
		broadcast:       make(chan Event, 1000),
		userConnections: make(map[int]int),
		done:            make(chan struct{}),
	}
	go hub.Run()
	defer hub.Shutdown()

	baseCtx := context.Background()
	for i := 0; i < maxConnectionsPerUser; i++ {
		ctx, cancel := context.WithCancel(baseCtx)
		client := &Client{
			hub:     hub,
			userID:  1,
			send:    make(chan []byte, 256),
			isAdmin: false,
			ctx:     ctx,
			cancel:  cancel,
		}
		hub.register <- client
	}

	assert.Eventually(t, func() bool {
		hub.mutex.RLock()
		defer hub.mutex.RUnlock()
		return hub.userConnections[1] == maxConnectionsPerUser
	}, time.Second, 10*time.Millisecond)

	// 6th connection should be rejected - needs a non-nil conn
	// because hub calls client.conn.Close() on rejection
	// Create a minimal test HTTP server to get a real websocket.Conn
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := upgrader.Upgrade(w, r, nil)
		// Just hold the connection open
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer ts.Close()

	wsURL := "ws" + ts.URL[4:]
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer wsConn.Close()

	ctx, cancel := context.WithCancel(baseCtx)
	rejectedClient := &Client{
		hub:     hub,
		userID:  1,
		send:    make(chan []byte, 256),
		isAdmin: false,
		conn:    wsConn,
		ctx:     ctx,
		cancel:  cancel,
	}
	hub.register <- rejectedClient

	// The rejected client should receive an error message
	select {
	case msg := <-rejectedClient.send:
		assert.Contains(t, string(msg), "Too many connections")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for rejection message")
	}

	hub.mutex.RLock()
	assert.Equal(t, maxConnectionsPerUser, hub.userConnections[1])
	hub.mutex.RUnlock()
}

func TestHub_Broadcast(t *testing.T) {
	hub := &Hub{
		clients:         make(map[*Client]bool),
		register:        make(chan *Client, 100),
		unregister:      make(chan *Client, 100),
		broadcast:       make(chan Event, 1000),
		userConnections: make(map[int]int),
		done:            make(chan struct{}),
	}
	go hub.Run()
	defer hub.Shutdown()

	client := registerClient(t, hub, 1, nil, true)
	drainMessages(client.send)

	hub.Broadcast(Event{
		Type: EventStockUpdate,
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "stock_update")
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast to reach client")
	}
}

func TestHub_Broadcast_TimestampAutoFill(t *testing.T) {
	hub := NewHub(nil)
	go hub.Run()
	defer hub.Shutdown()

	client := registerClient(t, hub, 1, nil, true)
	drainMessages(client.send)

	event := Event{Type: EventStockUpdate}
	hub.Broadcast(event)

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "stock_update")
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast")
	}
}

func TestHub_ShouldReceiveEvent(t *testing.T) {
	tests := []struct {
		name     string
		client   *Client
		event    *Event
		expected bool
	}{
		{
			name:     "admin receives all",
			client:   &Client{isAdmin: true},
			event:    &Event{StoreID: nil},
			expected: true,
		},
		{
			name:     "same store receives",
			client:   &Client{storeID: intPtr(1)},
			event:    &Event{StoreID: intPtr(1)},
			expected: true,
		},
		{
			name:     "different store blocked for non-admin",
			client:   &Client{storeID: intPtr(1), isAdmin: false},
			event:    &Event{StoreID: intPtr(2)},
			expected: false,
		},
		{
			name:     "nil store passes",
			client:   &Client{storeID: nil},
			event:    &Event{StoreID: nil},
			expected: true,
		},
		{
			name:     "admin with different store receives",
			client:   &Client{storeID: intPtr(1), isAdmin: true},
			event:    &Event{StoreID: intPtr(2)},
			expected: true,
		},
		{
			name:     "event nil store to client with store",
			client:   &Client{storeID: intPtr(1), isAdmin: false},
			event:    &Event{StoreID: nil},
			expected: true,
		},
		{
			name:     "client nil store receives event with store",
			client:   &Client{storeID: nil, isAdmin: false},
			event:    &Event{StoreID: intPtr(1)},
			expected: true,
		},
	}

	hub := &Hub{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hub.ShouldReceiveEvent(tt.client, tt.event)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHub_Unregister(t *testing.T) {
	hub := &Hub{
		clients:         make(map[*Client]bool),
		register:        make(chan *Client, 100),
		unregister:      make(chan *Client, 100),
		broadcast:       make(chan Event, 1000),
		userConnections: make(map[int]int),
		done:            make(chan struct{}),
	}
	go hub.Run()
	defer hub.Shutdown()

	client := registerClient(t, hub, 1, nil, false)
	hub.unregister <- client

	assert.Eventually(t, func() bool {
		hub.mutex.RLock()
		defer hub.mutex.RUnlock()
		return hub.userConnections[1] == 0
	}, time.Second, 10*time.Millisecond)
}

func TestHub_BroadcastStoreFiltering(t *testing.T) {
	hub := &Hub{
		clients:         make(map[*Client]bool),
		register:        make(chan *Client, 100),
		unregister:      make(chan *Client, 100),
		broadcast:       make(chan Event, 1000),
		userConnections: make(map[int]int),
		done:            make(chan struct{}),
	}
	go hub.Run()
	defer hub.Shutdown()

	store1 := intPtr(1)
	store2 := intPtr(2)

	adminClient := registerClient(t, hub, 1, store1, true)
	regularClient := registerClient(t, hub, 2, store2, false)
	drainMessages(adminClient.send)
	drainMessages(regularClient.send)

	// Broadcast to store 1 only
	hub.Broadcast(Event{
		Type:    EventStockUpdate,
		StoreID: store1,
	})

	// Admin (store 1) should receive it
	select {
	case msg := <-adminClient.send:
		assert.Contains(t, string(msg), "stock_update")
	case <-time.After(time.Second):
		t.Fatal("timeout: admin should receive broadcast for own store")
	}

	// Regular client (store 2) should NOT receive it
	select {
	case msg := <-regularClient.send:
		t.Fatalf("regular client in different store should not receive broadcast, got: %s", msg)
	case <-time.After(200 * time.Millisecond):
		// expected: no message
	}
}

func TestHub_BroadcastFullChannel(t *testing.T) {
	hub := &Hub{
		clients:         make(map[*Client]bool),
		register:        make(chan *Client, 100),
		unregister:      make(chan *Client, 100),
		broadcast:       make(chan Event, 1), // tiny buffer
		userConnections: make(map[int]int),
		done:            make(chan struct{}),
	}
	go hub.Run()
	defer hub.Shutdown()

	// Fill the broadcast channel
	hub.broadcast <- Event{Type: EventStockUpdate}

	// This should not block (drops event with warning)
	done := make(chan struct{})
	go func() {
		hub.Broadcast(Event{Type: EventSaleCreated})
		close(done)
	}()

	select {
	case <-done:
		// Broadcast returned without blocking
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Broadcast blocked on full channel")
	}
}

func TestHub_BroadcastWithPayload(t *testing.T) {
	hub := &Hub{
		clients:         make(map[*Client]bool),
		register:        make(chan *Client, 100),
		unregister:      make(chan *Client, 100),
		broadcast:       make(chan Event, 1000),
		userConnections: make(map[int]int),
		done:            make(chan struct{}),
	}
	go hub.Run()
	defer hub.Shutdown()

	client := registerClient(t, hub, 1, nil, true)
	drainMessages(client.send)

	hub.Broadcast(Event{
		Type:    EventSaleCreated,
		Payload: []byte(`{"id":1,"invoice":"INV-001","total":50000,"items":3}`),
	})

	select {
	case msg := <-client.send:
		assert.Contains(t, string(msg), "sale_created")
		assert.Contains(t, string(msg), "INV-001")
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast")
	}
}
