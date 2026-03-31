package broadcast

import (
	"sync"

	"github.com/realglebivanov/hstd/xrayconnectord/internal/server/admin/view"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/server/wsconn"
)

type Broadcast struct {
	mu   sync.Mutex
	subs map[*wsconn.WSConn]struct{}
}

func New() *Broadcast {
	return &Broadcast{subs: make(map[*wsconn.WSConn]struct{})}
}

func (s *Broadcast) Add(c *wsconn.WSConn) {
	s.mu.Lock()
	s.subs[c] = struct{}{}
	s.mu.Unlock()
}

func (s *Broadcast) Remove(c *wsconn.WSConn) {
	s.mu.Lock()
	delete(s.subs, c)
	s.mu.Unlock()
}

func (s *Broadcast) Broadcast(row *view.Row, sender *wsconn.WSConn) {
	msg := struct {
		Type string    `json:"type"`
		Row  *view.Row `json:"row"`
	}{"link_updated", row}

	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.subs {
		if c == sender {
			continue
		}
		c.WriteEvent(msg)
	}
}
