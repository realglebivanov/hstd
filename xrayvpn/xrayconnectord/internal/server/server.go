package server

import (
	"fmt"
	"log/slog"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/client"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/db"
)

type Server struct {
	db                *db.DB
	legacySubPath     string
	rootSecret        []byte
	adminUser         string
	adminPasswordHash string
	proxyDomain       string
	serverConfigs     []*client.ServerConfig
}

func New(rootSecret []byte) (*Server, error) {
	db, err := db.Open()
	if err != nil {
		return nil, fmt.Errorf("open database: %v", err)
	}

	return &Server{
		db:                db,
		legacySubPath:     hstdlib.MustEnv("SUB_PATH"),
		rootSecret:        rootSecret,
		adminUser:         hstdlib.MustEnv("ADMIN_USER"),
		adminPasswordHash: hstdlib.MustEnv("ADMIN_PASSWORD_HASH"),
		proxyDomain:       hstdlib.MustEnv("PROXY_DOMAIN"),
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

func (s *Server) Close() {
	if err := s.db.Close(); err != nil {
		slog.Error("close db", "err", err)
	}
}
