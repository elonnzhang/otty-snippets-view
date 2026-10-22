#!/bin/sh
# Development loop: vet, test, build the checkout and refresh the Otty
# details-panel view with it. Works with the panel open: the installed command
# changes on every run, which is what makes Otty start the new build.
#
#   ./release.sh            # vet + test + build + refresh the view
#   ./release.sh --fast     # skip vet/test
#   ./release.sh --tag      # also build dist/otty-snippets-view.zip for a release
#
# The steps are install.sh's: `sh install.sh --from dev` does the build and the
# view work. This script only adds the checks and the release zip.
# The Herdr plugin needs nothing extra: run.sh rebuilds its own copy from these
# sources whenever a .go file is newer than the binary it has.
set -eu
cd "$(dirname "$0")"

fast=0
zip=0
for a in "$@"; do
  case "$a" in
    --fast) fast=1 ;;
    --zip|--tag) zip=1 ;;
    -h|--help) sed -n '2,12p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) printf 'unknown option: %s\n' "$a" >&2; exit 2 ;;
  esac
done

say() { printf '%s\n' "$*"; }
step() { printf '\n== %s\n' "$*"; }

if [ "$fast" = 0 ]; then
  step "vet"
  go vet ./...
  step "test"
  go test ./...
fi

if [ "$zip" = 1 ]; then
  step "release zip"
  mkdir -p dist
  CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/arm64 .
  CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/amd64 .
  lipo -create dist/arm64 dist/amd64 -output dist/otty-snippets-view
  rm -f dist/arm64 dist/amd64
  rm -f dist/otty-snippets-view.zip
  ditto -c -k --keepParent dist/otty-snippets-view dist/otty-snippets-view.zip
  ls -lh dist/otty-snippets-view.zip | sed 's/^/  /'
fi

step "install"
sh ./install.sh --from dev
