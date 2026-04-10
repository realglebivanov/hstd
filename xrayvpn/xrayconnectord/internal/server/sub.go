package server

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/realglebivanov/hstd/hstdlib/ruapps"
	"github.com/realglebivanov/hstd/hstdlib/secret"
	"github.com/realglebivanov/hstd/hstdlib/sublink"
	"github.com/realglebivanov/hstd/hstdlib/xrayconf"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/client"
)

func (s *Server) handleSubReq(w http.ResponseWriter, r *http.Request) {
	l, httpErr := s.validateSubInput(r)
	if httpErr != nil {
		http.Error(w, httpErr.reason, httpErr.code)
		return
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	device := r.UserAgent() + " @ " + ip
	if r.UserAgent() == "" {
		device = ip
	}

	li, err := s.db.TrackDevice(l, device)
	if err != nil {
		slog.Error("track device", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	row, err := s.view.BuildRow(li)
	if err != nil {
		slog.Error("build row for broadcast", "err", err)
	} else {
		s.broadcast.Broadcast(row, nil)
	}

	uuid := secret.GenerateClientUUID(l.Index, s.rootSecret)
	rules := xrayconf.ExpandRules(s.routingRules, s.cidrs.Get())
	configs := client.BuildConfigs(uuid, s.serverConfigs, rules)

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(configs); err != nil {
		slog.Error("encode response", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("profile-update-interval", "1")
	w.Header().Set("profile-title", "base64:"+base64.StdEncoding.EncodeToString([]byte("hstd")))
	w.Header().Set("announce", "base64:"+base64.StdEncoding.EncodeToString([]byte("Не делитесь ссылкой на подписку — иначе она может быть заблокирована")))
	w.Header().Set("hide-settings", "1")
	w.Header().Set("per-app-proxy-mode", "bypass")
	w.Header().Set("per-app-proxy-list", ruapps.Joined())

	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		buf.WriteTo(gz)
		gz.Close()
	} else {
		buf.WriteTo(w)
	}
}

type httpError struct {
	reason string
	code   int
}

func (s *Server) validateSubInput(r *http.Request) (*sublink.Sublink, *httpError) {
	l, httpErr := s.buildSubLink(r)
	if httpErr != nil {
		return nil, httpErr
	}

	isEnabled, err := s.db.IsEnabled(l)
	if err != nil {
		slog.Error("check link enabled", "err", err)
		return nil, &httpError{"internal error", http.StatusInternalServerError}
	}
	if !isEnabled {
		slog.Info("link disabled", "index", l.Index)
		return nil, &httpError{"bad request", http.StatusBadRequest}
	}
	return l, nil
}

func (s *Server) buildSubLink(r *http.Request) (*sublink.Sublink, *httpError) {
	src := r.PathValue("link")

	if src == s.legacySubPath {
		return sublink.New(0, s.rootSecret), nil
	}

	sl, err := sublink.Unmarshal(src)
	if err != nil {
		slog.Warn("decode input link", "err", err)
		return nil, &httpError{"bad request", http.StatusBadRequest}
	}

	if !sl.IsValid(s.rootSecret) {
		return nil, &httpError{"bad request", http.StatusBadRequest}
	}

	return sl, nil
}
