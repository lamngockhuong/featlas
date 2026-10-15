#!/bin/sh
# Install the muninn binary from GitHub releases.
#   curl -fsSL https://raw.githubusercontent.com/lamngockhuong/muninn/main/install.sh | sh
# Environment:
#   MUNINN_VERSION      release tag to install, e.g. v0.1.0 (default: latest)
#   MUNINN_INSTALL_DIR  target directory (default: $HOME/.local/bin)
set -eu

repo="lamngockhuong/muninn"
version="${MUNINN_VERSION:-latest}"
dir="${MUNINN_INSTALL_DIR:-$HOME/.local/bin}"

fail() { echo "muninn install: $*" >&2; exit 1; }

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) fail "unsupported OS $(uname -s); on Windows download the .zip from https://github.com/$repo/releases" ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) fail "unsupported architecture $(uname -m)" ;;
esac

if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1" -o "$2"; }
  resolve() { curl -fsSLI -o /dev/null -w '%{url_effective}' "$1"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO "$2" "$1"; }
  resolve() { wget -q --max-redirect=5 -S --spider "$1" 2>&1 | awk '/^  Location: /{u=$2} END{print u}'; }
else
  fail "curl or wget is required"
fi

if [ "$version" = latest ]; then
  version="$(resolve "https://github.com/$repo/releases/latest" | sed 's|.*/tag/||')"
  [ -n "$version" ] || fail "cannot find the latest release"
fi
number="${version#v}"
archive="muninn_${number}_${os}_${arch}.tar.gz"
base="https://github.com/$repo/releases/download/$version"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM
echo "downloading muninn $version for $os/$arch"
fetch "$base/$archive" "$tmp/$archive" || fail "download failed: $base/$archive"
fetch "$base/checksums.txt" "$tmp/checksums.txt" || fail "download failed: $base/checksums.txt"

expected="$(grep " $archive\$" "$tmp/checksums.txt" | cut -d' ' -f1)"
[ -n "$expected" ] || fail "$archive is not listed in checksums.txt"
if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$tmp/$archive" | cut -d' ' -f1)"
else
  actual="$(shasum -a 256 "$tmp/$archive" | cut -d' ' -f1)"
fi
[ "$expected" = "$actual" ] || fail "checksum mismatch for $archive"

tar -xzf "$tmp/$archive" -C "$tmp" muninn
mkdir -p "$dir"
mv "$tmp/muninn" "$dir/muninn"
chmod +x "$dir/muninn"
echo "installed $("$dir/muninn" version) to $dir/muninn"
case ":$PATH:" in
  *":$dir:"*) ;;
  *) echo "add $dir to your PATH to run muninn" ;;
esac
