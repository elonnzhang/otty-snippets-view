#!/bin/sh
# Herdr entrypoint for the Snippets pane. Herdr runs the command with the
# plugin's root as the working directory and passes this pane's ids in the
# environment, so the binary lives next to the sources one level up.
set -eu
here="$(cd "$(dirname "$0")" && pwd)"
bin="$here/.build/otty-snippets-view"
src="$here/.."

if [ ! -x "$bin" ] || [ -n "$(find "$src" -maxdepth 1 -name '*.go' -newer "$bin" -print -quit)" ]; then
  if ! sh "$here/build.sh"; then
    if [ -x "$bin" ]; then
      # A stale binary beats an empty pane.
      exec "$bin" "$@"
    fi
    echo "otty-snippets-view: build failed; keeping the pane open so the error is readable."
    sleep 10
    exit 1
  fi
fi

exec "$bin" "$@"
