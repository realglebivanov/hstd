package cidrs

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/hstdlib/dataloader"
)

type CIDRs struct {
	loader *dataloader.Loader
	cidrs  atomic.Value
	cancel context.CancelFunc
}

func New() *CIDRs {
	stateDir := hstdlib.MustEnv("STATE_DIRECTORY")
	loader := dataloader.New(stateDir)

	cidrs, err := loader.LoadCIDRs()
	if err != nil {
		slog.Warn("initial CIDR load failed", "err", err)
	}
	c := &CIDRs{loader: loader}
	c.cidrs.Store(cidrs)
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
}

func (c *CIDRs) refreshLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cidrs, err := c.loader.LoadCIDRs()
			if err != nil {
				slog.Error("refresh CIDRs", "err", err)
				continue
			}
			c.cidrs.Store(cidrs)
			slog.Info("refreshed CIDRs", "count", len(cidrs))
		}
	}
}
