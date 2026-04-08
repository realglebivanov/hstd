package geodata

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"sync"

	"github.com/realglebivanov/hstd/hstdlib/dataloader/cache"
	"github.com/realglebivanov/hstd/hstdlib/httpclient"
)

type geoFile struct {
	url  string
	name string
}

type Loader struct {
	cache   *cache.Cache
	clients []*http.Client
	Notify  chan struct{}
}

const baseGeodataURL = "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/"

var geoFiles = []geoFile{
	{baseGeodataURL + "geoip.dat", "geoip.dat"},
	{baseGeodataURL + "geosite.dat", "geosite.dat"},
}

func NewLoader(cacheDir string) *Loader {
	return &Loader{
		cache:   cache.New(cacheDir),
		clients: []*http.Client{httpclient.Default, httpclient.Direct},
		Notify:  make(chan struct{}, 1),
	}
}

func (l *Loader) Load() (*Stale, error) {
	var stale []geoFile
	for _, f := range geoFiles {
		cr := l.cache.Read(f.name)
		switch cr.State {
		case cache.CacheFresh:
			slog.Info("using cached", "file", f.name)
		case cache.CacheStale:
			slog.Info("using stale", "file", f.name)
			stale = append(stale, f)
		case cache.CacheMissing:
			if err := l.fetchAndCache(f); err != nil {
				return nil, err
			}
		case cache.CacheError:
			return nil, fmt.Errorf("read %s: %w", f.name, cr.Err)
		}
	}
	return &Stale{stale: stale, loader: l}, nil
}

func (l *Loader) Refresh() error {
	errs := l.fetchAndCacheMany(geoFiles)
	return errors.Join(errs...)
}

func (l *Loader) fetchAndCacheMany(geoFiles []geoFile) []error {
	errs := make([]error, len(geoFiles))

	var wg sync.WaitGroup
	for i, f := range geoFiles {
		wg.Go(func() {
			err := l.fetchAndCache(f)
			if err != nil {
				slog.Warn("fetch and cache geofile", "name", f.name, "err", err)
			}
			errs[i] = err
		})
	}
	wg.Wait()

	return slices.DeleteFunc(errs, func(err error) bool { return err == nil })
}

func (l *Loader) fetchAndCache(f geoFile) error {
	slog.Info("downloading", "url", f.url)

	resp, err := l.fetch(f.url)
	if err != nil {
		return fmt.Errorf("download geodata %s: %w", f.url, err)
	}
	defer resp.Body.Close()

	return l.cache.WriteWith(f.name, func(dst *os.File) error {
		n, err := io.Copy(dst, resp.Body)
		if err != nil {
			return err
		}
		slog.Info("wrote", "file", dst.Name(), "bytes", n)
		return nil
	})
}

func (l *Loader) fetch(url string) (*http.Response, error) {
	var lastErr error
	for _, c := range l.clients {
		resp, err := c.Get(url)
		if err != nil {
			lastErr = err
			slog.Warn("fetch failed, trying next client", "url", url, "err", err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			slog.Warn("fetch failed, trying next client", "url", url, "err", lastErr)
			continue
		}
		return resp, nil
	}
	return nil, fmt.Errorf("all clients failed for %s: %w", url, lastErr)
}
