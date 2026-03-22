package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/olekukonko/tablewriter"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/config/store"
	"github.com/spf13/cobra"
)

func newLinkCmds() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link",
		Short: "Manage VLESS links",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:                "add <url>",
			Short:              "Add a VLESS link and activate it",
			DisableFlagParsing: true,
			Args:               cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := store.AddLink(args[0]); err != nil {
					return err
				}
				fmt.Println("link added")
				return send(xrayvpndProcess, syscall.SIGUSR2)
			},
		},
		&cobra.Command{
			Use:                "remove <id>",
			Short:              "Remove a link by ID",
			DisableFlagParsing: true,
			Args:               cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				activeChanged, err := store.RemoveLink(args[0])
				if err != nil {
					return err
				}
				fmt.Println("link removed")
				if activeChanged {
					return send(xrayvpndProcess, syscall.SIGUSR2)
				}
				return nil
			},
		},
		&cobra.Command{
			Use:                "choose <id>",
			Short:              "Set the active link by ID",
			DisableFlagParsing: true,
			Args:               cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := store.ChooseLink(args[0]); err != nil {
					return err
				}
				fmt.Println("active link changed")
				return send(xrayvpndProcess, syscall.SIGUSR2)
			},
		},
		&cobra.Command{
			Use:                "list",
			Short:              "List all saved links",
			DisableFlagParsing: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				st, err := store.GetState()
				if err != nil {
					return err
				}
				table := tablewriter.NewWriter(os.Stdout)
				table.Header("", "ID", "Link")
				if len(st.Links) == 0 {
					table.Footer("No links saved")
				}
				for _, l := range st.Links {
					active := ""
					if l.ID == st.ActiveID {
						active = "*"
					}
					table.Append(active, l.ID, l.Summary())
				}
				table.Render()
				return nil
			},
		},
	)
	return cmd
}
