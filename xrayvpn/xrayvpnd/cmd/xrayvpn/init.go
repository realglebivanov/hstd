package main

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/config/store"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init <secret> <server-host> <proxy-host> <pbk> <sni> <sid>",
		Short: "Initialize managed links if state is empty",
		Args:  cobra.ExactArgs(6),
		RunE: func(cmd *cobra.Command, args []string) error {
			secret, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("secret must be an integer: %w", err)
			}
			uuid := hstdlib.GenerateClientUUID(secret)

			serverLink := buildVLESSLink(uuid, args[1], args[3], args[4], args[5])
			proxyLink := buildVLESSLink(uuid, args[2], args[3], args[4], args[5])

			initErr := store.InitLinks(serverLink, proxyLink)
			if errors.Is(initErr, store.ErrAlreadyInitialized) {
				fmt.Println("links already initialized, skipping")
				return nil
			}
			if initErr != nil {
				return initErr
			}
			fmt.Println("links initialized (server active)")
			return nil
		},
	}
	return cmd
}

func buildVLESSLink(uuid, host, pbk, sni, sid string) string {
	q := url.Values{}
	q.Set("type", "tcp")
	q.Set("security", "reality")
	q.Set("flow", "xtls-rprx-vision")
	q.Set("fp", "chrome")
	q.Set("pbk", pbk)
	q.Set("sni", sni)
	q.Set("sid", sid)
	return fmt.Sprintf("vless://%s@%s:443?%s", uuid, host, q.Encode())
}
