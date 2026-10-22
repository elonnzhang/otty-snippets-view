#!/bin/sh
# One-shot build and install for development: vet, test, build a universal
# binary into dist/, then hand it to install.sh so the details-panel view is
# registered and the running copy is replaced.
#
#   ./release.sh            # vet + test + build + install to the Otty view
#   ./release.sh --fast     # skip vet/test
#   ./release.sh --tag          # also build dist/otty-snippets-view.zip for a release
#
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
    -h|--help) sed -n '2,10p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
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

step "build"
mkdir -p dist
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/arm64 .
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/amd64 .
lipo -create dist/arm64 dist/amd64 -output dist/otty-snippets-view
rm -f dist/arm64 dist/amd64
file dist/otty-snippets-view | sed 's/^/  /'

if [ "$zip" = 1 ]; then
  step "zip"
  rm -f dist/otty-snippets-view.zip
  ditto -c -k --keepParent dist/otty-snippets-view dist/otty-snippets-view.zip
  ls -lh dist/otty-snippets-view.zip | sed 's/^/  /'
fi

step "install"
label=$(git rev-parse --short HEAD 2>/dev/null || echo dev)
sh ./install.sh --from dist/otty-snippets-view --version "dev-$label" --force

say ""
say "Built and installed dev-$label."
