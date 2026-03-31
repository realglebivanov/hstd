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

func (b *Broadcast) Add(c *wsconn.WSConn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[c] = struct{}{}
}

func (b *Broadcast) Remove(c *wsconn.WSConn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.subs, c)
}

func (b *Broadcast) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for c := range b.subs {
		c.Close()
	}
}

func (b *Broadcast) Broadcast(row *view.Row, sender *wsconn.WSConn) {
	msg := struct {
		Type string    `json:"type"`
		Row  *view.Row `json:"row"`
	}{"link_updated", row}

	b.mu.Lock()
	defer b.mu.Unlock()
	for c := range b.subs {
		if c == sender {
			continue
		}
		c.Send(msg)
	}
}
