package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	updateRepo      = "krishsakthivel/git-savepoint"
	checkTimeout    = 30 * time.Second
	downloadTimeout = 5 * time.Minute
	maxAttempts     = 3
)

type ghAsset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

func cmdUpdate(args []string) {
	checkOnly := false
	for _, a := range args {
		if a == "--check" {
			checkOnly = true
		}
	}

	release, err := latestRelease()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		fmt.Fprintf(os.Stderr, "you can also grab it directly: https://github.com/%s/releases/latest\n", updateRepo)
		os.Exit(1)
	}

	if version != "dev" && release.TagName == version {
		fmt.Printf("already up to date (%s)\n", version)
		return
	}

	if version == "dev" {
		fmt.Printf("running a dev build, latest release is %s\n", release.TagName)
	} else {
		fmt.Printf("update available: %s -> %s\n", version, release.TagName)
	}

	if checkOnly {
		fmt.Println("run `git-savepoint update` to install it")
		return
	}

	assetName := assetNameForPlatform()
	var assetURL string
	for _, a := range release.Assets {
		if a.Name == assetName {
			assetURL = a.DownloadURL
			break
		}
	}
	if assetURL == "" {
		fmt.Fprintf(os.Stderr, "error: no build for %s/%s found in release %s\n", runtime.GOOS, runtime.GOARCH, release.TagName)
		os.Exit(1)
	}

	fmt.Printf("continue updating to %s? [y/N] ", release.TagName)
	var answer string
	fmt.Scanln(&answer)
	if strings.ToLower(strings.TrimSpace(answer)) != "y" {
		fmt.Println("aborted")
		return
	}

	fmt.Println("downloading", assetName+"...")
	data, err := downloadAsset(assetURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		fmt.Fprintf(os.Stderr, "you can also download it directly: %s\n", assetURL)
		os.Exit(1)
	}

	fatalIf(applyUpdate(data))
	fmt.Printf("updated to %s\n", release.TagName)
}

func assetNameForPlatform() string {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("git-savepoint-%s-%s%s", runtime.GOOS, runtime.GOARCH, ext)
}

// httpGetWithRetry retries transient network failures (timeouts, resets,
// dropped connections) a few times with a short backoff before giving up.
// It does not retry on non-2xx HTTP responses, only on transport errors.
func httpGetWithRetry(client *http.Client, req *http.Request) (*http.Response, error) {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Do(req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if attempt < maxAttempts {
			wait := time.Duration(attempt) * 3 * time.Second
			fmt.Printf("network hiccup, retrying in %s... (%v)\n", wait, err)
			time.Sleep(wait)
		}
	}
	return nil, fmt.Errorf("after %d attempts: %w", maxAttempts, lastErr)
}

func latestRelease() (*ghRelease, error) {
	client := &http.Client{Timeout: checkTimeout}
	req, err := http.NewRequest("GET", "https://api.github.com/repos/"+updateRepo+"/releases/latest", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "git-savepoint-updater")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := httpGetWithRetry(client, req)
	if err != nil {
		return nil, fmt.Errorf("checking for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("checking for updates: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("reading release info: %w", err)
	}
	return &release, nil
}

func downloadAsset(url string) ([]byte, error) {
	client := &http.Client{Timeout: downloadTimeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "git-savepoint-updater")

	resp, err := httpGetWithRetry(client, req)
	if err != nil {
		return nil, fmt.Errorf("downloading update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading update: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// applyUpdate writes the new binary next to the currently running one,
// then swaps it into place. The old binary is moved aside rather than
// deleted directly, since a running executable can't always be removed
// on every platform (Windows in particular).
func applyUpdate(newBinary []byte) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current exe: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("resolving current exe: %w", err)
	}

	dir := filepath.Dir(exe)
	tmp := filepath.Join(dir, ".git-savepoint.update.tmp")
	old := filepath.Join(dir, ".git-savepoint.old")

	if err := os.WriteFile(tmp, newBinary, 0755); err != nil {
		return fmt.Errorf("writing new binary: %w", err)
	}

	os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("replacing current binary (try closing other running copies first): %w", err)
	}

	if err := os.Rename(tmp, exe); err != nil {
		// try to put the original back so the install isn't left broken
		os.Rename(old, exe)
		return fmt.Errorf("moving new binary into place: %w", err)
	}

	if err := os.Remove(old); err != nil {
		fmt.Printf("note: couldn't remove the old binary (%s), you can delete it yourself\n", old)
	}
	return nil
}
