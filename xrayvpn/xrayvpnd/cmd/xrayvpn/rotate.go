package main

import (
	"fmt"
	"log"
	"strconv"
	"syscall"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/config/store"
	"github.com/spf13/cobra"
)

func newRotateCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "rotate <secret>",
		Short:              "Rotate client UUID based on today's date and a shared secret",
		DisableFlagParsing: true,
		Args:               cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			secret, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("secret must be an integer: %w", err)
			}
			uuid := hstdlib.GenerateClientUUID(secret)
			log.Printf("rotating client_id to %s", uuid)

			if err := store.RotateUUID(uuid); err != nil {
				return err
			}

			fmt.Println("client_id rotated")
			return send(xrayvpndProcess, syscall.SIGUSR2)
		},
	}
}
