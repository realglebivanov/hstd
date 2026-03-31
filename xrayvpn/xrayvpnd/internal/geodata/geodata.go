package geodata

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"log/slog"
	"net/http"
	"os"

	"github.com/realglebivanov/hstd/xrayvpnd/internal/cache"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/httpclient"
)

type geodataFile struct {
	url  string
	name string
}

const baseGeodataUrl = "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/"

var geodataFiles = []geodataFile{
	{baseGeodataUrl + "geoip.dat", "geoip.dat"},
	{baseGeodataUrl + "geosite.dat", "geosite.dat"},
}

func Load() error {
	for _, f := range geodataFiles {
		cr := cache.Read(f.name)
		switch cr.State {
		case cache.CacheFresh:
			slog.Info("using cached", "file", f.name)
		case cache.CacheStale:
			slog.Info("using stale, will refresh in background", "file", f.name)
			go tryToDownload(f)
		case cache.CacheMissing:
			if err := tryToDownload(f); err != nil {
				return err
			}
		case cache.CacheError:
			return fmt.Errorf("read %s: %w", f.name, cr.Err)
		default:
			return fmt.Errorf("unexpected cache state %d for %s", cr.State, f.name)
		}
	}

	return nil
}

func Refresh() error {
	errs := make([]error, len(geodataFiles))

	var wg sync.WaitGroup
	for i, f := range geodataFiles {
		wg.Go(func() {
			errs[i] = tryToDownload(f)
		})
	}
	wg.Wait()

	return errors.Join(errs...)
}

func tryToDownload(f geodataFile) error {
	if err := download(httpclient.Default, f); err != nil {
		slog.Warn("download geodata failed", "url", f.url, "err", err)
		if err := download(httpclient.Direct, f); err != nil {
			return fmt.Errorf("download geodata %s: %w", f.url, err)
		}
	}

	return nil
}

func download(client *http.Client, f geodataFile) error {
	slog.Info("downloading", "url", f.url)
	resp, err := client.Get(f.url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return cache.WriteWith(f.name, func(f *os.File) error {
		n, err := io.Copy(f, resp.Body)
		if err != nil {
			return err
		}
		slog.Info("wrote", "file", f.Name(), "bytes", n)
		return nil
	})
}
