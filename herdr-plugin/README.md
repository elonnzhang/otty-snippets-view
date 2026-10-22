# Herdr plugin for otty-snippets-view

Runs the same program as a Herdr pane, so Otty's snippet recipes are one keypress away in Herdr: pick a snippet, press `⏎`, and it is typed into the pane the sidebar was opened from.

Herdr has no plugin API for placing a widget inside its own sidebar, so the plugin opens a narrow split pane beside the current one. `split` is the closest thing Herdr offers.

## Install from GitHub

```sh
herdr plugin install elonnzhang/otty-snippets-view/herdr-plugin
```

The subdirectory matters: the repository root holds the Go program, the plugin manifest lives in `herdr-plugin/`. `plugin install` runs the manifest's `[[build]]` step (`build.sh`), so installing needs a Go toolchain on the machine. The checkout stays under `~/.config/herdr/plugins/github-installations/`; add `--ref v1.1.0` to pin a release.

Then bind a key to the plugin action and reload the config:

```toml
[[keys.command]]
key = "prefix+shift+s"
type = "plugin_action"
command = "local.otty-snippets.open"
description = "snippets"
```

Herdr qualifies action ids as `plugin-id.action-id`, so the value is `local.otty-snippets.open` — the same for a GitHub install and a local link, because the id comes from the manifest, not the source. The action runs `open.sh`, which asks Herdr to open the `snippets` pane as a split to the right and focus it.

To open the pane without a keybinding:

```sh
herdr plugin pane open --plugin local.otty-snippets --entrypoint snippets \
  --placement split --direction right --focus
```

## Install from a local checkout

```sh
herdr plugin link /path/to/otty-snippets-view/herdr-plugin
```

`plugin link` does not run build commands; `run.sh` builds on first use instead.

## Updating

```sh
herdr plugin update local.otty-snippets
```

A new installation directory replaces the old one. The running pane keeps its old files until you close it; the next pane you open uses the new build, no config reload needed.

## Behaviour

- The pane runs `run.sh`: it builds `..` into `.build/` when the binary is missing or older than a `.go` file, then execs it. A failed build keeps the previous binary; without one, the pane stays open for ten seconds so the error is readable.
- `build.sh` needs Go on `PATH`. If Go is missing but a binary already exists in `.build/`, it keeps that binary instead of failing.
- `HERDR_PANE_ID` is set, so the program draws inline instead of using the alternate screen: it behaves like a column, and closing the pane does not disturb the tab.
- `⏎` runs the snippet in the pane the plugin was invoked for; `o` only fills it in. The target comes from Herdr's invocation context (`focused_pane_id`), never the plugin's own pane. `SNIPPETS_TARGET_PANE` overrides it.
- Everything else works as in Otty: groups, nested groups, search, auto-grouping by `[group]-name`, English/Chinese UI (`--lang`, or `SNIPPETS_LANG`).
- The data is shared with the Otty view: `~/.config/otty/recipes` for snippets and `~/.config/otty/views/otty-snippets-view/groups.json` for groups. A GitHub install does not bring your recipes along; it reads the local ones.

## Layout notes

A split pane is as narrow as your layout allows. The tab bar, list and hints adapt; the detail area is hidden when the pane is shorter than six rows. Use a wider split (`herdr pane resize`) if you want the detail area visible.
