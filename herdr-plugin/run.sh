#!/bin/sh
# Herdr entrypoint for the Snippets sidebar. Herdr runs the command with the
# plugin's root as the working directory and passes this pane's ids in the
# environment, so the build lives next to the sources one level up.
set -eu
here="$(cd "$(dirname "$0")" && pwd)"
bin="$here/.build/otty-snippets-view"
src="$here/.."

if [ ! -x "$bin" ] || [ -n "$(find "$src" -maxdepth 1 -name '*.go' -newer "$bin" -print -quit)" ]; then
  mkdir -p "$here/.build"
  if ( cd "$src" && go build -trimpath -ldflags="-s -w" -o "$bin.new" . ); then
    mv "$bin.new" "$bin"
  else
    rm -f "$bin.new"
    [ -x "$bin" ] || { echo "otty-snippets-view: build failed (is Go installed?)" >&2; sleep 3; exit 1; }
  fi
fi

exec "$bin" "$@"
