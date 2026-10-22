# Herdr plugin for otty-snippets-view

Runs the same program as a Herdr pane, so Otty's snippet recipes are one keypress away in Herdr: pick a snippet, press `⏎`, and it is typed into the pane the sidebar was opened from.

Herdr has no plugin API for placing a widget inside its own sidebar, so the plugin opens a narrow split pane beside the current one. `split` is the closest thing Herdr offers.

## Install

From this directory (the plugin root):

```sh
herdr plugin link .
```

Then bind a key in `~/.config/herdr/config.toml` to the plugin action and reload the config:

```toml
[[keys.command]]
key = "prefix+y"
type = "plugin_action"
command = "local.otty-snippets.open"
description = "snippets"
```

Herdr qualifies action ids as `plugin-id.action-id`, so the `command` value is `local.otty-snippets.open`. If the plugin is installed under another id, adjust the prefix to match. The action runs `open.sh`, which asks Herdr to open the `snippets` pane as a split to the right and focus it.

To open the pane without a keybinding, the same in one command:

```sh
herdr plugin pane open --plugin local.otty-snippets --entrypoint snippets \
  --placement split --direction right --focus
```

## Behaviour

- The pane runs `run.sh`, which builds `..` with `go build` on first use (and whenever a `.go` file is newer) into `.build/`, then execs it. A failed build keeps the previous binary.
- `HERDR_PANE_ID` is set, so the program draws inline instead of using the alternate screen: it behaves like a column, and closing the pane does not disturb the tab.
- `⏎` runs the snippet in the pane the plugin was invoked for; `o` only fills it in. The target comes from Herdr's invocation context (`focused_pane_id`), never the plugin's own pane. `SNIPPETS_TARGET_PANE` overrides it.
- Everything else works as in Otty: groups, nested groups, search, auto-grouping by `[group]-name`, English/Chinese UI.
- The data is shared with the Otty view: `~/.config/otty/recipes` for snippets and `~/.config/otty/views/otty-snippets-view/groups.json` for groups.

## Layout notes

A split pane is as narrow as your layout allows. The tab bar, list and hints adapt; the detail area is hidden when the pane is shorter than six rows. Use a wider split (`herdr pane resize`) if you want the detail area visible.
