#!/bin/sh
# Herdr action: open the Snippets pane beside the active pane. Actions run in
# the background without a terminal, so the TUI must not run here; this only
# asks Herdr to open the pane entrypoint, which then runs run.sh.
set -eu
herdr="${HERDR_BIN_PATH:-herdr}"
exec "$herdr" plugin pane open --plugin local.otty-snippets --entrypoint snippets \
  --placement split --direction right --focus
