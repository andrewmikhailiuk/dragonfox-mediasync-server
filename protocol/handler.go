package protocol

import (
	"encoding/json"
	"log/slog"

	"dragonfox-mediasync-server/domain"
)

type Handler struct {
	broadcaster domain.Broadcaster
}

func NewHandler(b domain.Broadcaster) *Handler {
	return &Handler{broadcaster: b}
}

func (h *Handler) Handle(conn domain.Connection, data []byte) {
	var msg domain.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		slog.Warn("invalid message", "clientId", conn.ID(), "error", err)
		return
	}

	if msg.Type == "ping" {
		pong := domain.Message{Type: "pong", Timestamp: msg.Timestamp, ClientID: conn.ID()}
		if resp, err := json.Marshal(pong); err == nil {
			conn.Send(resp)
		}
		return
	}

	msg.ClientID = conn.ID()
	broadcast, err := json.Marshal(msg)
	if err != nil {
		slog.Warn("marshal error", "clientId", conn.ID(), "error", err)
		return
	}

	h.broadcaster.Broadcast(conn, broadcast)
}
