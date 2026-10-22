package main

// Herdr adapter.
//
// The program serves two hosts:
//
//   - Otty's details panel (no Herdr environment): full-screen, snippets go to
//     the focused Otty pane through `otty pane send-text`.
//   - A Herdr plugin pane (HERDR_PANE_ID is set, started by herdr-plugin/run.sh):
//     drawn inline so it reads as a sidebar column, and snippets go to the pane
//     Herdr invoked the plugin for.
//
// Everything host-specific lives here; the UI code is shared. Herdr's own Go
// bindings are deliberately not imported: shelling out to the CLI costs nothing
// at startup and adds no dependency to the binary.

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// inHerdr reports whether this process was started by Herdr.
func inHerdr() bool { return os.Getenv("HERDR_PANE_ID") != "" }

// herdrCtx is the slice of Herdr's invocation context this program uses.
type herdrCtx struct {
	FocusedPaneID  string `json:"focused_pane_id"`
	FocusedPaneCwd string `json:"focused_pane_cwd"`
	WorkspaceID    string `json:"workspace_id"`
	TabID          string `json:"tab_id"`
}

func herdrJSONEnv() herdrCtx {
	var c herdrCtx
	if raw := os.Getenv("HERDR_PLUGIN_CONTEXT_JSON"); raw != "" {
		_ = json.Unmarshal([]byte(raw), &c)
	}
	return c
}

// herdrTarget is the pane snippets are sent to. The plugin's own pane is never
// the target: a snippet must not type into the sidebar itself.
func herdrTarget() string {
	if p := os.Getenv("SNIPPETS_TARGET_PANE"); p != "" {
		return p
	}
	self := os.Getenv("HERDR_PANE_ID")
	if p := herdrJSONEnv().FocusedPaneID; p != "" && p != self {
		return p
	}
	// An event hook carries the pane it fired for.
	if raw := os.Getenv("HERDR_PLUGIN_EVENT_JSON"); raw != "" {
		var ev struct {
			PaneID        string `json:"pane_id"`
			FocusedPaneID string `json:"focused_pane_id"`
		}
		if json.Unmarshal([]byte(raw), &ev) == nil {
			for _, p := range []string{ev.FocusedPaneID, ev.PaneID} {
				if p != "" && p != self {
					return p
				}
			}
		}
	}
	if p := herdrCurrentPane(); p != "" && p != self {
		return p
	}
	return ""
}

func herdrBin() string {
	if p := os.Getenv("HERDR_BIN_PATH"); p != "" {
		return p
	}
	if p, err := exec.LookPath("herdr"); err == nil {
		return p
	}
	return "herdr"
}

// herdrJSON runs a herdr command that prints its usual {"result": ...} envelope
// and unmarshals the result into out.
func herdrJSON(out any, args ...string) error {
	raw, err := exec.Command(herdrBin(), args...).Output()
	if err != nil {
		return err
	}
	var env struct{ Result json.RawMessage }
	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}
	return json.Unmarshal(env.Result, out)
}

func herdrCurrentPane() string {
	var r struct {
		Pane struct {
			PaneID string `json:"pane_id"`
		} `json:"pane"`
	}
	if herdrJSON(&r, "pane", "current", "--json") != nil {
		return ""
	}
	return r.Pane.PaneID
}

// sendSnippetHerdr types the snippet into the target pane. Enter is a separate
// call so a snippet that ends in a newline cannot submit twice.
func sendSnippetHerdr(s Snippet, run bool) string {
	pane := herdrTarget()
	if pane == "" {
		return tr("send.nopane")
	}
	if err := exec.Command(herdrBin(), "pane", "send-text", pane, expand(s.Text)).Run(); err != nil {
		return fmt.Sprintf(tr("send.failed"), strings.TrimSpace(err.Error()))
	}
	if !run {
		return tr("send.filled")
	}
	if err := exec.Command(herdrBin(), "pane", "send-keys", pane, "enter").Run(); err != nil {
		return fmt.Sprintf(tr("send.failed"), err)
	}
	return tr("send.ran")
}
