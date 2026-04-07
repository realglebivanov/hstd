package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/config/repo"
	"github.com/spf13/cobra"
)

type managedProcess struct {
	name    string
	pidFile string
}

var (
	db *repo.DB

	xrayvpndProcess = managedProcess{
		name:    "daemon",
		pidFile: hstdlib.XrayVpnPIDFile,
	}
	tun2socksdProcess = managedProcess{
		name:    "tunnel",
		pidFile: hstdlib.Tun2SocksPIDFile,
	}
)

func main() {
	root := &cobra.Command{
		Use:          "xrayvpn",
		Short:        "Control the xrayvpn tunnel and daemon",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			db, err = repo.Open()
			return err
		},
	}

	root.AddCommand(
		newStartCmd(),
		newStopCmd(),
		newRefreshCmd(),
		newConnCmd(),
		newSubCmd(),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func reloadDaemon() error {
	return send(xrayvpndProcess, syscall.SIGUSR2)
}

func send(proc managedProcess, sig syscall.Signal) error {
	p, err := findProcess(proc)
	if err != nil {
		return err
	}
	if err = p.Signal(sig); err != nil {
		return fmt.Errorf("signal failed: %w", err)
	}
	slog.Info("sent signal", "proc", proc.name, "pid", p.Pid)
	return nil
}

func findProcess(proc managedProcess) (*os.Process, error) {
	pid, err := readPID(proc)
	if err != nil {
		return nil, fmt.Errorf("%s not running (no pid file)", proc.name)
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("find %s process %d: %w", proc.name, pid, err)
	}
	if err = p.Signal(syscall.Signal(0)); err != nil {
		return nil, fmt.Errorf("%s pid %d not alive", proc.name, pid)
	}
	return p, nil
}

func readPID(proc managedProcess) (int, error) {
	data, err := os.ReadFile(proc.pidFile)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}
