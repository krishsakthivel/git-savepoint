
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
		return "", fmt.Errorf("not a git repo (or any parent up to mount point): %w", err)
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