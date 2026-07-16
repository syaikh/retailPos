package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

type mockValidator struct {
	claims *Claims
	err    error
}

func (m *mockValidator) ValidateToken(token string) (*Claims, error) {
	return m.claims, m.err
}

type authError struct{ msg string }

func (e *authError) Error() string { return e.msg }

func setupGinWithWS(hub *Hub) *httptest.Server {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/ws", func(c *gin.Context) {
		ServeWebSocket(hub, c)
	})
	return httptest.NewServer(r)
}

func wsDialWithOrigin(t *testing.T, url, origin string) (*websocket.Conn, error) {
	t.Helper()
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	wsURL := strings.Replace(url, "http://", "ws://", 1) + "/ws"
	conn, _, err := dialer.Dial(wsURL, http.Header{"Origin": []string{origin}})
	return conn, err
}

func wsDial(t *testing.T, url string) (*websocket.Conn, error) {
	t.Helper()
	return wsDialWithOrigin(t, url, "http://localhost:5173")
}

func sendWSAuth(t *testing.T, conn *websocket.Conn, token string) {
	t.Helper()
	msg, _ := json.Marshal(authMessage{Type: "auth", Token: token})
	if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		t.Fatalf("send auth: %v", err)
	}
}

func readOneMessage(t *testing.T, conn *websocket.Conn, timeout time.Duration) []byte {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	return msg
}

func connectAndAuth(t *testing.T, hub *Hub, srv *httptest.Server, token string) *websocket.Conn {
	t.Helper()
	conn, err := wsDial(t, srv.URL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	sendWSAuth(t, conn, token)
	_ = readOneMessage(t, conn, 3*time.Second)
	return conn
}

func TestServeWebSocket_AuthSuccess(t *testing.T) {
	validator := &mockValidator{
		claims: &Claims{ID: 1, Role: "cashier", StoreID: intPtr(10)},
	}
	hub := newTestHub(validator)
	go hub.Run()
	defer hub.Shutdown()

	srv := setupGinWithWS(hub)
	defer srv.Close()

	conn, err := wsDial(t, srv.URL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	sendWSAuth(t, conn, "valid-token")

	msg := readOneMessage(t, conn, 3*time.Second)
	var evt Event
	if err := json.Unmarshal(msg, &evt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if evt.Type != EventUserOnline {
		t.Fatalf("expected user_online_count, got %s", evt.Type)
	}

	hub.mutex.RLock()
	count := len(hub.clients)
	hub.mutex.RUnlock()
	if count != 1 {
		t.Fatalf("expected 1 client, got %d", count)
	}
}

func TestServeWebSocket_AuthFailure(t *testing.T) {
	validator := &mockValidator{err: &authError{"invalid token"}}
	hub := newTestHub(validator)
	go hub.Run()
	defer hub.Shutdown()

	srv := setupGinWithWS(hub)
	defer srv.Close()

	conn, err := wsDial(t, srv.URL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	sendWSAuth(t, conn, "bad-token")

	_, _, err = conn.ReadMessage()
	if err == nil {
		conn.Close()
		t.Fatal("expected connection to be closed after auth failure")
	}
}

func TestServeWebSocket_InvalidAuthFormat(t *testing.T) {
	validator := &mockValidator{}
	hub := newTestHub(validator)
	go hub.Run()
	defer hub.Shutdown()

	srv := setupGinWithWS(hub)
	defer srv.Close()

	conn, err := wsDial(t, srv.URL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	msg, _ := json.Marshal(map[string]string{"type": "wrong"})
	_ = conn.WriteMessage(websocket.TextMessage, msg)

	_, _, err = conn.ReadMessage()
	if err == nil {
		conn.Close()
		t.Fatal("expected connection to close on invalid auth format")
	}
}

func TestServeWebSocket_MissingAuthType(t *testing.T) {
	validator := &mockValidator{}
	hub := newTestHub(validator)
	go hub.Run()
	defer hub.Shutdown()

	srv := setupGinWithWS(hub)
	defer srv.Close()

	conn, err := wsDial(t, srv.URL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	msg, _ := json.Marshal(authMessage{Type: "login", Token: "tok"})
	_ = conn.WriteMessage(websocket.TextMessage, msg)

	_, _, err = conn.ReadMessage()
	if err == nil {
		conn.Close()
		t.Fatal("expected connection to close on missing auth type")
	}
}

func TestServeWebSocket_EmptyToken(t *testing.T) {
	validator := &mockValidator{}
	hub := newTestHub(validator)
	go hub.Run()
	defer hub.Shutdown()

	srv := setupGinWithWS(hub)
	defer srv.Close()

	conn, err := wsDial(t, srv.URL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	msg, _ := json.Marshal(authMessage{Type: "auth", Token: ""})
	_ = conn.WriteMessage(websocket.TextMessage, msg)

	_, _, err = conn.ReadMessage()
	if err == nil {
		conn.Close()
		t.Fatal("expected connection to close on empty token")
	}
}

func TestServeWebSocket_WritePumpDeliversMessages(t *testing.T) {
	validator := &mockValidator{claims: &Claims{ID: 2, Role: "cashier"}}
	hub := newTestHub(validator)
	go hub.Run()
	defer hub.Shutdown()

	srv := setupGinWithWS(hub)
	defer srv.Close()

	conn := connectAndAuth(t, hub, srv, "valid-tok")
	defer conn.Close()

	payload, _ := json.Marshal(map[string]string{"msg": "hello"})
	hub.Broadcast(Event{Type: EventStockUpdate, Payload: payload})

	deadline := time.Now().Add(3 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for stock_update event")
		}
		_ = conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			continue
		}
		var evt Event
		if err := json.Unmarshal(raw, &evt); err != nil {
			continue
		}
		if evt.Type == EventStockUpdate {
			return
		}
	}
}

func TestServeWebSocket_StoreFiltering(t *testing.T) {
	validator := &mockValidator{
		claims: &Claims{ID: 3, Role: "cashier", StoreID: intPtr(10)},
	}
	hub := newTestHub(validator)
	go hub.Run()
	defer hub.Shutdown()

	srv := setupGinWithWS(hub)
	defer srv.Close()

	conn := connectAndAuth(t, hub, srv, "valid-tok")
	defer conn.Close()

	payload, _ := json.Marshal(map[string]string{"msg": "other store"})
	hub.Broadcast(Event{
		Type:    EventStockUpdate,
		Payload: payload,
		StoreID: intPtr(99),
	})

	_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, _, err := conn.ReadMessage()
	if err == nil {
		t.Fatal("expected no message for different store")
	}
}

func TestServeWebSocket_AdminReceivesAllStoreEvents(t *testing.T) {
	validator := &mockValidator{
		claims: &Claims{ID: 4, Role: "superadmin", StoreID: intPtr(10)},
	}
	hub := newTestHub(validator)
	go hub.Run()
	defer hub.Shutdown()

	srv := setupGinWithWS(hub)
	defer srv.Close()

	conn := connectAndAuth(t, hub, srv, "valid-tok")
	defer conn.Close()

	payload, _ := json.Marshal(map[string]string{"msg": "all stores"})
	hub.Broadcast(Event{
		Type:    EventSaleCreated,
		Payload: payload,
		StoreID: intPtr(99),
	})

	deadline := time.Now().Add(3 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for sale_created event")
		}
		_ = conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			continue
		}
		var evt Event
		if err := json.Unmarshal(raw, &evt); err != nil {
			continue
		}
		if evt.Type == EventSaleCreated {
			return
		}
	}
}

func TestServeWebSocket_WritePumpCancelledContext(t *testing.T) {
	validator := &mockValidator{claims: &Claims{ID: 5, Role: "cashier"}}
	hub := newTestHub(validator)
	go hub.Run()
	defer hub.Shutdown()

	srv := setupGinWithWS(hub)
	defer srv.Close()

	conn := connectAndAuth(t, hub, srv, "valid-tok")
	defer conn.Close()

	hub.mutex.RLock()
	var target *Client
	for c := range hub.clients {
		if c.userID == 5 {
			target = c
			break
		}
	}
	hub.mutex.RUnlock()

	if target == nil {
		t.Fatal("client not found")
	}

	target.cancel()

	require.Eventually(t, func() bool {
		hub.mutex.RLock()
		defer hub.mutex.RUnlock()
		_, ok := hub.clients[target]
		return !ok
	}, 2*time.Second, 50*time.Millisecond, "expected client to be unregistered after context cancel")
}

func TestServeWebSocket_ReadPumpDisconnect(t *testing.T) {
	validator := &mockValidator{claims: &Claims{ID: 6, Role: "cashier"}}
	hub := newTestHub(validator)
	go hub.Run()
	defer hub.Shutdown()

	srv := setupGinWithWS(hub)
	defer srv.Close()

	conn := connectAndAuth(t, hub, srv, "valid-tok")

	conn.Close()

	require.Eventually(t, func() bool {
		hub.mutex.RLock()
		defer hub.mutex.RUnlock()
		return len(hub.clients) == 0
	}, 2*time.Second, 50*time.Millisecond, "expected 0 clients after disconnect")
}

func TestServeWebSocket_OriginRejected(t *testing.T) {
	validator := &mockValidator{claims: &Claims{ID: 7, Role: "cashier"}}
	hub := newTestHub(validator)
	go hub.Run()
	defer hub.Shutdown()

	srv := setupGinWithWS(hub)
	defer srv.Close()

	conn, err := wsDialWithOrigin(t, srv.URL, "http://evil.com")
	if err == nil {
		conn.Close()
		t.Fatal("expected dial to fail with bad origin")
	}
}

func TestServeWebSocket_AllowlistedOrigin(t *testing.T) {
	validator := &mockValidator{claims: &Claims{ID: 8, Role: "cashier"}}
	hub := newTestHub(validator)
	go hub.Run()
	defer hub.Shutdown()

	srv := setupGinWithWS(hub)
	defer srv.Close()

	conn := connectAndAuth(t, hub, srv, "valid-tok")
	defer conn.Close()

	hub.mutex.RLock()
	count := len(hub.clients)
	hub.mutex.RUnlock()
	if count != 1 {
		t.Fatalf("expected 1 client, got %d", count)
	}
}

func TestServeWebSocket_MaxConnectionsPerUser(t *testing.T) {
	validator := &mockValidator{claims: &Claims{ID: 9, Role: "cashier"}}
	hub := newTestHub(validator)
	go hub.Run()
	defer hub.Shutdown()

	srv := setupGinWithWS(hub)
	defer srv.Close()

	conns := make([]*websocket.Conn, 0, maxConnectionsPerUser)
	for i := 0; i < maxConnectionsPerUser; i++ {
		time.Sleep(600 * time.Millisecond)
		c, err := wsDial(t, srv.URL)
		if err != nil {
			t.Fatalf("dial %d: %v", i, err)
		}
		sendWSAuth(t, c, "valid-tok")
		_ = readOneMessage(t, c, 3*time.Second)
		conns = append(conns, c)
	}

	hub.mutex.RLock()
	count := len(hub.clients)
	userCount := hub.userConnections[9]
	hub.mutex.RUnlock()
	if count != maxConnectionsPerUser {
		t.Fatalf("expected %d clients, got %d", maxConnectionsPerUser, count)
	}
	if userCount != maxConnectionsPerUser {
		t.Fatalf("expected userConnections=%d, got %d", maxConnectionsPerUser, userCount)
	}

	time.Sleep(600 * time.Millisecond)
	extra, err := wsDial(t, srv.URL)
	if err != nil {
		t.Fatalf("dial extra: %v", err)
	}
	sendWSAuth(t, extra, "valid-tok")

	time.Sleep(500 * time.Millisecond)

	hub.mutex.RLock()
	count = len(hub.clients)
	hub.mutex.RUnlock()
	if count > maxConnectionsPerUser {
		t.Fatalf("expected at most %d clients, got %d", maxConnectionsPerUser, count)
	}
	extra.Close()

	for _, c := range conns {
		c.Close()
	}
}

func TestServeWebSocket_NonAdminFilteredByStore(t *testing.T) {
	validator := &mockValidator{
		claims: &Claims{ID: 10, Role: "cashier", StoreID: intPtr(5)},
	}
	hub := newTestHub(validator)
	go hub.Run()
	defer hub.Shutdown()

	srv := setupGinWithWS(hub)
	defer srv.Close()

	conn := connectAndAuth(t, hub, srv, "valid-tok")
	defer conn.Close()

	payload, _ := json.Marshal(map[string]string{"data": "store5"})
	hub.Broadcast(Event{
		Type:    EventStockUpdate,
		Payload: payload,
		StoreID: intPtr(5),
	})

	deadline := time.Now().Add(3 * time.Second)
	found := false
	for !found && time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			continue
		}
		var evt Event
		if err := json.Unmarshal(raw, &evt); err != nil {
			continue
		}
		if evt.Type == EventStockUpdate {
			found = true
		}
	}
	if !found {
		t.Fatal("expected stock_update for matching store")
	}
}
