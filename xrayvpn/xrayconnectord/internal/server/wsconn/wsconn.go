package wsconn

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/server/admin/view"
)

type LinksMsg struct {
	Type string      `json:"type"`
	Rows []*view.Row `json:"rows"`
}

type LinkUpdatedMsg struct {
	Type string    `json:"type"`
	Row  *view.Row `json:"row"`
}

type ErrorMsg struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type EventError struct {
	Message string
}

func (e *EventError) Error() string { return e.Message }

type UpdateLinkReq struct {
	Index   int     `json:"index"`
	Comment *string `json:"comment"`
	Enabled *bool   `json:"enabled"`
}

type WSConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
	done chan struct{}
	wg   sync.WaitGroup
}

const (
	pingInterval = 30 * time.Second
	readTimeout  = 60 * time.Second
	pongTimeout  = 40 * time.Second
	writeTimeout = 5 * time.Second
)

var upgrader = websocket.Upgrader{}

func Upgrade(w http.ResponseWriter, r *http.Request) (*WSConn, error) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return &WSConn{conn: c, done: make(chan struct{})}, nil
}

func (c *WSConn) WriteEvent(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.conn.WriteJSON(v); err != nil {
		slog.Warn("ws broadcast write", "err", err)
		return err
	}

	return nil
}

func (c *WSConn) StartKeepAlive() {
	c.conn.SetReadDeadline(time.Now().Add(readTimeout))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongTimeout))
		return nil
	})

	c.wg.Go(func() {
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-c.done:
				return
			case <-ticker.C:
				if err := c.ping(); err != nil {
					slog.Warn("send ping control message", "err", err)
					return
				}
			}
		}
	})
}

func (c *WSConn) Close() {
	close(c.done)
	c.wg.Wait()
	c.conn.Close()
}

func (c *WSConn) ReadEvent() (any, error) {
	_, data, err := c.conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, &EventError{Message: "invalid json"}
	}

	var v any
	switch envelope.Type {
	case "update_link":
		v = &UpdateLinkReq{}
	default:
		return nil, &EventError{Message: fmt.Sprintf("unknown type: %s", envelope.Type)}
	}

	if err := json.Unmarshal(data, v); err != nil {
		return nil, &EventError{Message: fmt.Sprintf("invalid json for %s", envelope.Type)}
	}
	return v, nil
}

func (c *WSConn) ping() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeTimeout))
}
