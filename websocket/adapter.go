package websocket

import (
	"log/slog"
	"time"

	"github.com/gorilla/websocket"

	"dragonfox-mediasync-server/domain"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

type Conn struct {
	id          string
	room        string
	ws          *websocket.Conn
	send        chan []byte
	broadcaster domain.Broadcaster
	handler     domain.MessageHandler
}

func NewConn(id, room string, ws *websocket.Conn, b domain.Broadcaster, h domain.MessageHandler) *Conn {
	return &Conn{
		id:          id,
		room:        room,
		ws:          ws,
		send:        make(chan []byte, 256),
		broadcaster: b,
		handler:     h,
	}
}

func (c *Conn) ID() string   { return c.id }
func (c *Conn) Room() string { return c.room }

func (c *Conn) Send(data []byte) error {
	select {
	case c.send <- data:
		return nil
	default:
		return websocket.ErrCloseSent
	}
}

func (c *Conn) Close() error {
	return c.ws.Close()
}

func (c *Conn) Start() {
	c.broadcaster.Register(c)
	go c.writePump()
	go c.readPump()
}

func (c *Conn) readPump() {
	defer func() {
		c.broadcaster.Unregister(c)
		c.ws.Close()
	}()

	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("read error", "clientId", c.id, "error", err)
			}
			return
		}

		c.handler.Handle(c, data)
	}
}

func (c *Conn) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.ws.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
