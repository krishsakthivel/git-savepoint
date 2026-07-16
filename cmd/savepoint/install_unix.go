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

// removes the installed copy and takes the PATH line back out of
// whichever shell rc file we added it to
func uninstallFromPath() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("finding home dir: %w", err)
	}
	installDir := filepath.Join(home, ".local", "bin")
	target := filepath.Join(installDir, "git-savepoint")

	rc := shellRCFile(home)
	if rc != "" {
		if err := removeInstallLines(rc); err != nil {
			fmt.Printf("couldn't clean up %s automatically: %v\n", rc, err)
			fmt.Println("you can remove the '# added by git-savepoint' block from it yourself")
		} else {
			fmt.Printf("removed the PATH line from %s (if it was there)\n", rc)
		}
	}

	exe, err := os.Executable()
	if err == nil && filepath.Clean(exe) == filepath.Clean(target) {
		fmt.Println("this is the installed copy currently running, it can't delete its own file while running.")
		fmt.Printf("run this after closing: rm %s\n", target)
		return nil
	}

	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing %s: %w", target, err)
	}
	fmt.Printf("removed %s\n", target)
	return nil
}

// strips the "# added by git-savepoint" comment and the PATH export
// line right after it, leaving everything else in the rc file alone
func removeInstallLines(rc string) error {
	data, err := os.ReadFile(rc)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	lines := strings.Split(string(data), "\n")
	var kept []string
	for i := 0; i < len(lines); i++ {
		if strings.Contains(lines[i], "# added by git-savepoint") {
			// also skip the export line right after it, if that's what's there
			if i+1 < len(lines) && strings.Contains(lines[i+1], `export PATH="$HOME/.local/bin:$PATH"`) {
				i++
			}
			continue
		}
		kept = append(kept, lines[i])
	}
	return os.WriteFile(rc, []byte(strings.Join(kept, "\n")), 0644)
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
