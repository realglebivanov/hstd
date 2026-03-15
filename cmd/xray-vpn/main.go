// xray-vpn sends signals to the xray-vpnd daemon.
//
//	xray-vpn start   → SIGUSR2
//	xray-vpn stop    → SIGUSR1
//	xray-vpn status  → check PID
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

const pidFile = "/run/xray-vpn/xray-vpn.pid"

func main() {
	if len(os.Args) != 2 {
		usage()
	}
	switch os.Args[1] {
	case "start":
		send(syscall.SIGUSR2)
	case "stop":
		send(syscall.SIGUSR1)
	case "status":
		status()
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <start|stop|status>\n", os.Args[0])
	os.Exit(1)
}

func readPID() (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func alive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}

func send(sig syscall.Signal) {
	pid, err := readPID()
	if err != nil {
		fmt.Fprintln(os.Stderr, "daemon not running (no pid file)")
		os.Exit(1)
	}
	if !alive(pid) {
		fmt.Fprintf(os.Stderr, "daemon pid %d not alive\n", pid)
		os.Exit(1)
	}
	p, _ := os.FindProcess(pid)
	if err := p.Signal(sig); err != nil {
		fmt.Fprintf(os.Stderr, "signal failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("sent to pid %d\n", pid)
}

func status() {
	pid, err := readPID()
	if err != nil {
		fmt.Println("daemon is not running.")
		return
	}
	if alive(pid) {
		fmt.Printf("daemon running (pid %d)\n", pid)
	} else {
		fmt.Printf("stale pid %d\n", pid)
	}
}
