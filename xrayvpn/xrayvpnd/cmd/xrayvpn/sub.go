package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/hstdlib/sublink"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/config/store"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/conns"
	"github.com/spf13/cobra"
)

func newSubCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sub",
		Short: "Manage subscriptions",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "add <url>",
			Short: "Add a subscription URL",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := store.AddSub(args[0]); err != nil {
					return err
				}
				fmt.Println("subscription added")
				return nil
			},
		},
		&cobra.Command{
			Use:   "url <host> <root-secret-hex>",
			Short: "Print the full subscription URL for a managed host",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				rootSecret := hstdlib.MustHex(args[1])
				subPath, err := sublink.SubPath(0, rootSecret)
				if err != nil {
					return err
				}
				fmt.Printf("https://%s:8080/%s\n", args[0], subPath)
				return nil
			},
		},
		&cobra.Command{
			Use:   "remove <id>",
			Short: "Remove a subscription by ID",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := store.RemoveSub(args[0]); err != nil {
					return err
				}
				fmt.Println("subscription removed")
				return reloadDaemon()
			},
		},
		&cobra.Command{
			Use:   "sync",
			Short: "Sync conns from all subscriptions",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				n, err := conns.SyncAll()
				if err != nil {
					return err
				}
				fmt.Printf("synced %d conns\n", n)
				if err := reloadDaemon(); err != nil {
					slog.Warn("failed to reload daemon", "err", err)
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "List all subscriptions",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				st, err := store.GetState()
				if err != nil {
					return err
				}
				table := tablewriter.NewWriter(os.Stdout)
				table.Header("ID", "URL")
				if len(st.Subs) == 0 {
					table.Footer("No subscriptions")
				}
				for _, s := range st.Subs {
					url := s.URL
					if len(url) > 60 {
						url = url[:57] + "..."
					}
					table.Append(s.ID, url)
				}
				table.Render()
				return nil
			},
		},
	)
	return cmd
}