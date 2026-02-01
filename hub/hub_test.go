package hub

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockConn struct {
	id       string
	room     string
	received [][]byte
	closed   bool
	mu       sync.Mutex
	sendErr  error
}

func (m *mockConn) ID() string   { return m.id }
func (m *mockConn) Room() string { return m.room }

func (m *mockConn) Send(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sendErr != nil {
		return m.sendErr
	}
	m.received = append(m.received, data)
	return nil
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockConn) getReceived() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.received
}

func TestHub_Broadcast(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(*Hub) ([]*mockConn, *mockConn)
		wantReceived map[string]int
	}{
		{
			name: "broadcast to room members",
			setup: func(h *Hub) ([]*mockConn, *mockConn) {
				sender := &mockConn{id: "sender", room: "room1"}
				receiver1 := &mockConn{id: "recv1", room: "room1"}
				receiver2 := &mockConn{id: "recv2", room: "room1"}
				h.Register(sender)
				h.Register(receiver1)
				h.Register(receiver2)
				return []*mockConn{receiver1, receiver2}, sender
			},
			wantReceived: map[string]int{"recv1": 1, "recv2": 1},
		},
		{
			name: "no cross-room broadcast",
			setup: func(h *Hub) ([]*mockConn, *mockConn) {
				sender := &mockConn{id: "sender", room: "room1"}
				receiver := &mockConn{id: "recv1", room: "room2"}
				h.Register(sender)
				h.Register(receiver)
				return []*mockConn{receiver}, sender
			},
			wantReceived: map[string]int{"recv1": 0},
		},
		{
			name: "single client in room",
			setup: func(h *Hub) ([]*mockConn, *mockConn) {
				sender := &mockConn{id: "sender", room: "room1"}
				h.Register(sender)
				return []*mockConn{}, sender
			},
			wantReceived: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New()
			receivers, sender := tt.setup(h)

			h.Broadcast(sender, []byte("test message"))

			for _, r := range receivers {
				expected := tt.wantReceived[r.ID()]
				assert.Len(t, r.getReceived(), expected, "receiver %s", r.ID())
			}
		})
	}
}

func TestHub_Stats(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*Hub)
		wantRooms   int
		wantClients int
	}{
		{
			name:        "empty hub",
			setup:       func(h *Hub) {},
			wantRooms:   0,
			wantClients: 0,
		},
		{
			name: "one room one client",
			setup: func(h *Hub) {
				h.Register(&mockConn{id: "c1", room: "r1"})
			},
			wantRooms:   1,
			wantClients: 1,
		},
		{
			name: "multiple rooms",
			setup: func(h *Hub) {
				h.Register(&mockConn{id: "c1", room: "r1"})
				h.Register(&mockConn{id: "c2", room: "r1"})
				h.Register(&mockConn{id: "c3", room: "r2"})
			},
			wantRooms:   2,
			wantClients: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New()
			tt.setup(h)

			rooms, clients := h.Stats()

			assert.Equal(t, tt.wantRooms, rooms)
			assert.Equal(t, tt.wantClients, clients)
		})
	}
}

func TestHub_RoomCleanup(t *testing.T) {
	h := New()
	conn := &mockConn{id: "c1", room: "r1"}

	h.Register(conn)
	rooms, _ := h.Stats()
	require.Equal(t, 1, rooms)

	h.Unregister(conn)
	rooms, clients := h.Stats()
	assert.Equal(t, 0, rooms)
	assert.Equal(t, 0, clients)
}
