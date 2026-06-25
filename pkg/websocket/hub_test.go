package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())

	client1 := &Client{
		hub:     hub,
		userID:  1,
		send:    make(chan []byte, 256),
		isAdmin: true,
		ctx:     ctx1,
		cancel:  cancel1,
	}

	client2 := &Client{
		hub:     hub,
		userID:  2,
		send:    make(chan []byte, 256),
		isAdmin: false,
		ctx:     ctx2,
		cancel:  cancel2,
	}

	hub.register <- client1
	assert.Eventually(t, func() bool {
		hub.mutex.RLock()
		defer hub.mutex.RUnlock()
		return hub.userConnections[1] == 1
	}, time.Second, 10*time.Millisecond)

	hub.register <- client2
	assert.Eventually(t, func() bool {
		hub.mutex.RLock()
		defer hub.mutex.RUnlock()
		return hub.userConnections[2] == 1
	}, time.Second, 10*time.Millisecond)

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

	client := &Client{
		hub:     hub,
		userID:  1,
		send:    make(chan []byte, 256),
		isAdmin: true,
		ctx:     context.Background(),
		cancel:  func() {},
	}
	hub.register <- client
	assert.Eventually(t, func() bool {
		hub.mutex.RLock()
		defer hub.mutex.RUnlock()
		return hub.userConnections[1] == 1
	}, time.Second, 10*time.Millisecond)

	// Drain the user_online_count event sent during registration
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
	}

	hub := &Hub{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hub.ShouldReceiveEvent(tt.client, tt.event)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func intPtr(i int) *int { return &i }
