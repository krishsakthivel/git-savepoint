//go:build !windows

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// puts a copy of itself in ~/.local/bin and, if thats not already on
// PATH, adds it to whatever shell rc file it can find
func installToPath() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current exe: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("finding home dir: %w", err)
	}
	installDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("creating install dir: %w", err)
	}
	target := filepath.Join(installDir, "git-savepoint")

	if filepath.Clean(exe) != filepath.Clean(target) {
		if err := copyFile(exe, target); err != nil {
			return fmt.Errorf("copying binary: %w", err)
		}
		os.Chmod(target, 0755)
		fmt.Printf("copied to %s\n", target)
	}

	if onPath(installDir) {
		fmt.Println("~/.local/bin is already on your PATH, you're good: git-savepoint")
		return nil
	}

	rc := shellRCFile(home)
	line := `export PATH="$HOME/.local/bin:$PATH"`
	if rc == "" {
		fmt.Println("couldn't figure out your shell config, add this line to it yourself:")
		fmt.Println("  " + line)
		return nil
	}

	already, _ := fileContains(rc, line)
	if already {
		fmt.Println("PATH line already in", rc+", just open a new terminal")
		return nil
	}

	f, err := os.OpenFile(rc, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("couldn't write to", rc+", add this line yourself:")
		fmt.Println("  " + line)
		return nil
	}
	defer f.Close()
	fmt.Fprintf(f, "\n# added by git-savepoint\n%s\n", line)

	fmt.Printf("added ~/.local/bin to PATH in %s\n", rc)
	fmt.Println("run `source " + rc + "` or open a new terminal, then just run: git-savepoint")
	return nil
}

func onPath(dir string) bool {
	for _, p := range strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)) {
		if filepath.Clean(p) == filepath.Clean(dir) {
			return true
		}
	}
	return false
}

func shellRCFile(home string) string {
	shell := os.Getenv("SHELL")
	switch {
	case strings.Contains(shell, "zsh"):
		return filepath.Join(home, ".zshrc")
	case strings.Contains(shell, "bash"):
		if _, err := os.Stat(filepath.Join(home, ".bash_profile")); err == nil {
			return filepath.Join(home, ".bash_profile")
		}
		return filepath.Join(home, ".bashrc")
	default:
		return ""
	}
}

func fileContains(path, substr string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return strings.Contains(string(data), substr), nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
