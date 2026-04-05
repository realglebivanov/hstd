package main

import (
	"fmt"
	"syscall"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/hstdlib/secret"
	"github.com/realglebivanov/hstd/hstdlib/xrayconf"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/config/store"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init <secret> <server-host> <proxy-host> <pbk> <sni> <sid>",
		Short: "Initialize managed links if state is empty",
		Args:  cobra.ExactArgs(6),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootSecret, err := hstdlib.ParseHexSecret(args[0])
			if err != nil {
				return fmt.Errorf("secret must be hex: %w", err)
			}

			uuid := secret.GenerateClientUUID(0, rootSecret)
			base := xrayconf.VLESSLink{
				UUID:        uuid,
				Port:        443,
				Network:     "tcp",
				Security:    "reality",
				Flow:        "xtls-rprx-vision",
				Fingerprint: "chrome",
				PublicKey:   args[3],
				SNI:         args[4],
				ShortID:     args[5],
			}

			base.Address = args[1]
			serverLink := base.String()

			base.Address = args[2]
			proxyLink := base.String()

			if err := store.ReplaceDefaultLinks(serverLink, proxyLink); err != nil {
				return err
			}

			fmt.Println("links initialized (server active)")
			return send(xrayvpndProcess, syscall.SIGUSR2)
		},
	}
	return cmd
}
