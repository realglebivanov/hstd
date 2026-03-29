package server

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"

	"github.com/realglebivanov/hstd/hstdlib/secret"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/client"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/link"
)

func (s *Server) HandleSubReq(w http.ResponseWriter, r *http.Request) {
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

	if err := s.db.TrackDevice(l, device); err != nil {
		slog.Error("track device", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	uuid := secret.GenerateClientUUID(l.Index, s.rootSecret)
	configs := client.BuildConfigs(uuid, s.serverConfigs)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("profile-update-interval", "1")
	w.Header().Set("profile-title", "base64:"+base64.StdEncoding.EncodeToString([]byte("hstd")))

	if err := json.NewEncoder(w).Encode(configs); err != nil {
		slog.Error("encode response", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
}

type httpError struct {
	reason string
	code   int
}

func (s *Server) validateSubInput(r *http.Request) (*link.Link, *httpError) {
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

func (s *Server) buildSubLink(r *http.Request) (*link.Link, *httpError) {
	src := r.URL.Path[1:]

	if src == s.legacySubPath {
		return link.New(0, s.rootSecret), nil
	}

	l, err := link.Unmarshal(src)
	if err != nil {
		slog.Warn("decode input link", "err", err)
		return nil, &httpError{"bad request", http.StatusBadRequest}
	}

	if !l.IsValid(s.rootSecret) {
		return nil, &httpError{"bad request", http.StatusBadRequest}
	}

	return l, nil
}
