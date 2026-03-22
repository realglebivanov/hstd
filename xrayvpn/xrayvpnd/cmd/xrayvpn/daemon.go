package main

import (
	"syscall"

	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "start",
		Short:              "Start the VPN tunnel",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return send(tun2socksdProcess, syscall.SIGUSR2)
		},
	}
}

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "stop",
		Short:              "Stop the VPN tunnel",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return send(tun2socksdProcess, syscall.SIGUSR1)
		},
	}
}

func newRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "refresh",
		Short:              "Refresh daemon config and geodata",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return send(xrayvpndProcess, syscall.SIGHUP)
		},
	}
}
