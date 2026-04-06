package main

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/config/store"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/conns"
	"github.com/spf13/cobra"
)

func newConnCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "conn",
		Short: "Manage VLESS conns",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "add <url>",
			Short: "Add a VLESS conn",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := conns.Add(args[0]); err != nil {
					return err
				}
				fmt.Println("conn added")
				return reloadDaemon()
			},
		},
		&cobra.Command{
			Use:                "remove <id>",
			Short:              "Remove a conn by ID",
			DisableFlagParsing: true,
			Args:               cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				activeChanged, err := store.RemoveConn(args[0])
				if err != nil {
					return err
				}
				fmt.Println("conn removed")
				if activeChanged {
					return reloadDaemon()
				}
				return nil
			},
		},
		&cobra.Command{
			Use:                "choose <id>",
			Short:              "Set the active conn by ID",
			DisableFlagParsing: true,
			Args:               cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := store.ChooseConn(args[0]); err != nil {
					return err
				}
				fmt.Println("active conn changed")
				return reloadDaemon()
			},
		},
		&cobra.Command{
			Use:                "list",
			Short:              "List all saved conns",
			DisableFlagParsing: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				st, err := store.GetState()
				if err != nil {
					return err
				}
				table := tablewriter.NewWriter(os.Stdout)
				table.Header("", "ID", "Conn", "Sub")
				if len(st.Conns) == 0 {
					table.Footer("No conns saved")
				}
				for _, l := range st.Conns {
					active := ""
					if l.ID == st.ActiveID {
						active = "*"
					}
					table.Append(active, l.ID, l.Summary(), l.SubID)
				}
				table.Render()
				return nil
			},
		},
	)
	return cmd
}