package hub

import (
	"log/slog"
	"sync"

	"dragonfox-mediasync-server/domain"
)

type room struct {
	clients map[string]domain.Connection
	mu      sync.RWMutex
}

type Hub struct {
	rooms map[string]*room
	mu    sync.RWMutex
}

func New() *Hub {
	return &Hub{
		rooms: make(map[string]*room),
	}
}

func (h *Hub) Register(conn domain.Connection) {
	h.mu.Lock()
	r, exists := h.rooms[conn.Room()]
	if !exists {
		r = &room{clients: make(map[string]domain.Connection)}
		h.rooms[conn.Room()] = r
	}
	h.mu.Unlock()

	r.mu.Lock()
	r.clients[conn.ID()] = conn
	count := len(r.clients)
	r.mu.Unlock()

	slog.Info("client connected", "room", conn.Room(), "clientId", conn.ID(), "clients", count)
}

func (h *Hub) Unregister(conn domain.Connection) {
	h.mu.RLock()
	r, exists := h.rooms[conn.Room()]
	h.mu.RUnlock()

	if !exists {
		return
	}

	r.mu.Lock()
	delete(r.clients, conn.ID())
	count := len(r.clients)
	r.mu.Unlock()

	slog.Info("client disconnected", "room", conn.Room(), "clientId", conn.ID(), "clients", count)

	if count == 0 {
		h.mu.Lock()
		delete(h.rooms, conn.Room())
		h.mu.Unlock()
		slog.Info("room removed", "room", conn.Room())
	}
}

func (h *Hub) Broadcast(sender domain.Connection, data []byte) {
	h.mu.RLock()
	r, exists := h.rooms[sender.Room()]
	h.mu.RUnlock()

	if !exists {
		return
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for id, conn := range r.clients {
		if id == sender.ID() {
			continue
		}
		if err := conn.Send(data); err != nil {
			go func(c domain.Connection) {
				h.Unregister(c)
			}(conn)
		}
	}
}

func (h *Hub) Stats() (rooms, clients int) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	rooms = len(h.rooms)
	for _, r := range h.rooms {
		r.mu.RLock()
		clients += len(r.clients)
		r.mu.RUnlock()
	}
	return rooms, clients
}
