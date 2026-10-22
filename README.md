# otty-snippets-view

[中文](README_zh.md)

A view for Otty's details panel. It manages Otty's text-snippet recipes (`~/.config/otty/recipes/*.ottyrecipe`) and their groups, and sends a snippet to the focused terminal pane.

## Install

One line installs the latest release and registers it as the details-panel view. Running the same line again updates to the newest release:

```sh
curl -fsSL https://raw.githubusercontent.com/elonnzhang/otty-snippets-view/main/install.sh | sh
```

The script puts the binary in `~/.config/otty/views/otty-snippets-view/`, writes `details-view` into `~/.config/otty/config.toml` (keeping an existing `--lang`), reloads the config and stops any running copy, so the next ⌘⇧R opens the new build. It keeps the previous binary as `.bak`, records the installed tag in `.version`, and skips the work when that tag is already the latest. It never turns on `ipc-allow-send-keys` — it only reminds you when that switch is off. `details-view` is a list, one view per line; the script rewrites only the snippets entry (or adds it at the end) and writes the other views back as they were.

| Option | Meaning |
| --- | --- |
| `--check` | print installed and latest tags, then stop |
| `--dir <path>` | install directory (default `~/.config/otty/views/otty-snippets-view`) |
| `--version <tag>` | a specific release instead of the latest |
| `--from <binary>` | install a local build instead of downloading; `--from dev` builds the checkout |
| `--force` | reinstall the same tag (`FORCE=1 sh` before the installer when piping) |
| `--no-config` | only place the binary; leave Otty's config alone |
| `--no-restart` | leave a running view alone (it keeps the old build until reopened) |
| `--config-file <p>` | read and write this config instead of Otty's own |

Options after a pipe go through `sh -s --`, e.g. `curl -fsSL …/install.sh | sh -s -- --check`.

It reads the latest tag from the `releases/latest` redirect, so it needs neither `gh` nor a GitHub token and does not hit the API rate limit.

With Go installed, `go install` works too. It builds for this machine only (no universal binary) and does not touch Otty's config:

```sh
go install github.com/elonnzhang/otty-snippets-view@latest
```

The binary lands in `$(go env GOPATH)/bin/otty-snippets-view`; `otty-snippets-view --version` prints what it was built from. Register it with `init`:

```sh
otty-snippets-view init            # add or update the snippets entry, then reload Otty
otty-snippets-view init --reset    # remove it again
```

`init` points the entry at the binary it is run as and keeps that entry's other fields and `--lang`. Like the script, it changes only the snippets entry (`id="snippets"`, or a command that runs `otty-snippets-view`) in the `details-view` list; the other views stay. Then reopen the panel (⌘⇧R twice).

| Flag | Meaning |
| --- | --- |
| `--lang en\|cn` | language for the view's command (default: keep the current one, else English) |
| `--print` | print the `details-view` list it would write, one view per line, and stop |
| `--reset` | remove the snippets entry instead of writing it |
| `--config-file <p>` | edit this config instead of Otty's own (no reload) |
| `--no-reload` | don't ask a running Otty to reload its config |

From a checkout, `./release.sh` is the development loop: vet, test, then build and install the working tree (`sh install.sh --from dev`); `--fast` skips vet/test and `--zip` also builds the release zip. `sh install.sh --from dev` on its own is the same loop without the checks, and works with the panel open — the installed command changes on every run, which is what makes Otty start the new build. The Herdr plugin needs nothing extra — its `run.sh` rebuilds the binary it uses whenever a `.go` file is newer.

`details-view` must be quoted `key="value"` pairs; JSON is rejected. Otty does not restart a running view when the config is reloaded, which is why the script rewrites the command and stops the old process: with the panel open it switches to the new build, otherwise reopen the panel (⌘⇧R twice).

Sending a snippet to a pane requires `ipc-allow-send-keys = true` in the config. Without it, run/fill fails with "rejected (check ipc-allow-send-keys)".

## Options

| Option | Meaning |
| --- | --- |
| `--lang en\|cn` | UI language, English by default. Also `--lang=cn`; `zh`, `zh-CN` and `中文` work as well. |
| `--version` | print the build version and exit (also works for a `go install` build). |
| `SNIPPETS_LANG` | Same as `--lang`; the flag wins. |
| `OTTY_RECIPES_DIR` | Recipe directory, default `~/.config/otty/recipes`. |
| `SNIPPETS_HOME` | Where `groups.json` and `trash/` live, default `~/.config/otty/views/otty-snippets-view`. |

For a Chinese UI, use `command=".../otty-snippets-view --lang cn"`.

## Layout

From top to bottom:

1. Tab bar: **List** (flat, each row tagged with its group) and **Groups** (tree by group, foldable, for managing groups). Tab, `1`/`2` or a mouse click switches tabs.
2. Snippet list.
3. Detail area: name, alias, group and the full text with syntax highlighting. The language is guessed, falling back to shell. Drag the top border (`── detail ═══ ──`) with the mouse to resize it, or press `+` / `-`.
4. Key hints, in two rows: each key with its tip directly below. Status messages get the line just above the hints, so the keys always stay visible.

The selected tab and the detail height are saved in `groups.json`.

## Keys

| Key | Action |
| --- | --- |
| `⏎` / double-click | Run in the focused pane (on a group: fold) |
| `o` | Fill in without running |
| right-click | Context menu: on a snippet fill in / edit / move / delete / new; on a group new sub-group / rename / move / delete; on empty space new group / snippet |
| `y` | Copy to the clipboard |
| `a` | New snippet (in the Groups tab it goes into the current group) |
| `e` | Edit snippet; on a group, rename it |
| `m` | Move to a group (menu); on a group header, move the group under another |
| `M` | New sub-group under the selected group |
| `c` | Color of the selected snippet or group (`1`–`6` pick, `0` clears); also in the right-click menu |
| drag | Drag a snippet onto a group (Groups tab) |
| `d` / `x` / `Delete` | Delete, after a confirmation box (`y` deletes, any other key cancels); the file moves to `trash/` |
| `n` / `r` | New group / rename group |
| `[` / `]` | Reorder groups |
| `/` | Search; `Esc` clears it |
| `R` / `⌃R` | Reload from disk |
| `↑↓` `jk` `PgUp/PgDn` `g/G` `←→` | Navigate; left/right also fold and unfold |
| click empty space | clear the selection (the detail area empties) |
| click a hint | run the key it names |

From a cleared selection, `↓` selects the first row and `↑` the last one.

New and rename group, new and edit snippet, and the delete confirmation open a box in the middle of the screen; the list stays visible around it. A click outside the box cancels, and a name that is rejected keeps the box open with the reason inside it.

Form: `Tab` / `⇧Tab` switches fields, `⌃S` saves, `Esc` cancels. Alias is required and cannot contain spaces. A long body grows the box to at most two thirds of the screen and scrolls inside it beyond that (`↑`/`↓` mark what is out of view, the wheel scrolls it), and typing keeps the caret in view. Click anywhere in a field — name, alias or the body — to put the caret on that character and keep typing there.

Placeholders in a snippet's text: `{{clipboard}}` `{{date}}` `{{time}}`. `{{cursor}}` only works in Otty's own expander and is removed here.

## Groups

- A name like `[group]-name` puts the snippet into group `group` automatically. The group is created if it doesn't exist; matching ignores case. The list shows the snippet as `name` inside its group.
- Moving a snippet by hand overrides its prefix. Moving it back to its prefix group drops the override.
- Deleting a group moves its members to Ungrouped and does not recreate the group from their prefixes.
- Renaming a group keeps its prefix-matched members.
- Ungrouped is only a display label. You can't create a group with that name in either language.

### Nested groups

- Groups form a tree. Press `M` on a group to create a sub-group under it, or `m` to move it under another group (or back to the top level). In the Groups tab you can also drag one group header onto another.
- A group can't be moved under itself or one of its own sub-groups.
- Folding a group hides its sub-groups and their snippets too.
- `[` / `]` reorder a group among its siblings only.
- Deleting a group deletes its sub-groups as well; all their snippets move to Ungrouped.

## Herdr plugin

The same binary also runs as a Herdr plugin pane, showing the snippets beside the pane you are working in. Install the plugin once (needs Go: the manifest builds the binary during install):

```sh
herdr plugin install elonnzhang/otty-snippets-view/herdr-plugin
```

and bind a key to its action in `~/.config/herdr/config.toml`:

```toml
[[keys.command]]
key = "prefix+shift+s"
type = "plugin_action"
command = "local.otty-snippets.open"
description = "snippets"
```

`⏎` types the snippet into the pane the sidebar was opened from (Herdr's invocation context, not the sidebar itself), `o` only fills it in without pressing Enter. `SNIPPETS_TARGET_PANE` overrides the target. Snippets and groups are the same files the Otty view uses, so both hosts stay in sync.

Herdr has no widget slot in its own sidebar, so the plugin opens a split pane — the closest placement it offers. See [herdr-plugin/README.md](herdr-plugin/README.md) for details.

## Files

- `~/.config/otty/recipes/*.ottyrecipe`: the snippets themselves, the same files Otty's Recipes settings edit. Writes are atomic and keep Otty's format. A file with fields this tool doesn't know is never rewritten; edit it in Otty instead.
- `groups.json`: groups, manual assignments (`"none"` pins a snippet to Ungrouped), item colors and UI state. A corrupt file is renamed to `groups.json.bad`.
- `trash/`: deleted recipes, named `<timestamp>-<id>.ottyrecipe`.

Changes made in Otty's Recipes settings are picked up within a second.

## Development

```sh
go vet ./... && go test ./...
go build -o otty-snippets-view .
```

A universal binary for sharing (arm64 + x86_64 on macOS):

```sh
mkdir -p dist
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/arm64 .
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/amd64 .
lipo -create dist/arm64 dist/amd64 -output dist/otty-snippets-view
rm dist/arm64 dist/amd64
ditto -c -k --keepParent dist/otty-snippets-view dist/otty-snippets-view.zip
```

`-s -w` drops the symbol table and DWARF data (21 MB → 16 MB); the zip is about 5.4 MB.

Most of the size is the syntax highlighter: chroma's `lexers` package embeds all 296 lexer definitions (about 5 MB on its own), and any single import of it pulls them all in. Shipping only a handful of lexers would need our own small `embed.FS` and registry, which costs auto-detection, so the full set stays.

`internal/noquery/` must stay imported by `main.go`. Bubble Tea's `init` asks the terminal for its background color and waits up to 5s for an answer; a terminal that never answers leaves the panel blank that long. `noquery` answers first. Import order can't guarantee which one runs first now that the module path is `github.com/…/otty-snippets-view`, so treat it as a race in our favor: the renderer keeps whichever answer it gets first.

`main.go` only starts the program; the view is `internal/app/`. Code map: `store.go` (recipe and metadata files), `meta.go` (grouping rules, talking to Otty), `model.go` (state, rows, layout), `update.go` (keys, mouse), `actions.go`, `view.go` / `detail.go` (rendering), `highlight.go`, `form.go`, `i18n.go` (all UI strings). `dist/` holds shareable builds, not sources.

## Share it

Send a colleague the install line from [Install](#install). It pulls the universal (arm64 + x86_64) build from the [latest release](https://github.com/elonnzhang/otty-snippets-view/releases/latest) and writes the config line themselves; the same line updates them later. Snippets are not bundled: each person's own `~/.config/otty/recipes` is listed, and a fresh `groups.json` is created in `~/.config/otty/views/otty-snippets-view`. Sending to a pane needs `ipc-allow-send-keys = true`, which the script leaves to them.

To install by hand instead, download `otty-snippets-view.zip` from that release, unzip it, and add:

```toml
details-view = 'id="snippets" kind="tui" title="Snippets" icon="file" command="/path/to/otty-snippets-view" mobile="false"'
```

An agent can do the install from the prompt in [SETUP-PROMPT.md](SETUP-PROMPT.md). If Gatekeeper blocks the first launch, right-click → Open, or run `xattr -d com.apple.quarantine otty-snippets-view`.
