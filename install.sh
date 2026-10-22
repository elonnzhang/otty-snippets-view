#!/bin/sh
# Install or update otty-snippets-view from a GitHub release, then register it
# as Otty's details-panel view. Run it again later to update.
#
#   curl -fsSL https://raw.githubusercontent.com/elonnzhang/otty-snippets-view/main/install.sh | sh
#   curl -fsSL .../install.sh | sh -s -- --version v1.1.0
#   sh install.sh --check                    # only report installed vs latest
#   sh install.sh --from ./otty-snippets-view  # install a local build
#
# Variables:
#   FORCE=1             replace a details-view that belongs to something else
#                       (the only option that can't follow a pipe), e.g.
#                       curl -fsSL .../install.sh | FORCE=1 sh
#
# Options:
#   --dir <path>        where the binary goes (default ~/.config/otty/views/otty-snippets-view)
#   --version <tag>     release tag to install (default: the latest release)
#   --from <binary>     install this file instead of downloading a release
#   --check             print the installed and latest tags, then exit
#   --force             reinstall even when already on that tag
#   --no-config         only install the binary; leave Otty's config alone
#   --no-restart        don't stop a running view (it keeps the old build until reopened)
#   --config-file <p>   read/write this config instead of Otty's own (skips reload)
#   -h, --help          show this text
#
# The script never turns on ipc-allow-send-keys: that switch is the user's call,
# so it only prints a reminder when the switch is off.
set -eu

REPO=elonnzhang/otty-snippets-view
ASSET=otty-snippets-view.zip
BIN=otty-snippets-view

# Replace a details-view that belongs to something else. This is the one
# option that can't be passed after a pipe, where its arguments belong to sh.
FORCE=${FORCE:-0}

dir="${OTTY_SNIPPETS_DIR:-$HOME/.config/otty/views/otty-snippets-view}"
tag=""
from=""
do_config=1
restart=1
config_file=""
force=$FORCE
check=0

say() { printf '%s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

usage() {
  cat <<'TEXT'
Install or update otty-snippets-view from a GitHub release, then register it as
Otty's details-panel view.

  curl -fsSL https://raw.githubusercontent.com/elonnzhang/otty-snippets-view/main/install.sh | sh
  curl -fsSL .../install.sh | sh -s -- --version v1.1.0
  sh install.sh --check
  sh install.sh --from ./otty-snippets-view

  --dir <path>        where the binary goes (default ~/.config/otty/views/otty-snippets-view)
  --version <tag>     release tag to install (default: the latest release)
  --from <binary>     install this file instead of downloading a release
  --check             print the installed and latest tags, then exit
  --force             reinstall even when already on that tag
  --no-config         only install the binary; leave Otty's config alone
  --no-restart        don't stop a running view
  --config-file <p>   read/write this config instead of Otty's own (skips reload)
  -h, --help          show this text
TEXT
}

while [ $# -gt 0 ]; do
  case "$1" in
    --dir) [ $# -ge 2 ] || die "--dir needs a path"; dir=$2; shift 2 ;;
    --version) [ $# -ge 2 ] || die "--version needs a tag"; tag=$2; shift 2 ;;
    --from) [ $# -ge 2 ] || die "--from needs a binary"; from=$2; shift 2 ;;
    --config-file) [ $# -ge 2 ] || die "--config-file needs a path"; config_file=$2; shift 2 ;;
    --check) check=1; shift ;;
    --no-config) do_config=0; shift ;;
    --no-restart) restart=0; shift ;;
    --force) force=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) die "unknown option: $1" ;;
  esac
done

[ -n "$dir" ] || die "install directory is empty"
case "$dir" in /*) ;; *) die "--dir must be absolute: $dir" ;; esac
bin="$dir/$BIN"
installed=$(cat "$dir/.version" 2>/dev/null || echo "")

if [ "$do_config" = 1 ]; then
  command -v otty >/dev/null 2>&1 || die "otty is not on PATH; re-run with --no-config to install the binary only"
fi
command -v curl >/dev/null 2>&1 || die "curl is required"

# The redirect names the latest tag; it needs no API token and has no rate limit.
latest() {
  curl -fsSI -o /dev/null -w '%{redirect_url}' "https://github.com/$REPO/releases/latest" 2>/dev/null \
    | sed 's#.*/tag/##; s#/$##'
}

if [ "$check" = 1 ]; then
  want="${tag:-local}"
  [ -n "$from" ] || want=$(latest) || true
  [ -n "$want" ] || die "could not reach GitHub to find the latest release"
  printf 'installed: %s\nlatest:    %s\n' "${installed:-unknown}" "$want"
  if [ "$installed" = "$want" ]; then
    say "up to date"
  else
    say "update available: sh install.sh"
  fi
  exit 0
fi

# ---- fetch --------------------------------------------------------------------

if [ -z "$from" ]; then
  [ -n "$tag" ] || tag=$(latest) || true
  [ -n "$tag" ] || die "could not reach GitHub to find the latest release; pass --version <tag>"
fi

if [ "$force" != 1 ] && [ -z "$from" ] && [ -n "$installed" ] && [ "$installed" = "$tag" ]; then
  say "already on $tag"
  exit 0
fi

tmp=$(mktemp -d "${TMPDIR:-/tmp}/otty-snippets.XXXXXX")
trap 'rm -rf "$tmp"' EXIT INT TERM

src_bin=""
if [ -n "$from" ]; then
  [ -f "$from" ] || die "no such file: $from"
  src_bin=$from
  [ -n "$tag" ] || tag=local
else
  url="https://github.com/$REPO/releases/download/$tag/$ASSET"
  say "downloading $url"
  if ! curl -fsSL --retry 2 -o "$tmp/$ASSET" "$url"; then
    die "download failed: $url"
  fi
  mkdir -p "$tmp/unpacked"
  if ! unzip -q "$tmp/$ASSET" -d "$tmp/unpacked"; then
    die "could not unzip $ASSET"
  fi
  # The archive has held the binary at ./otty-snippets-view and at
  # ./dist/otty-snippets-view; find it by name so either layout works.
  src_bin=$(find "$tmp/unpacked" -type f -name "$BIN" | head -n 1)
  [ -n "$src_bin" ] || die "$ASSET does not contain a $BIN binary"
fi

if command -v file >/dev/null 2>&1; then
  file "$src_bin" | grep -q 'Mach-O' || die "not a macOS binary: $src_bin"
fi

# ---- install ------------------------------------------------------------------

mkdir -p "$dir"
if [ -x "$bin" ]; then
  cp -p "$bin" "$bin.bak"
  say "kept the previous build as $bin.bak"
fi
cp "$src_bin" "$bin.new"
chmod 755 "$bin.new"
mv "$bin.new" "$bin"
# Downloaded archives carry the quarantine flag; this binary has no signature to
# clear it, so drop it here instead of failing on first launch.
xattr -d com.apple.quarantine "$bin" 2>/dev/null || true
printf '%s\n' "$tag" > "$dir/.version"
say "installed $bin ($tag)"

# ---- configure Otty -----------------------------------------------------------

if [ "$do_config" = 1 ]; then
  otty_config() {
    if [ -n "$config_file" ]; then
      otty --config-file "$config_file" config "$@"
    else
      otty config "$@"
    fi
  }

  want=""
  if [ -n "$config_file" ] && [ ! -e "$config_file" ]; then
    : # a missing override file is an empty config
  else
    want=$(otty_config get details-view 2>/dev/null) || die "could not read details-view from the Otty config"
  fi

  lang=""
  if [ -n "${want:-}" ]; then
    lang=$(printf '%s\n' "$want" | sed -n 's/.*--lang[= ]\([A-Za-z_-]*\).*/\1/p')
    case "$lang" in
      zh|cn|zh-cn|zh_cn|chinese|中文) lang=cn ;;
      en|en-us) lang=en ;;
      *) [ -z "$lang" ] || lang= ;;
    esac
  fi

  cmd="$bin"
  [ -n "$lang" ] && cmd="$bin --lang $lang"

  if [ -z "${want:-}" ]; then
    want="id=\"snippets\" kind=\"tui\" title=\"Snippets\" icon=\"file\" command=\"$cmd\" mobile=\"false\""
    say "adding details-view"
  else
    # Never take over a details panel that belongs to another tool.
    case "$want" in
      *otty-snippets-view*|*'id="snippets"'*) ;;
      *)
        [ "$force" = 1 ] || die "details-view belongs to something else; set FORCE=1 to replace it (the piped form cannot take --force):
  $want"
        ;;
    esac
    case "$want" in
      *command=*) want=$(printf '%s\n' "$want" | sed "s#command=\"[^\"]*\"#command=\"$cmd\"#") ;;
      *)
        warn "details-view has no command= field; replacing it"
        want="id=\"snippets\" kind=\"tui\" title=\"Snippets\" icon=\"file\" command=\"$cmd\" mobile=\"false\""
        ;;
    esac
    say "updating details-view"
  fi
  otty_config set details-view "$want" >/dev/null
  say "details-view = $want"

  if [ -z "$config_file" ]; then
    otty config reload >/dev/null 2>&1 || warn "could not ask Otty to reload its config"
  fi

  send_keys=$(otty_config get ipc-allow-send-keys 2>/dev/null || true)
  if [ "$send_keys" != "true" ]; then
    say ""
    say "note: ipc-allow-send-keys is off, so run/fill will answer \"rejected\"."
    say "      Turning it on is your decision: set ipc-allow-send-keys = true in the Otty config."
  fi
fi

# ---- stop the running view ----------------------------------------------------
#
# Otty keeps a running view as it is when the config is reloaded, so the old
# build would stay on screen until the panel is reopened anyway.

stopped=""
if [ "$restart" = 1 ] && [ -x "$bin" ]; then
  pids=$(ps -axo pid=,command= | awk -v b="$bin" '$2 == b {print $1}')
  if [ -n "$pids" ]; then
    # shellcheck disable=SC2086
    kill $pids 2>/dev/null || true
    i=0
    while [ $i -lt 20 ]; do
      pids=$(ps -axo pid=,command= | awk -v b="$bin" '$2 == b {print $1}')
      [ -z "$pids" ] && break
      sleep 0.1
      i=$((i + 1))
    done
    # shellcheck disable=SC2086
    [ -z "$pids" ] || kill -9 $pids 2>/dev/null || true
    stopped="The old view was stopped. "
  fi
fi

say ""
say "${stopped}Open the details panel with ⌘⇧R (press it twice: once to close, once to open)."
