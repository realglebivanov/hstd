package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/sessions"
	"github.com/realglebivanov/hstd/hstdlib"
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
	httpServer    *http.Server
}

func New(rootSecret []byte) (*Server, error) {
	db, err := db.Open()
	if err != nil {
		return nil, fmt.Errorf("open database: %v", err)
	}

	return &Server{
		db:            db,
		rootSecret:    rootSecret,
		legacySubPath: hstdlib.MustEnv("SUB_PATH"),
		broadcast:     broadcast.New(),
		auth: &auth{
			sessions: sessions.NewCookieStore(rootSecret),
			sessionOpts: &sessions.Options{
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
				Domain:   hstdlib.MustEnv("PROXY_DOMAIN"),
				MaxAge:   int(time.Hour.Seconds()),
				Secure:   true,
			},
			adminUser:         hstdlib.MustEnv("ADMIN_USER"),
			adminPasswordHash: hstdlib.MustEnv("ADMIN_PASSWORD_HASH"),
		},
		view: &view.Builder{
			RootSecret:  rootSecret,
			ProxyDomain: hstdlib.MustEnv("PROXY_DOMAIN"),
		},
		serverConfigs: []*client.ServerConfig{{
			Remark:     "Обычный ВПН",
			Host:       hstdlib.MustEnv("SERVER_HOST"),
			RealityPbk: hstdlib.MustEnv("REALITY_PBK"),
			RealitySni: hstdlib.MustEnv("REALITY_SNI"),
			RealitySid: hstdlib.MustEnv("REALITY_SID"),
		}, {
			Remark:     "Обход белых списков",
			Host:       hstdlib.MustEnv("PROXY_HOST"),
			RealityPbk: hstdlib.MustEnv("REALITY_PBK"),
			RealitySni: hstdlib.MustEnv("REALITY_SNI"),
			RealitySid: hstdlib.MustEnv("REALITY_SID"),
		}}}, nil
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /admin/ws", s.handleAdminWS)
	mux.HandleFunc("GET /admin/", s.handleAdminPage)
	mux.HandleFunc("GET /{link}", s.handleSubReq)

	credsDir := hstdlib.MustEnv("CREDENTIALS_DIRECTORY")
	s.httpServer = &http.Server{Addr: ":8080", Handler: mux}

	slog.Info("listening on :8080")
	return s.httpServer.ListenAndServeTLS(
		filepath.Join(credsDir, "tls_cert"),
		filepath.Join(credsDir, "tls_key"),
	)
}

func (s *Server) Stop() {
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
