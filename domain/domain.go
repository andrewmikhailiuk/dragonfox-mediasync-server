package domain

type Message struct {
	Type      string `json:"type"`
	Position  *int64 `json:"position,omitempty"`
	Timestamp int64  `json:"timestamp"`
	ClientID  string `json:"clientId,omitempty"`
}

type Connection interface {
	ID() string
	Room() string
	Send(data []byte) error
	Close() error
}

type Broadcaster interface {
	Register(conn Connection)
	Unregister(conn Connection)
	Broadcast(sender Connection, data []byte)
	Stats() (rooms, clients int)
}

type MessageHandler interface {
	Handle(conn Connection, data []byte)
}
