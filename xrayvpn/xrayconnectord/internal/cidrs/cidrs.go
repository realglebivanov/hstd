package cidrs

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/realglebivanov/hstd/hstdlib"
	datacidrs "github.com/realglebivanov/hstd/hstdlib/dataloader/cidrs"
)

type CIDRs struct {
	loader *datacidrs.Loader
	cidrs  atomic.Value
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func New() *CIDRs {
	stateDir := hstdlib.MustEnv("STATE_DIRECTORY")
	loader := datacidrs.NewLoader(stateDir)

	c := &CIDRs{loader: loader}
	c.loadCIDRs()
	return c
}

func (c *CIDRs) Get() []string {
	return c.cidrs.Load().([]string)
}

func (c *CIDRs) StartRefresh(interval time.Duration) {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	go c.refreshLoop(ctx, interval)
}

func (c *CIDRs) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
}

func (c *CIDRs) refreshLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.loader.Update:
			c.loadCIDRs()
		case <-ticker.C:
			c.loadCIDRs()
		}
	}
}

func (c *CIDRs) loadCIDRs() {
	data, err := c.loader.Load()
	if err != nil {
		slog.Error("load CIDRs", "err", err)
		return
	}
	c.cidrs.Store(data.CIDRs)
	slog.Info("loaded CIDRs", "count", len(data.CIDRs))

	if data.ShouldRefresh() {
		c.wg.Go(func() { data.Refresh() })
	}
}
