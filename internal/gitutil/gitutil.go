package gitutil

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

func RunWithEnv(dir string, env []string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

func RepoRoot(startDir string) (string, error) {
	out, err := Run(startDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("not a git repository (or any parent up to mount point): %w", err)
	}
	return out, nil
}

func GitDir(repoRoot string) (string, error) {
	out, err := Run(repoRoot, "rev-parse", "--git-dir")
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(out) {
		return out, nil
	}
	return filepath.Join(repoRoot, out), nil
}

func HeadCommit(repoRoot string) string {
	out, err := Run(repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return ""
	}
	return out
}

func UnsupportedFeatures(repoRoot string) ([]string, error) {
	gitDir, err := GitDir(repoRoot)
	if err != nil {
		return nil, err
	}

	var found []string

	if fileExists(filepath.Join(gitDir, "MERGE_HEAD")) {
		found = append(found, "a merge in progress")
	}
	if fileExists(filepath.Join(gitDir, "rebase-merge")) || fileExists(filepath.Join(gitDir, "rebase-apply")) {
		found = append(found, "a rebase in progress")
	}
	if fileExists(filepath.Join(repoRoot, ".gitmodules")) {
		found = append(found, "submodules")
	}
	if isSparseCheckout(repoRoot, gitDir) {
		found = append(found, "sparse checkout")
	}
	if isLinkedWorktree(repoRoot, gitDir) {
		found = append(found, "worktrees")
	}
	if usesGitLFS(repoRoot) {
		found = append(found, "Git LFS")
	}

	return found, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isSparseCheckout(repoRoot, gitDir string) bool {
	if out, err := Run(repoRoot, "config", "--bool", "core.sparseCheckout"); err == nil && out == "true" {
		return true
	}
	return fileExists(filepath.Join(gitDir, "info", "sparse-checkout"))
}

func isLinkedWorktree(repoRoot, gitDir string) bool {
	commonDir, err := Run(repoRoot, "rev-parse", "--git-common-dir")
	if err != nil {
		return false
	}
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Join(repoRoot, commonDir)
	}
	return filepath.Clean(commonDir) != filepath.Clean(gitDir)
}

func usesGitLFS(repoRoot string) bool {
	data, err := os.ReadFile(filepath.Join(repoRoot, ".gitattributes"))
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "filter=lfs")
}
