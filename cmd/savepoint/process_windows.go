//go:build windows

package main

import (
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

const createNewProcessGroup = 0x00000200

// detachedProcAttr starts the daemon in its own process group so it
// isn't killed by a Ctrl+C sent to the parent console window.
func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: createNewProcessGroup}
}

// isProcessRunning shells out to `tasklist` since Windows has no
// equivalent of Unix's "signal 0" liveness check without importing
// golang.org/x/sys/windows (which we're avoiding to stay dependency-free).
func isProcessRunning(pid int) bool {
	out, err := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid), "/NH").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), strconv.Itoa(pid))
}

// terminateProcess kills the daemon. Windows doesn't have graceful
// SIGTERM-style signals for arbitrary processes, so this is a hard kill.
func terminateProcess(proc *os.Process) error {
	return proc.Kill()
}

// registerStopSignals wires up Ctrl+C. Windows consoles deliver this as
// os.Interrupt.
func registerStopSignals(sig chan os.Signal) {
	signal.Notify(sig, os.Interrupt)
}
