//go:build windows

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// puts a copy of itself in %LOCALAPPDATA%\git-savepoint and adds that
// folder to the user's PATH, so you can just type git-savepoint from
// any terminal afterwards instead of needing the .exe sitting right
// there or typing the full path
func installToPath() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current exe: %w", err)
	}

	installDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "git-savepoint")
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("creating install dir: %w", err)
	}
	target := filepath.Join(installDir, "git-savepoint.exe")

	// if we're already running from the install location theres nothing
	// to copy (also avoids trying to overwrite ourselves mid-run)
	if filepath.Clean(exe) != filepath.Clean(target) {
		if err := copyFile(exe, target); err != nil {
			return fmt.Errorf("copying exe: %w", err)
		}
		fmt.Printf("copied to %s\n", target)
	}

	// using [Environment]::SetEnvironmentVariable here instead of setx -
	// setx has a ~1024 char limit and will silently truncate a long PATH,
	// this way reads/writes the registry properly
	script := fmt.Sprintf(`
$dir = %q
$current = [Environment]::GetEnvironmentVariable('Path','User')
if ($current -split ';' -notcontains $dir) {
    $new = if ($current) { $current.TrimEnd(';') + ';' + $dir } else { $dir }
    [Environment]::SetEnvironmentVariable('Path', $new, 'User')
    Write-Output "added $dir to your PATH"
} else {
    Write-Output "$dir is already on your PATH"
}
`, installDir)

	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("updating PATH: %w: %s", err, out)
	}
	fmt.Print(string(out))

	fmt.Println()
	fmt.Println("open a new terminal window for this to take effect, then just run: git-savepoint")
	return nil
}

// removes the installed copy and takes installDir back off the user PATH
func uninstallFromPath() error {
	installDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "git-savepoint")
	target := filepath.Join(installDir, "git-savepoint.exe")

	script := fmt.Sprintf(`
$dir = %q
$current = [Environment]::GetEnvironmentVariable('Path','User')
if ($current) {
    $parts = $current -split ';' | Where-Object { $_ -and $_ -ne $dir }
    [Environment]::SetEnvironmentVariable('Path', ($parts -join ';'), 'User')
    Write-Output "removed $dir from your PATH"
} else {
    Write-Output "PATH was empty, nothing to remove"
}
`, installDir)

	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("updating PATH: %w: %s", err, out)
	}
	fmt.Print(string(out))

	// windows locks a running exe, so if this IS the installed copy
	// currently executing, it cant delete itself - tell the person
	// instead of failing weirdly
	exe, err := os.Executable()
	if err == nil && filepath.Clean(exe) == filepath.Clean(target) {
		fmt.Println("this is the installed copy currently running, so it can't delete its own file yet.")
		fmt.Printf("you can manually delete this folder afterwards: %s\n", installDir)
		return nil
	}

	if err := os.RemoveAll(installDir); err != nil {
		return fmt.Errorf("removing %s: %w", installDir, err)
	}
	fmt.Printf("removed %s\n", installDir)
	return nil
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
