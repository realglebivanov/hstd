package wsconn

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WSCconn struct {
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

func Upgrade(w http.ResponseWriter, r *http.Request) (*WSCconn, error) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return &WSCconn{conn: c, done: make(chan struct{})}, nil
}

func (c *WSCconn) WriteJSON(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(v)
}

func (c *WSCconn) StartKeepAlive() {
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
				if c.ping() != nil {
					return
				}
			}
		}
	})
}

func (c *WSCconn) Close() {
	close(c.done)
	c.wg.Wait()
	c.conn.Close()
}

func (c *WSCconn) ReadMessage() ([]byte, error) {
	_, data, err := c.conn.ReadMessage()
	return data, err
}

func (c *WSCconn) ping() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeTimeout))
}
