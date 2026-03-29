package server

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/link"
	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
)

//go:embed admin.html
var adminTmplSrc string
var adminTmpl = template.Must(template.New("admin").Parse(adminTmplSrc))

func (s *Server) HandleAdminReq(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok || user != s.adminUser || bcrypt.CompareHashAndPassword([]byte(s.adminPasswordHash), []byte(pass)) != nil {
		w.Header().Set("WWW-Authenticate", `Basic realm="admin"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.adminPage(w)
	case http.MethodPost:
		s.adminAction(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) adminPage(w http.ResponseWriter) {
	links, err := s.db.List(hstdlib.XrayClientCount)
	if err != nil {
		slog.Error("admin list", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	var rows []*linkRow
	for _, l := range links {
		lr, err := s.buildLinkRow(&l)
		if err != nil {
			slog.Error("build link row", "err", err)
			continue
		}
		rows = append(rows, lr)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := adminTmpl.Execute(w, rows); err != nil {
		slog.Error("execute admin tpl", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) adminAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	idx, err := strconv.Atoi(r.FormValue("index"))
	if err != nil {
		http.Error(w, "bad index", http.StatusBadRequest)
		return
	}

	switch r.FormValue("action") {
	case "enable":
		err = s.db.SetEnabled(idx, true)
	case "disable":
		err = s.db.SetEnabled(idx, false)
	case "comment":
		err = s.db.SetComment(idx, r.FormValue("comment"))
	default:
		http.Error(w, "unknown action", http.StatusBadRequest)
		return
	}

	if err != nil {
		slog.Error("admin action", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/", http.StatusSeeOther)
}

type linkRow struct {
	Index   int
	URL     string
	QR      template.URL
	Devices []string
	Enabled bool
	Comment string
}

func (s *Server) buildLinkRow(l *link.LinkInfo) (*linkRow, error) {
	hex, err := link.New(l.Index, s.rootSecret).Marshal()
	if err != nil {
		return nil, fmt.Errorf("admin marshal link %d: %v", l.Index, err)
	}

	url := fmt.Sprintf("https://%s:8080/%s", s.proxyDomain, hex)
	png, err := qrcode.Encode(url, qrcode.Highest, 200)
	if err != nil {
		return nil, fmt.Errorf("admin qr link %d: %v", l.Index, err)
	}
	qr := template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(png))

	var devices []string
	if l.Devices != "" {
		devices = strings.Split(l.Devices, "\n")
	}

	return &linkRow{
		Index:   l.Index,
		URL:     url,
		QR:      qr,
		Devices: devices,
		Enabled: l.Enabled,
		Comment: l.Comment,
	}, nil
}
