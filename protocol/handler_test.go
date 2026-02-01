package protocol

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"dragonfox-mediasync-server/domain"
)

type mockConn struct {
	id   string
	room string
	sent [][]byte
	mu   sync.Mutex
}

func (m *mockConn) ID() string   { return m.id }
func (m *mockConn) Room() string { return m.room }

func (m *mockConn) Send(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, data)
	return nil
}

func (m *mockConn) Close() error { return nil }

func (m *mockConn) getSent() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sent
}

type mockBroadcaster struct {
	broadcasts []broadcastCall
	mu         sync.Mutex
}

type broadcastCall struct {
	senderID string
	data     []byte
}

func (m *mockBroadcaster) Register(conn domain.Connection)   {}
func (m *mockBroadcaster) Unregister(conn domain.Connection) {}
func (m *mockBroadcaster) Stats() (int, int)                 { return 0, 0 }

func (m *mockBroadcaster) Broadcast(sender domain.Connection, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.broadcasts = append(m.broadcasts, broadcastCall{senderID: sender.ID(), data: data})
}

func (m *mockBroadcaster) getBroadcasts() []broadcastCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.broadcasts
}

func TestHandler_PingPong(t *testing.T) {
	broadcaster := &mockBroadcaster{}
	handler := NewHandler(broadcaster)
	conn := &mockConn{id: "client1", room: "room1"}

	ping := domain.Message{Type: "ping", Timestamp: 12345}
	data, _ := json.Marshal(ping)

	handler.Handle(conn, data)

	sent := conn.getSent()
	require.Len(t, sent, 1)

	var pong domain.Message
	err := json.Unmarshal(sent[0], &pong)
	require.NoError(t, err)

	assert.Equal(t, "pong", pong.Type)
	assert.Equal(t, int64(12345), pong.Timestamp)
	assert.Equal(t, "client1", pong.ClientID)

	assert.Empty(t, broadcaster.getBroadcasts())
}

func TestHandler_Broadcast(t *testing.T) {
	broadcaster := &mockBroadcaster{}
	handler := NewHandler(broadcaster)
	conn := &mockConn{id: "client1", room: "room1"}

	msg := domain.Message{Type: "toggle", Timestamp: 99999}
	data, _ := json.Marshal(msg)

	handler.Handle(conn, data)

	broadcasts := broadcaster.getBroadcasts()
	require.Len(t, broadcasts, 1)
	assert.Equal(t, "client1", broadcasts[0].senderID)

	var sent domain.Message
	err := json.Unmarshal(broadcasts[0].data, &sent)
	require.NoError(t, err)

	assert.Equal(t, "toggle", sent.Type)
	assert.Equal(t, "client1", sent.ClientID)
}

func TestHandler_InvalidJSON(t *testing.T) {
	broadcaster := &mockBroadcaster{}
	handler := NewHandler(broadcaster)
	conn := &mockConn{id: "client1", room: "room1"}

	handler.Handle(conn, []byte("not json"))

	assert.Empty(t, conn.getSent())
	assert.Empty(t, broadcaster.getBroadcasts())
}
