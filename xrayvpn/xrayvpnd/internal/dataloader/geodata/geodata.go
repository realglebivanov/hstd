package geodata

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/realglebivanov/hstd/xrayvpnd/internal/cache"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/dataloader/httpclient"
)

type GeoFile struct {
	Url  string
	Name string
}

const baseGeodataUrl = "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/"

var geoFiles = []GeoFile{
	{baseGeodataUrl + "geoip.dat", "geoip.dat"},
	{baseGeodataUrl + "geosite.dat", "geosite.dat"},
}

func Load() (stale []GeoFile, err error) {
	for _, f := range geoFiles {
		cr := cache.Read(f.Name)
		switch cr.State {
		case cache.CacheFresh:
			slog.Info("using cached", "file", f.Name)
		case cache.CacheStale:
			slog.Info("using stale", "file", f.Name)
			stale = append(stale, f)
		case cache.CacheMissing:
			if err := tryToDownload(f); err != nil {
				return nil, err
			}
		case cache.CacheError:
			return nil, fmt.Errorf("read %s: %w", f.Name, cr.Err)
		default:
			return nil, fmt.Errorf("unexpected cache state %d for %s", cr.State, f.Name)
		}
	}

	return stale, nil
}

func Download(f GeoFile) error {
	return tryToDownload(f)
}

func Refresh() error {
	errs := make([]error, len(geoFiles))

	var wg sync.WaitGroup
	for i, f := range geoFiles {
		wg.Go(func() {
			errs[i] = tryToDownload(f)
		})
	}
	wg.Wait()

	return errors.Join(errs...)
}

func tryToDownload(f GeoFile) error {
	if err := download(httpclient.Default, f); err != nil {
		slog.Warn("download geodata failed", "url", f.Url, "err", err)
		if err := download(httpclient.Direct, f); err != nil {
			return fmt.Errorf("download geodata %s: %w", f.Url, err)
		}
	}

	return nil
}

func download(client *http.Client, f GeoFile) error {
	slog.Info("downloading", "url", f.Url)
	resp, err := client.Get(f.Url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return cache.WriteWith(f.Name, func(f *os.File) error {
		n, err := io.Copy(f, resp.Body)
		if err != nil {
			return err
		}
		slog.Info("wrote", "file", f.Name(), "bytes", n)
		return nil
	})
}
