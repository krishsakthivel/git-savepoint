// the git-savepoint cli
//
//	git-savepoint start [--daemon]
//	git-savepoint status
//	git-savepoint timeline
//	git-savepoint restore <checkpoint-id>
//	git-savepoint stop
//
// hopefully these should be the commands
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"git-savepoint/internal/checkpoint"
	"git-savepoint/internal/gitutil"
	"git-savepoint/internal/restore"
	"git-savepoint/internal/storage"
	"git-savepoint/internal/watcher"
)

func main() {
	if len(os.Args) < 2 {
		cmdQuickToggle()
		return
	}

	// install and help dont need to be run inside a repo, so handle
	// those before we go looking for one
	switch os.Args[1] {
	case "install":
		fatalIf(installToPath())
		return
	case "-h", "--help", "help":
		usage()
		return
	}

	cwd, err := os.Getwd()
	fatalIf(err)
	repoRoot, err := gitutil.RepoRoot(cwd)
	fatalIf(err)

	switch os.Args[1] {
	case "start":
		cmdStart(repoRoot, os.Args[2:])
	case "status":
		cmdStatus(repoRoot)
	case "timeline":
		cmdTimeline(repoRoot)
	case "restore":
		cmdRestore(repoRoot, os.Args[2:])
	case "stop":
		cmdStop(repoRoot)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`git-savepoint - a local-first time machine for your Git working tree

Usage:
  git-savepoint install            copy itself onto your PATH, so "git-savepoint" just works anywhere
  git-savepoint start [--daemon]   watch the repo and checkpoint automatically
  git-savepoint status             show whether the watcher is running and recent activity
  git-savepoint timeline           list checkpoints, newest last
  git-savepoint restore <id>       restore the working tree to a checkpoint (id, timestamp, or "latest")
  git-savepoint stop               stop the background watcher

Running git-savepoint with no arguments (e.g. double-clicking the .exe)
starts watching the current folder, and running it again stops it.
`)
}

// what runs when you just double click the exe, no args
// since theres no terminal to type into, make double click a toggle:
// not running yet -> start watching, keep window open
// already running -> stop it, so clicking again turns it off
func cmdQuickToggle() {
	cwd, err := os.Getwd()
	fatalIf(err)

	repoRoot, err := gitutil.RepoRoot(cwd)
	if err != nil {
		fmt.Println("This folder isn't a Git repository yet.")
		fmt.Println("Run `git init` here first, then run git-savepoint again.")
		pauseForExit()
		return
	}

	if pid, running := runningPID(repoRoot); running {
		proc, err := os.FindProcess(pid)
		if err == nil {
			terminateProcess(proc)
		}
		os.Remove(pidFile(repoRoot))
		fmt.Printf("Stopped watching %s (was pid %d)\n", repoRoot, pid)
		pauseForExit()
		return
	}

	fmt.Println("git-savepoint watching", repoRoot)
	fmt.Println("Leave this window open while you work.")
	fmt.Println("Run git-savepoint again (double-click it) to stop, or press Ctrl+C.")
	fmt.Println()

	writePID(repoRoot, os.Getpid())
	defer os.Remove(pidFile(repoRoot))
	runWatchLoop(repoRoot)

	fmt.Println("stopped watching")
	pauseForExit()
}

// keeps the window open after double click so it doesnt just flash and close
func pauseForExit() {
	fmt.Print("\nPress Enter to close this window...")
	fmt.Scanln()
}

func fatalIf(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// stop/start a pid file under git

func pidFile(repoRoot string) string {
	gitDir, err := gitutil.GitDir(repoRoot)
	if err != nil {
		gitDir = filepath.Join(repoRoot, ".git")
	}
	return filepath.Join(gitDir, "git-savepoint.pid")
}

func logFile(repoRoot string) string {
	gitDir, err := gitutil.GitDir(repoRoot)
	if err != nil {
		gitDir = filepath.Join(repoRoot, ".git")
	}
	return filepath.Join(gitDir, "git-savepoint.log")
}

func cmdStart(repoRoot string, args []string) {
	daemon := false
	childProcess := false
	for _, a := range args {
		switch a {
		case "--daemon", "-d":
			daemon = true
		case "--__internal-foreground":
			childProcess = true
		}
	}

	if pid, running := runningPID(repoRoot); running {
		fmt.Printf("git-savepoint is already watching this repo (pid %d)\n", pid)
		return
	}

	if daemon {
		startDaemon(repoRoot)
		return
	}

	fmt.Println("git-savepoint watching", repoRoot)
	if !childProcess {
		fmt.Println("(run with --daemon to background this; Ctrl+C to stop)")
	}
	writePID(repoRoot, os.Getpid())
	defer os.Remove(pidFile(repoRoot))

	runWatchLoop(repoRoot)
}

func startDaemon(repoRoot string) {
	lf, err := os.Create(logFile(repoRoot))
	fatalIf(err)
	defer lf.Close()

	exe, err := os.Executable()
	fatalIf(err)

	// note for future, we dont write the pid file on here by purpose
	// (if we did, the child could read it before its actually legit and
	// think its a dupe of itself)
	proc, err := os.StartProcess(exe, []string{exe, "start", "--__internal-foreground"}, &os.ProcAttr{
		Dir:   repoRoot,
		Files: []*os.File{nil, lf, lf},
		Sys:   detachedProcAttr(),
	})
	fatalIf(err)

	fmt.Printf("git-savepoint started in background (pid %d), logging to %s\n", proc.Pid, logFile(repoRoot))
}

func writePID(repoRoot string, pid int) {
	_ = os.WriteFile(pidFile(repoRoot), []byte(strconv.Itoa(pid)), 0644)
}

func runningPID(repoRoot string) (int, bool) {
	data, err := os.ReadFile(pidFile(repoRoot))
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, false
	}
	if !isProcessRunning(pid) {
		return 0, false
	}
	return pid, true
}

func cmdStop(repoRoot string) {
	pid, running := runningPID(repoRoot)
	if !running {
		fmt.Println("git-savepoint is not running for this repo")
		return
	}
	proc, err := os.FindProcess(pid)
	fatalIf(err)
	fatalIf(terminateProcess(proc))
	os.Remove(pidFile(repoRoot))
	fmt.Println("stopped git-savepoint")
}

func runWatchLoop(repoRoot string) {
	cfg := watcher.DefaultConfig()
	w := watcher.New(repoRoot, cfg)
	w.OnCheckpoint = func(msg string, err error) {
		if err == checkpoint.ErrNoChanges {
			return
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "checkpoint failed:", err)
			return
		}
		fmt.Printf("[%s] checkpoint: %s\n", time.Now().Format("15:04:05"), msg)
	}

	stop := make(chan struct{})
	sig := make(chan os.Signal, 1)
	registerStopSignals(sig)
	go func() {
		<-sig
		close(stop)
	}()

	w.Run(stop)
}

//status/timeline

func cmdStatus(repoRoot string) {
	if pid, running := runningPID(repoRoot); running {
		fmt.Printf("watching:      yes (pid %d)\n", pid)
	} else {
		fmt.Println("watching:      no (run `git-savepoint start`)")
	}

	all, err := storage.List(repoRoot)
	fatalIf(err)
	fmt.Printf("checkpoints:   %d\n", len(all))
	if len(all) > 0 {
		last := all[len(all)-1]
		fmt.Printf("last checkpoint: %s (%s ago) - %s\n",
			last.Time().Format("15:04:05"), time.Since(last.Time()).Round(time.Second), last.Message)
	}
}

func cmdTimeline(repoRoot string) {
	all, err := storage.List(repoRoot)
	fatalIf(err)
	if len(all) == 0 {
		fmt.Println("No checkpoints yet. Run `git-savepoint start` to begin.")
		return
	}

	var currentDay string
	for _, cp := range all {
		day := cp.Time().Format("Monday, Jan 2")
		if day != currentDay {
			fmt.Println()
			fmt.Println(day)
			currentDay = day
		}
		fmt.Printf("  %s   %-40s [%s]\n", cp.Time().Format("15:04"), cp.Message, cp.Commit[:8])
	}
	fmt.Println()
}

// restore

func cmdRestore(repoRoot string, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: git-savepoint restore <checkpoint-id|latest>")
		os.Exit(1)
	}
	id := args[0]

	target, err := storage.Find(repoRoot, id)
	fatalIf(err)

	fmt.Printf("This will restore your working tree to the checkpoint at %s:\n  %s\n",
		target.Time().Format("15:04:05"), target.Message)
	fmt.Print("A safety checkpoint of your current state will be taken first. Continue? [y/N] ")

	var answer string
	fmt.Scanln(&answer)
	if strings.ToLower(strings.TrimSpace(answer)) != "y" {
		fmt.Println("aborted")
		return
	}

	result, err := restore.To(repoRoot, id)
	fatalIf(err)

	if result.SafetyCheckpoint != nil {
		fmt.Printf("safety checkpoint saved: %s\n", result.SafetyCheckpoint.Time().Format("15:04:05"))
	}
	fmt.Printf("restored to checkpoint: %s - %s\n",
		result.RestoredTo.Time().Format("15:04:05"), result.RestoredTo.Message)
}
