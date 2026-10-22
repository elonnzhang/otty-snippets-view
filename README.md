# otty-snippets-view

[中文](README_zh.md)

A view for Otty's details panel. It manages Otty's text-snippet recipes (`~/.config/otty/recipes/*.ottyrecipe`) and their groups, and sends a snippet to the focused terminal pane.

## Install

Build:

```sh
cd ~/.config/otty/views/otty-snippets-view
go build -o otty-snippets-view .
```

Register the view in `~/.config/otty/config.toml`:

```toml
details-view = 'id="snippets" kind="tui" title="Snippets" icon="file" command="/Users/elon/.config/otty/views/otty-snippets-view/otty-snippets-view" mobile="false"'
```

The value must be quoted `key="value"` pairs; JSON is rejected. Reload with `otty config reload`, then reopen the details panel (⌘⇧R twice). Otty does not restart a running view on reload.

Sending a snippet to a pane requires `ipc-allow-send-keys = true` in the config. Without it, run/fill fails with "rejected (check ipc-allow-send-keys)".

The config points straight at the binary; rebuild and reload the config (or reopen the panel) after changing the Go code.

## Options

| Option | Meaning |
| --- | --- |
| `--lang en\|cn` | UI language, English by default. Also `--lang=cn`; `zh`, `zh-CN` and `中文` work as well. |
| `SNIPPETS_LANG` | Same as `--lang`; the flag wins. |
| `OTTY_RECIPES_DIR` | Recipe directory, default `~/.config/otty/recipes`. |
| `SNIPPETS_HOME` | Where `groups.json` and `trash/` live, default `~/.config/otty/views/otty-snippets-view`. |

For a Chinese UI, use `command=".../otty-snippets-view --lang cn"`.

## Layout

From top to bottom:

1. Tab bar: **List** (flat, each row tagged with its group) and **Groups** (tree by group, foldable, for managing groups). Tab, `1`/`2` or a mouse click switches tabs.
2. Snippet list.
3. Detail area: name, alias, group and the full text with syntax highlighting. The language is guessed, falling back to shell. Drag the top border (`── detail ═══ ──`) with the mouse to resize it, or press `+` / `-`.
4. Key hints, in two rows: each key with its tip directly below. Status messages and confirmations get the line just above the hints, so the keys always stay visible.

The selected tab and the detail height are saved in `groups.json`.

## Keys

| Key | Action |
| --- | --- |
| `⏎` / double-click | Run in the focused pane (on a group: fold) |
| `o` / right-click | Fill in without running |
| `y` | Copy to the clipboard |
| `a` | New snippet (in the Groups tab it goes into the current group) |
| `e` | Edit snippet; on a group, rename it |
| `m` | Move to a group (menu); on a group header, move the group under another |
| `M` | New sub-group under the selected group |
| drag | Drag a snippet onto a group (Groups tab) |
| `d` / `x` / `Delete` | Delete; the file moves to `trash/` |
| `n` / `r` | New group / rename group |
| `[` / `]` | Reorder groups |
| `/` | Search; `Esc` clears it |
| `R` / `⌃R` | Reload from disk |
| `↑↓` `jk` `PgUp/PgDn` `g/G` `←→` | Navigate; left/right also fold and unfold |

Form: `Tab` / `⇧Tab` switches fields, `⌃S` saves, `Esc` cancels. Alias is required and cannot contain spaces.

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

The same binary also runs as a Herdr plugin pane, showing the snippets beside the pane you are working in. Link the plugin once:

```sh
herdr plugin link /path/to/otty-snippets-view/herdr-plugin
```

and bind a key to its action in `~/.config/herdr/config.toml`:

```toml
[[keys.command]]
key = "prefix+y"
type = "plugin_action"
command = "local.otty-snippets.open"
description = "snippets"
```

`⏎` types the snippet into the pane the sidebar was opened from (Herdr's invocation context, not the sidebar itself), `o` only fills it in without pressing Enter. `SNIPPETS_TARGET_PANE` overrides the target. Snippets and groups are the same files the Otty view uses, so both hosts stay in sync.

Herdr has no widget slot in its own sidebar, so the plugin opens a split pane — the closest placement it offers. See [herdr-plugin/README.md](herdr-plugin/README.md) for details.

## Files

- `~/.config/otty/recipes/*.ottyrecipe`: the snippets themselves, the same files Otty's Recipes settings edit. Writes are atomic and keep Otty's format. A file with fields this tool doesn't know is never rewritten; edit it in Otty instead.
- `groups.json`: groups, manual assignments (`"none"` pins a snippet to Ungrouped) and UI state. A corrupt file is renamed to `groups.json.bad`.
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

`noquery/` must stay the first import in `main.go`. Bubble Tea's `init` queries the terminal background color and waits up to 5s, which leaves the panel blank. `noquery` sets the answer first, and the module is named `a/otty-snippets-view` so its import path sorts before `bubbletea`.

Code map: `store.go` (recipe and metadata files), `meta.go` (grouping rules, talking to Otty), `model.go` (state, rows, layout), `update.go` (keys, mouse), `actions.go`, `view.go` / `detail.go` (rendering), `highlight.go`, `form.go`, `i18n.go` (all UI strings). `dist/` holds shareable builds, not sources.

## Share it

Download `otty-snippets-view.zip` from the [latest release](https://github.com/elonnzhang/otty-snippets-view/releases/latest): a universal (arm64 + x86_64) build. It is not committed to git; build your own with the commands under Development. A colleague only needs the binary and one config line:

```toml
details-view = 'id="snippets" kind="tui" title="Snippets" icon="file" command="/path/to/otty-snippets-view" mobile="false"'
```

An agent can do the install from the prompt in [SETUP-PROMPT.md](SETUP-PROMPT.md). Then `otty config reload` and reopen the details panel. If Gatekeeper blocks the first launch, right-click → Open, or run `xattr -d com.apple.quarantine otty-snippets-view`. Snippets are not bundled: each person's own `~/.config/otty/recipes` is listed, and a fresh `groups.json` is created in `~/.config/otty/views/otty-snippets-view`. Sending to a pane needs `ipc-allow-send-keys = true`.
