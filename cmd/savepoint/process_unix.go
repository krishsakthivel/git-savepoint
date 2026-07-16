//go:build !windows

package main

import (
	"os"
	"os/signal"
	"syscall"
)

// detachedProcAttr fully detaches the daemon into its own session so it
// survives the parent shell exiting.
func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}

// isProcessRunning uses the classic "signal 0" trick: sending signal 0
// doesn't actually deliver anything, it just checks whether the kernel
// would let us signal that pid.
func isProcessRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// terminateProcess asks the daemon to shut down gracefully.
func terminateProcess(proc *os.Process) error {
	return proc.Signal(syscall.SIGTERM)
}

// registerStopSignals wires up Ctrl+C (SIGINT) and a graceful
// `kill`/service-manager shutdown (SIGTERM).
func registerStopSignals(sig chan os.Signal) {
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
}
