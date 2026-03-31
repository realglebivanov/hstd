package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/sessions"
	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/client"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/db"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/server/admin/view"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/server/broadcast"
	"golang.org/x/crypto/bcrypt"
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

func (s *Server) Start() {
	http.HandleFunc("GET /admin/ws", s.handleAdminWS)
	http.HandleFunc("GET /admin/", s.handleAdminPage)
	http.HandleFunc("/", s.handleSubReq)

	credsDir := hstdlib.MustEnv("CREDENTIALS_DIRECTORY")
	certFile := filepath.Join(credsDir, "tls_cert")
	keyFile := filepath.Join(credsDir, "tls_key")

	slog.Info("listening on :8080")
	if err := http.ListenAndServeTLS(":8080", certFile, keyFile, nil); err != nil {
		slog.Error("listen", "err", err)
		os.Exit(1)
	}
}

func (s *Server) basicAuth(w http.ResponseWriter, r *http.Request) bool {
	sesh, _ := s.auth.sessions.Get(r, "hstd#xrayconnectord#subsrv")

	if _, ok := sesh.Values["id"]; ok {
		return true
	}

	user, pass, ok := r.BasicAuth()
	if !ok || user != s.auth.adminUser {
		return false
	}

	if bcrypt.CompareHashAndPassword([]byte(s.auth.adminPasswordHash), []byte(pass)) != nil {
		return false
	}

	sesh.Options = s.auth.sessionOpts
	sesh.Values["id"] = true

	return sesh.Save(r, w) == nil
}

func (s *Server) Stop() {
	if err := s.db.Close(); err != nil {
		slog.Error("close db", "err", err)
	}
}
