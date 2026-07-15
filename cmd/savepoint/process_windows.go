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

func detachedProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: createNewProcessGroup}
}

func isProcessRunning(pid int) bool {
	out, err := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid), "/NH").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), strconv.Itoa(pid))
}

func terminateProcess(proc *os.Process) error {
	return proc.Kill()
}

func registerStopSignals(sig chan os.Signal) {
	signal.Notify(sig, os.Interrupt)
}