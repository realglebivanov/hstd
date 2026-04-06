package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/sessions"
	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/hstdlib/secret"
	"github.com/realglebivanov/hstd/hstdlib/xrayconf"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/cidrs"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/client"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/db"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/server/admin/view"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/server/broadcast"
)

type auth struct {
	sessions          *sessions.CookieStore
	sessionOpts       *sessions.Options
	adminUser         string
	adminPasswordHash string
}

type Server struct {
	db            *db.DB
	rootSecret    []byte
	legacySubPath string
	broadcast     *broadcast.Broadcast
	auth          *auth
	view          *view.Builder
	serverConfigs []*client.ServerConfig
	routingRules  []xrayconf.RouteRule
	httpServer    *http.Server
	cidrs         *cidrs.CIDRs
}

const configPath = "/etc/subsrv/config.json"

type subsrvConfig struct {
	Servers      []*client.ServerConfig `json:"servers"`
	RoutingRules []xrayconf.RouteRule   `json:"routingRules"`
}

func New(rootSecret []byte) (*Server, error) {
	db, err := db.Open()
	if err != nil {
		return nil, fmt.Errorf("open database: %v", err)
	}

	cfgData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", configPath, err)
	}
	var cfg subsrvConfig
	if err := json.Unmarshal(cfgData, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", configPath, err)
	}

	return &Server{
		db:            db,
		rootSecret:    rootSecret,
		routingRules:  cfg.RoutingRules,
		cidrs:         cidrs.New(),
		legacySubPath: hstdlib.MustEnv("SUB_PATH"),
		broadcast:     broadcast.New(),
		auth: &auth{
			sessions: sessions.NewCookieStore(secret.DeriveCookieKey(rootSecret)),
			sessionOpts: &sessions.Options{
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
				Domain:   hstdlib.MustEnv("PROXY_DOMAIN"),
				MaxAge:   int(time.Hour.Seconds()),
				Secure:   true,
				Path:     "/",
			},
			adminUser:         hstdlib.MustEnv("ADMIN_USER"),
			adminPasswordHash: hstdlib.MustEnv("ADMIN_PASSWORD_HASH"),
		},
		view: &view.Builder{
			RootSecret:  rootSecret,
			ProxyDomain: hstdlib.MustEnv("PROXY_DOMAIN"),
		},
		serverConfigs: cfg.Servers,
	}, nil
}

func (s *Server) Start() error {
	s.cidrs.StartRefresh(2 * time.Hour)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /admin/ws", s.handleAdminWS)
	mux.HandleFunc("GET /admin/", s.handleAdminPage)
	mux.HandleFunc("GET /{link}", s.handleSubReq)

	credsDir := hstdlib.MustEnv("CREDENTIALS_DIRECTORY")
	s.httpServer = &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("listening on :8080")
	return s.httpServer.ListenAndServeTLS(
		filepath.Join(credsDir, "tls_cert"),
		filepath.Join(credsDir, "tls_key"),
	)
}

func (s *Server) Stop() {
	s.cidrs.Stop()
	s.broadcast.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		slog.Error("http shutdown", "err", err)
	}

	if err := s.db.Close(); err != nil {
		slog.Error("close db", "err", err)
	}
}
