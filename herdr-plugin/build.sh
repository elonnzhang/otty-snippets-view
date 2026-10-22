#!/bin/sh
# Builds the TUI from the checkout one level up. Runs during `herdr plugin
# install` (see [[build]] in herdr-plugin.toml) and from run.sh when the
# sources are newer than the binary.
set -eu
here="$(cd "$(dirname "$0")" && pwd)"
src="$(cd "$here/.." && pwd)"
bin="$here/.build/otty-snippets-view"

mkdir -p "$here/.build"

if ! command -v go >/dev/null 2>&1; then
  if [ -x "$bin" ]; then
    echo "go not found; keeping the existing binary"
    exit 0
  fi
  echo "otty-snippets-view: Go is not installed." >&2
  echo "Install Go (https://go.dev/dl/) or copy a prebuilt binary to $bin." >&2
  exit 1
fi

cd "$src"
go build -trimpath -ldflags="-s -w" -o "$bin.new" .
mv "$bin.new" "$bin"
