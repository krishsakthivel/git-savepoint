//	git-savepoint start [--daemon]
//	git-savepoint status
//	git-savepoint timeline
//	git-savepoint restore <checkpoint-id>
//	git-savepoint stop
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

func main(){
	if len(os.Args) < 2{
		usage()
		os.Exit(1)
	}
	cwd, err := os.Getwd()
	fatalIf(err)
	repoRoot, err := gitutil.RepoRoot(cwd)
	fatalIf(err)

	switch os.Args[1]{
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
		case "-h", "--help", "help":
			usage()
		default:
			fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
			usage()
			os.Exit(1)
	}
}

// usage commands
func usage(){
	fmt.Print(`git-savepoint - a local-first time machine for your Git working tree
 
Usage:
  git-savepoint start [--daemon]   watch the repo and checkpoint automatically
  git-savepoint status             show whether the watcher is running and recent activity
  git-savepoint timeline           list checkpoints, newest last
  git-savepoint restore <id>       restore the working tree to a checkpoint (id, timestamp, or "latest")
  git-savepoint stop               stop the background watcher
  `)
}

func fatalIf(err error){
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
//83
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
		fmt.Println("(run with --daemon to background this; ctrl+C to stop)")
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