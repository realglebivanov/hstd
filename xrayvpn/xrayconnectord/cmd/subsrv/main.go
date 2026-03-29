package main

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/server"
)

func main() {
	rootSecret := hstdlib.MustEnvHex("SECRET")

	s, err := server.New(rootSecret)
	if err != nil {
		slog.Error("init server", "err", err)
		os.Exit(1)
	}
	defer s.Close()

	http.HandleFunc("/admin/", s.HandleAdminReq)
	http.HandleFunc("/", s.HandleSubReq)

	credsDir := hstdlib.MustEnv("CREDENTIALS_DIRECTORY")
	certFile := filepath.Join(credsDir, "tls_cert")
	keyFile := filepath.Join(credsDir, "tls_key")

	slog.Info("listening on :8080")
	if err := http.ListenAndServeTLS(":8080", certFile, keyFile, nil); err != nil {
		slog.Error("listen", "err", err)
		os.Exit(1)
	}
}
