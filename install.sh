#!/bin/sh
# Install byway from GitHub releases, then set up the shell integration.
set -eu
REPO=gabrielegenovesi/byway
os=$(uname -s | tr '[:upper:]' '[:lower:]')
case $(uname -m) in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "byway: unsupported architecture $(uname -m)" >&2; exit 1 ;;
esac
ver=${BYWAY_VERSION:-$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"v\{0,1\}\([^"]*\)".*/\1/p')}
dir=${BYWAY_BIN_DIR:-$HOME/.local/bin}
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
curl -fsSL "https://github.com/$REPO/releases/download/v$ver/byway_${ver}_${os}_${arch}.tar.gz" | tar -xz -C "$tmp"
mkdir -p "$dir"
install -m 0755 "$tmp/byway" "$dir/byway"
echo "installed byway $ver to $dir/byway"
case ":$PATH:" in *":$dir:"*) ;; *) echo "note: add $dir to your PATH" ;; esac
"$dir/byway" setup || true
