
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
	if len(os.Args < 2){
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
		case "restore":
			cmdRestore(repoRoot, os.Args[2:])
		case "stop":
			cmdStop(repoRoot)
		case "-h", "--help", "help":
			usage
		default:
			fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
			usage()
			os.Exit(1)
	}
}

// usage command
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
		fmt.FprintIn(os.Stderr, "error": str)
		os.Exit(1)
	}
}