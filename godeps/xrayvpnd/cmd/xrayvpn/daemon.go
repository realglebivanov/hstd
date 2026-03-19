package main

import (
	"syscall"

	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "start",
		Short:              "Start the VPN tunnel (SIGUSR2)",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return send(syscall.SIGUSR2)
		},
	}
}

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "stop",
		Short:              "Stop the VPN tunnel (SIGUSR1)",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return send(syscall.SIGUSR1)
		},
	}
}

func newRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "refresh",
		Short:              "Refresh config and geodata (SIGHUP)",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return send(syscall.SIGHUP)
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "status",
		Short:              "Check daemon status",
		DisableFlagParsing: true,
		Run: func(cmd *cobra.Command, args []string) {
			status()
		},
	}
}
