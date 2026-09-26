package websocket

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureLogs redirects the default slog logger into a recording handler for
// the duration of the test.
type captureLogs struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *captureLogs) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *captureLogs) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r)
	return nil
}

func (h *captureLogs) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *captureLogs) WithGroup(_ string) slog.Handler      { return h }

// atOrAbove returns every captured record at level or higher than min.
func (h *captureLogs) atOrAbove(min slog.Level) []slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []slog.Record
	for _, r := range h.records {
		if r.Level >= min {
			out = append(out, r)
		}
	}
	return out
}

func startCapture(t *testing.T) *captureLogs {
	t.Helper()
	h := &captureLogs{}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })
	return h
}

func attrsOf(r slog.Record) map[string]any {
	attrs := make(map[string]any)
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})
	return attrs
}

// A normal connection must produce exactly one concise lifecycle event on the
// way in and one on the way out — no overlapping "auth OK"/"registered" pairs.
func TestHub_LifecycleLogsOncePerConnection(t *testing.T) {
	h := startCapture(t)
	hub := newTestHub(nil)
	go hub.Run()

	client := &Client{
		hub:    hub,
		userID: 7,
		role:   "cashier",
		ip:     "10.0.0.9",
		send:   make(chan []byte, 1),
		ctx:    context.Background(),
		cancel: func() {},
	}

	hub.register <- client
	require.Eventually(t, func() bool {
		hub.mutex.RLock()
		defer hub.mutex.RUnlock()
		return len(hub.clients) == 1
	}, time.Second, 10*time.Millisecond)

	hub.unregister <- client
	require.Eventually(t, func() bool {
		hub.mutex.RLock()
		defer hub.mutex.RUnlock()
		return len(hub.clients) == 0
	}, time.Second, 10*time.Millisecond)

	hub.Shutdown()

	infos := h.atOrAbove(slog.LevelInfo)
	require.Len(t, infos, 2, "expected one connect and one disconnect event, got %v", infos)

	assert.Equal(t, "WebSocket client connected", infos[0].Message)
	assert.Equal(t, slog.LevelInfo, infos[0].Level)
	connected := attrsOf(infos[0])
	assert.Equal(t, int64(7), connected["user_id"])
	assert.Equal(t, "10.0.0.9", connected["ip"], "client IP is retained for auth diagnostics")

	assert.Equal(t, "WebSocket client disconnected", infos[1].Message)
	assert.Equal(t, slog.LevelInfo, infos[1].Level)
	disconnected := attrsOf(infos[1])
	assert.Equal(t, int64(7), disconnected["user_id"])
	assert.Equal(t, "10.0.0.9", disconnected["ip"])
}

// readPump and the slow-client drop can both push the same client onto
// h.unregister. The second arrival must not re-announce a disconnect.
func TestHub_DuplicateUnregisterLogsOnce(t *testing.T) {
	h := startCapture(t)
	hub := newTestHub(nil)
	go hub.Run()

	client := &Client{
		hub:    hub,
		userID: 3,
		role:   "cashier",
		ip:     "10.0.0.4",
		send:   make(chan []byte, 1),
		ctx:    context.Background(),
		cancel: func() {},
	}

	hub.register <- client
	require.Eventually(t, func() bool {
		hub.mutex.RLock()
		defer hub.mutex.RUnlock()
		return len(hub.clients) == 1
	}, time.Second, 10*time.Millisecond)

	hub.unregister <- client
	require.Eventually(t, func() bool {
		hub.mutex.RLock()
		defer hub.mutex.RUnlock()
		return len(hub.clients) == 0
	}, time.Second, 10*time.Millisecond)

	// A second unregister for the same client (no double close of send).
	hub.unregister <- client
	require.Eventually(t, func() bool {
		hub.userConnMu.RLock()
		defer hub.userConnMu.RUnlock()
		return hub.userConnections[3] == 0
	}, time.Second, 10*time.Millisecond)

	hub.Shutdown()

	disconnects := 0
	for _, rec := range h.atOrAbove(slog.LevelInfo) {
		if rec.Message == "WebSocket client disconnected" {
			disconnects++
		}
	}
	assert.Equal(t, 1, disconnects, "a duplicate unregister must not log a second disconnect")
}

func TestIsExpectedCloseError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"close already sent", websocket.ErrCloseSent, true},
		{"wrapped close already sent", fmtErrWrap(websocket.ErrCloseSent), true},
		{"socket closed", net.ErrClosed, true},
		{"wrapped socket closed", fmtErrWrap(net.ErrClosed), true},
		{"genuine write failure", errors.New("broken pipe"), false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isExpectedCloseError(tt.err))
		})
	}
}

func fmtErrWrap(err error) error {
	return errors.Join(errors.New("context"), err)
}

// A close frame written after gorilla has already sent one is the normal
// shutdown path and must not be reported as a warning.
func TestWritePump_CloseAfterCloseSentIsNotWarned(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	serverConnCh := make(chan *websocket.Conn, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverConnCh <- conn
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer ts.Close()

	dialer := &websocket.Dialer{}
	dialerConn, _, err := dialer.Dial("ws"+ts.URL[len("http"):], nil)
	require.NoError(t, err)
	defer func() { _ = dialerConn.Close() }()

	serverConn := <-serverConnCh
	// The first close write makes every later write return ErrCloseSent.
	require.NoError(t, serverConn.WriteMessage(websocket.CloseMessage, []byte{}))

	h := startCapture(t)
	hub := &Hub{}
	send := make(chan []byte)
	close(send)
	client := &Client{hub: hub, userID: 5, conn: serverConn, send: send}
	ctx, cancel := context.WithCancel(context.Background())
	client.ctx, client.cancel = ctx, cancel

	hub.wg.Add(1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		client.writePump()
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("writePump did not return")
	}

	assert.Empty(t, h.atOrAbove(slog.LevelWarn),
		"expected websocket: close sent during shutdown to be a non-error, got %v", h.atOrAbove(slog.LevelWarn))
}
