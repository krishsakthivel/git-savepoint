#!/usr/bin/env bash
set -euo pipefail

REPO="krishsakthivel/git-savepoint"

os="$(uname -s)"
arch="$(uname -m)"

case "$os" in
  Linux) goos="linux" ;;
  Darwin) goos="darwin" ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac

case "$arch" in
  x86_64|amd64) goarch="amd64" ;;
  arm64|aarch64) goarch="arm64" ;;
  *) echo "unsupported architecture: $arch" >&2; exit 1 ;;
esac

asset="git-savepoint-${goos}-${goarch}"
url="https://github.com/${REPO}/releases/latest/download/${asset}"

tmp="$(mktemp)"
cleanup() { rm -f "$tmp"; }
trap cleanup EXIT

echo "downloading ${asset}..."
curl -fsSL "$url" -o "$tmp"
chmod +x "$tmp"

echo "installing..."
"$tmp" install

echo
echo "done. open a new terminal and run: git-savepoint"