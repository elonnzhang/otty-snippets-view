# Setup prompt for an agent

Copy the block below into a coding agent on the machine where Otty should show the panel.

```text
Set up the Otty Snippets details-panel view (otty-snippets-view) on this machine. Do this
inside an Otty pane, with OTTY_PANE_ID set; if it is empty, stop and say so.

Run the installer from the project's default branch, then check what it did:

1. Install
   curl -fsSL https://raw.githubusercontent.com/elonizzhang/otty-snippets-view/main/install.sh | sh

   It downloads the latest release, places the binary in ~/.config/otty/views/otty-snippets-view/,
   writes `details-view` with `otty config set`, reloads the config and stops any old view.
   If it refuses because details-view belongs to another tool, stop and ask me — don't add --force.

2. Verify
   - otty config validate   (must pass)
   - otty config get details-view   (must contain command="/…/otty-snippets-view")
   - Tell me that Otty does not restart a running view, so I have to press ⌘⇧R twice
     (close and reopen the details panel).

3. Leave alone
   - Do not enable or change `ipc-allow-send-keys`. It is Otty's global security switch for
     letting a view send text to a terminal pane, and I decide whether it is on. Just mention in
     your report that run/fill will report "rejected (check ipc-allow-send-keys)" while it is off.

4. Report back
   - The exact config line and the installed version (cat ~/.config/otty/views/otty-snippets-view/.version)
   - The results of validate and the config read
   - What I need to do next (⌘⇧R twice)
   - Where the data lives: snippets are read from ~/.config/otty/recipes/*.ottyrecipe; groups and
     UI state go to ~/.config/otty/views/otty-snippets-view/groups.json, created on first run
   - If the panel comes up blank, ask me what it shows instead of restarting Otty repeatedly
```
