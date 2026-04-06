package view

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"log/slog"
	"strings"

	"github.com/realglebivanov/hstd/hstdlib/sublink"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/db"
	qrcode "github.com/skip2/go-qrcode"
)

type Row struct {
	Index   int      `json:"index"`
	Version int      `json:"version"`
	URL     string   `json:"url"`
	QR      string   `json:"qr"`
	Devices []string `json:"devices"`
	Enabled bool     `json:"enabled"`
	Comment string   `json:"comment"`
}

type HTMLContext struct {
	Vue template.JS
}

type Builder struct {
	RootSecret  []byte
	ProxyDomain string
}

//go:embed admin.html
var adminTmplSrc string
var AdminTmpl = template.Must(template.New("admin").Parse(adminTmplSrc))

//go:embed vue.global.prod.js
var vueJS []byte

func (b *Builder) BuildHTMLContext() *HTMLContext {
	return &HTMLContext{template.JS(vueJS)}
}

func (b *Builder) BuildRows(links []db.SublinkInfo) ([]*Row, error) {
	rows := []*Row{}
	for _, l := range links {
		r, err := b.BuildRow(&l)
		if err != nil {
			slog.Warn("build link row", "err", err)
			continue
		}
		rows = append(rows, r)
	}
	return rows, nil
}

func (b *Builder) BuildRow(info *db.SublinkInfo) (*Row, error) {
	url, qr, err := b.buildUrlAndQR(info)
	if err != nil {
		return nil, err
	}

	devices := []string{}
	if info.Devices != "" {
		devices = strings.Split(info.Devices, "\n")
	}

	return &Row{
		Index:   info.Index,
		Version: info.Version,
		URL:     url,
		QR:      "data:image/png;base64," + qr,
		Devices: devices,
		Enabled: info.Enabled,
		Comment: info.Comment,
	}, nil
}

func (b *Builder) buildUrlAndQR(info *db.SublinkInfo) (string, string, error) {
	subPath, err := sublink.SubPath(info.Index, b.RootSecret)
	if err != nil {
		return "", "", err
	}

	url := fmt.Sprintf("https://%s:8080/%s", b.ProxyDomain, subPath)
	png, err := qrcode.Encode(url, qrcode.Highest, 200)
	if err != nil {
		return "", "", fmt.Errorf("qr link %d: %v", info.Index, err)
	}
	qr := base64.StdEncoding.EncodeToString(png)

	return url, qr, nil
}
