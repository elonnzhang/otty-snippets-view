# AGENTS.md

This file provides guidance to coding agents (Claude Code, Codex and others) when working with code in this repository.

otty-snippets-view is a Bubble Tea TUI for Otty's details panel; the same binary also runs as a Herdr plugin pane. It lists Otty's text-snippet recipes, sorts them into a group tree and types the chosen snippet into the focused terminal pane. README.md is the spec for user-facing behaviour (keys, grouping rules, dialogs, install options); read it before changing behaviour.

## Commands

```sh
go vet ./... && go test ./...           # the checks release.sh runs
go test -run TestNestedGroups -v ./...  # one test; every test is in package app under internal/
go build -o /tmp/otty-snippets-view .   # compile check (don't build into the repo root, see below)
./release.sh                            # vet + test, then build and install into the live Otty panel
./release.sh --fast                     # skip vet/test; --zip also builds the universal dist/ zip (lipo, ditto)
sh install.sh --from dev                # build + install + refresh the panel, without the checks
```

There is no linter config; `gofmt -l .` should stay empty. The root is only `main.go`, which calls `app.Run` (`internal/app/run.go`).

## This checkout is also the live install

The repo lives at `~/.config/otty/views/otty-snippets-view`, which is both install.sh's default install dir and the default `SNIPPETS_HOME`; Otty's `details-view` runs `./otty-snippets-view` from here. The gitignored files in the root are the user's real state, not fixtures:

- `otty-snippets-view`, `.bak`, `.version` are the binary Otty runs and its install record. `go build -o otty-snippets-view .` here replaces it.
- `groups.json` and `trash/` are the user's real groups, assignments, colors and deleted recipes.
- Running the binary, or a test that skips the env setup, without `OTTY_RECIPES_DIR` and `SNIPPETS_HOME` reads and writes the real `~/.config/otty/recipes` and this `groups.json`.
- `release.sh` / `install.sh --from dev` rewrite `details-view` in Otty's config, reload it and stop the running view. That is the intended dev loop, but it acts on the user's running app. `otty-snippets-view init` also writes the real config unless given `--config-file <tmp>` or `--print`; and run as a `go build` output elsewhere it points the panel at that file.
- `details-view` is a list key (one view per line in `otty config get`, the whole list in `otty config set`). Anything that writes it must change only the snippets entry and write the other views back; `otty config unset details-view` removes all of them.

Never enable or change `ipc-allow-send-keys` or `ipc-allow-sensitive-sessions`. They are the user's security switches; install.sh and SETUP-PROMPT.md only remind about them.

## Architecture

The root `main.go` starts the program; all view code is `package app` in `internal/app/`. `internal/noquery/`, which main.go must keep importing, answers lipgloss's start-up background-colour query, which otherwise leaves the panel blank for up to 5 s. README's "Code map" line says which file holds what.

**Data.** Snippets are Otty's own files (`*.ottyrecipe`, TOML, `scope = "text_snippet"`). `saveSnippet` writes them by hand in Otty's encoder format (`tomlStr`) via tmp file + rename. A file with unknown fields or `version != 1` is parsed with `Foreign` set and must never be rewritten. Deleting moves the file to `trash/`. Everything this tool adds (groups, manual assignments, colors, tab, detail height) lives in `Meta`, stored in `groups.json`; nothing goes into recipe files.

**Grouping** (meta.go). `groupOf` resolves a snippet in this order: an explicit `Assign` entry (`"none"` pins it to Ungrouped), then the group named by its `[group]-` name prefix (`auto`, rebuilt by `resolve`, which also creates missing groups), then Ungrouped. Ungrouped is gid `""` and only a display label, never a stored group. `remove`, `rename` and `move` add or drop `Assign` entries so that `resolve` does not undo them (e.g. deleting a group pins its prefixed members to `"none"`). Groups form a tree through `Parent`; walk it with `children` / `treeOrder`, not slice order.

**State and update.** `model` (model.go) holds all state. `mode` decides which handler `onKey` dispatches to and which overlay `View` draws. `reload()` loads snippets, prunes and resolves meta, then `rebuild()` flattens everything into `rows` for the current tab (List: flat; Groups: tree walk with folding). `sel` indexes `rows`, and `-1` means nothing is selected. A 1 s `tick` compares `signature()` (path + mtime of the recipes) and reloads, but only in `modeNormal` with no drag or resize in progress. Call `persist()` after changing `Meta`.

**Rendering and hit-testing.** `View` builds the screen top to bottom: tab bar, list, divider, detail, status (overwrites the last detail row), two hint rows. Every line goes through `fitw` (ANSI-aware truncate and pad). Dialogs, the context menu and the form are painted over the finished lines with `paintBox`. Mouse handling recomputes the same geometry instead of storing it: `layout()`, `tabSpans`, `hintSpans` (mirrors `hintRows` column by column), `dialogBox`, `confirmBox`, `formBox`, `ctxBox`. When you change how something is drawn, change its geometry function so clicks still land. A clicked hint replays its key through `clickHint` → `onKey`. form.go keeps its own wrap and scroll bookkeeping (`measure`, `formFollow`, `clickText`) because the bubbles textarea exposes neither its viewport offset nor its wrapped lines.

**Sending** (meta.go, herdr.go). Everything goes through CLIs; Herdr's Go bindings are deliberately not imported. `sendSnippet` picks the path:

- Herdr plugin pane (`HERDR_PANE_ID` set): `herdr pane send-text` into the invoking pane, never the plugin's own pane. Enter is a separate `send-keys`.
- Otty panel: `focusedPane()` asks `otty pane show --json` and ignores the view's own `OTTY_PANE_ID`. A view process has no `otty` on PATH, so `ottyCLI()` falls back to the app bundle. Text goes through `otty pane send-text`, and exit code 7 means Otty refused (its stderr names the reason).
- An Otty pane running a Herdr client: Otty refuses to type there, so the snippet goes through that client's Herdr session. Otty's `ipc-allow-send-keys` and `ipc-allow-sensitive-sessions` rules are re-checked first, and when Herdr can't report the target's foreground process the send fails closed.

`expand()` fills `{{clipboard}}` / `{{date}}` / `{{time}}` and strips `{{cursor}}`. The program is macOS-only (`pbcopy`/`pbpaste`, the Otty app path). Under Herdr, `app.Run` skips the alt screen so the pane reads as a sidebar column.

**Herdr plugin** (`herdr-plugin/`). A manifest plus shell scripts that build the parent directory into `herdr-plugin/.build/`; `run.sh` rebuilds when any `.go` file in the checkout is newer than that binary, so source changes need no extra step there.

**Strings and styles.** Every UI string is a `{English, Chinese}` pair in `dict` (i18n.go), read with `tr(key)`; a missing key renders as the key itself. One-off pairs use `pick(en, zh)`. Built-in styles and item colors use ANSI palette indices (`lipgloss.Color("4")`), not fixed RGB, so the view follows the Otty theme.

## Tests

UI tests drive the whole model without a terminal. `ctxTestModel(t)` (internal/app/ctxmenu_test.go) points `OTTY_RECIPES_DIR` and `SNIPPETS_HOME` at temp dirs, sets `lang = "en"`, saves one `[db]-ping` snippet and returns a 60×20 model in the Groups tab. Feed it `m.onKey(tea.KeyMsg{...})` and `m.onMouse(...)` (helpers: `click`, `rightClick`, `rowIndex`, `hintColumn`, `openForm`, `openConfirm`), then assert on `ansi.Strip(m.View())` or model fields. Any new test that touches storage needs the same env isolation. Herdr tests replace the CLI with a shell script through `HERDR_BIN_PATH` (and `PATH`) that logs its arguments.

## Conventions

- User-visible changes update README.md and README_zh.md together; the Chinese README is a full translation, not a summary.
- Commit subjects are short and imperative, sometimes prefixed with the area (`install.sh: …`, `README: …`); the body explains what changed for the user and why.
