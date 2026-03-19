package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:          "xrayvpn",
		Short:        "Control the xrayvpnd daemon",
		SilenceUsage: true,
	}

	root.AddCommand(
		newStartCmd(),
		newStopCmd(),
		newRefreshCmd(),
		newStatusCmd(),
		newLinkCmds(),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func send(sig syscall.Signal) error {
	p, err := findDaemon()
	if err != nil {
		return err
	}
	if err = p.Signal(sig); err != nil {
		return fmt.Errorf("signal failed: %w", err)
	}
	log.Printf("sent to pid %d\n", p.Pid)
	return nil
}

func status() {
	p, err := findDaemon()
	if err != nil {
		log.Println("daemon is not running.")
		return
	}
	log.Printf("daemon running (pid %d)\n", p.Pid)
}

func readPID() (int, error) {
	data, err := os.ReadFile(hstdlib.XrayVpnPIDFile)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func findDaemon() (*os.Process, error) {
	pid, err := readPID()
	if err != nil {
		return nil, fmt.Errorf("daemon not running (no pid file)")
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("find process %d: %w", pid, err)
	}
	if err = p.Signal(syscall.Signal(0)); err != nil {
		return nil, fmt.Errorf("daemon pid %d not alive", pid)
	}
	return p, nil
}
