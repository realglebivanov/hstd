package dataloader

import (
	"github.com/realglebivanov/hstd/hstdlib/dataloader/cache"
	"github.com/realglebivanov/hstd/hstdlib/dataloader/cidrs"
	"github.com/realglebivanov/hstd/hstdlib/dataloader/geodata"
)

type Loader struct {
	cidrs   *cidrs.Loader
	geodata *geodata.Loader
}

func New(cacheDir string) *Loader {
	c := cache.New(cacheDir)
	return &Loader{
		cidrs:   cidrs.NewLoader(c),
		geodata: geodata.NewLoader(c),
	}
}

func (l *Loader) LoadCIDRs() ([]string, error) {
	return l.cidrs.Load()
}

func (l *Loader) LoadGeodata() (*geodata.Stale, error) {
	return l.geodata.Load()
}

func (l *Loader) RefreshGeodata() error {
	return l.geodata.Refresh()
}
