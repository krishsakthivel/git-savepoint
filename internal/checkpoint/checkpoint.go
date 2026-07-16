
package checkpoint

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"git-savepoint/internal/gitutil"
	"git-savepoint/internal/storage"
)


var DefaultIgnore = []string{
	"node_modules/",
	".git/",
	"dist/",
	"build/",
	".env",
	".env.*",
	"*.log",
}

func Create(repoRoot string, message string) (storage.Checkpoint, error) {
	if message == "" {
		message = fmt.Sprintf("checkpoint %s", time.Now().Format("15:04:05"))
	}

	gitDir, err := gitutil.GitDir(repoRoot)
	if err != nil {
		return storage.Checkpoint{}, err
	}

	scratchIndex := filepath.Join(gitDir, "git-savepoint-index.tmp")
	defer os.Remove(scratchIndex)

	env := []string{"GIT_INDEX_FILE=" + scratchIndex}


	head := gitutil.HeadCommit(repoRoot)
	if head != "" {
		if _, err := gitutil.RunWithEnv(repoRoot, env, "read-tree", head); err != nil {
			return storage.Checkpoint{}, fmt.Errorf("seeding scratch index: %w", err)
		}
	}

	addArgs := []string{"add", "--all"}
	addArgs = append(addArgs, "--", ".")
	for _, pattern := range DefaultIgnore {

		addArgs = append(addArgs, ":(exclude)"+pattern)
	}
	if _, err := gitutil.RunWithEnv(repoRoot, env, addArgs...); err != nil {
		return storage.Checkpoint{}, fmt.Errorf("staging working tree: %w", err)
	}

	treeHash, err := gitutil.RunWithEnv(repoRoot, env, "write-tree")
	if err != nil {
		return storage.Checkpoint{}, fmt.Errorf("writing tree: %w", err)
	}


	if last, ok, _ := storage.Latest(repoRoot); ok {
		lastTree, _ := gitutil.Run(repoRoot, "rev-parse", last.Commit+"^{tree}")
		if lastTree == treeHash {
			return storage.Checkpoint{}, ErrNoChanges
		}
	}

	commitArgs := []string{"commit-tree", treeHash, "-m", message}
	if last, ok, _ := storage.Latest(repoRoot); ok {
		commitArgs = append(commitArgs, "-p", last.Commit)
	} else if head != "" {
		commitArgs = append(commitArgs, "-p", head)
	}

	commitHash, err := gitutil.Run(repoRoot, commitArgs...)
	if err != nil {
		return storage.Checkpoint{}, fmt.Errorf("creating commit object: %w", err)
	}

	return storage.Save(repoRoot, commitHash, message)
}


var ErrNoChanges = fmt.Errorf("no changes since last checkpoint")