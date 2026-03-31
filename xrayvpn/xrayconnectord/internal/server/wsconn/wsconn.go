package wsconn

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/server/wsconn/state"
)

type WSConn struct {
	state *state.ConnState
}

var upgrader = websocket.Upgrader{}

func Upgrade(w http.ResponseWriter, r *http.Request) (*WSConn, error) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	s, err := state.New(c)
	if err != nil {
		return nil, err
	}

	return &WSConn{state: s}, nil
}

func (c *WSConn) Send(v any) error {
	return c.state.Send(v)
}

func (c *WSConn) ReadEvent() (any, error) {
	data, err := c.state.ReadData()
	if err != nil {
		return nil, err
	}

	return parseEvent(data)
}

func (c *WSConn) Close() {
	c.state.Close()
}
