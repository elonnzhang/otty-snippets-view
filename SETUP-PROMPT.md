# Setup prompt for an agent

Copy the block below into a coding agent on the machine where Otty should show the panel. Replace `<path>` first.

```text
Set up the Otty Snippets details-panel view (otty-snippets-view) on this machine.

I have already put the otty-snippets-view executable at <path> (for example ~/bin/otty-snippets-view).
If that path is missing or is not a file, stop and ask me — do not download or guess a path.

Do these steps in order and stop with the reason if any step fails:

1. Check the binary
   - chmod +x <path>
   - If Gatekeeper will block it (`xattr -l <path>` shows com.apple.quarantine), run `xattr -d com.apple.quarantine <path>`.
   - Verify it runs: `<path> --lang fr` must print `unsupported --lang "fr" (use en or cn)` and exit with code 2.
   - Running it directly may also print `could not open a new TTY`; that is expected, it is not meant to be run by hand.

2. Register the view
   - Read ~/.config/otty/config.toml first. `details-view` may appear several times (for the built-in views); never delete those lines.
   - Write it with the CLI, not by editing TOML quoting by hand:
     otty config set details-view 'id="snippets" kind="tui" title="Snippets" icon="file" command="<path>" mobile="false"'
   - The value must be quoted key="value" pairs. JSON or a bare string is rejected by Otty.
   - If a `details-view` line with id="snippets" already exists, replace it rather than adding a second one.

3. Apply
   - otty config validate   (must pass)
   - otty config reload
   - Tell me that Otty does not restart a running view, so I have to press ⌘⇧R twice (close and reopen the details panel) and pick Snippets.

4. Leave alone
   - Do not enable or change `ipc-allow-send-keys`. It is Otty's global security switch for letting a view send text to a terminal pane, and I decide whether it is on. Just mention in your report that run/fill will report "rejected (check ipc-allow-send-keys)" while it is off.

5. Report back
   - The exact config line written, with the real path
   - The results of validate and reload
   - What I need to do next (⌘⇧R twice)
   - Where the data lives: snippets are read from ~/.config/otty/recipes/*.ottyrecipe; groups and UI state go to ~/.config/otty/views/otty-snippets-view/groups.json, created on first run
   - If the panel comes up blank, ask me what it shows instead of restarting Otty repeatedly
```
