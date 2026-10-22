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
	"path/filepath"
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
	if herdrJSON(&r, "pane", "current") != nil {
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

// ---- Otty pane hosting a Herdr client ----------------------------------------
//
// When the focused Otty pane runs a Herdr client, Otty refuses to type into it:
// the client keeps its remote-machine ssh bridges in the pane's foreground, so
// Otty sees an SSH session. The keys would only reach Herdr's focused pane
// anyway, so send there through Herdr instead, with Otty's rule kept: a target
// that is itself an SSH session needs ipc-allow-sensitive-sessions.

// herdrClient is a Herdr client process found in an Otty pane.
type herdrClient struct {
	session string // --session of the client, "" for the default one
	socket  string // HERDR_SOCKET_PATH in the client's environment, if any
}

// herdrClientIn finds the Herdr client in the foreground of the Otty pane.
func herdrClientIn(ottyPane string) (herdrClient, bool) {
	out, err := exec.Command("ps", "-axo", "pid=,stat=,args=").Output()
	if err != nil {
		return herdrClient{}, false
	}
	want := "OTTY_PANE_ID=" + strings.TrimPrefix(ottyPane, "p_")
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 3 || !strings.Contains(f[1], "+") || filepath.Base(f[2]) != "herdr" || isHerdrServer(f[3:]) {
			continue
		}
		env, err := exec.Command("ps", "eww", "-o", "command=", "-p", f[0]).Output()
		if err != nil {
			continue
		}
		if c, ok := parseHerdrClient(string(env), want); ok {
			return c, true
		}
	}
	return herdrClient{}, false
}

func isHerdrServer(args []string) bool {
	return len(args) > 0 && (args[0] == "server" || args[0] == "remote-client-bridge")
}

// parseHerdrClient reads `ps eww` output of a herdr process: argv, then env.
func parseHerdrClient(cmdline, wantPane string) (herdrClient, bool) {
	var c herdrClient
	match := false
	f := strings.Fields(cmdline)
	for i, tok := range f {
		switch {
		case tok == wantPane:
			match = true
		case tok == "--session" && i+1 < len(f):
			c.session = f[i+1]
		case strings.HasPrefix(tok, "--session="):
			c.session = strings.TrimPrefix(tok, "--session=")
		case strings.HasPrefix(tok, "HERDR_SOCKET_PATH="):
			c.socket = strings.TrimPrefix(tok, "HERDR_SOCKET_PATH=")
		}
	}
	return c, match
}

// cmd runs the herdr CLI against the session this client is attached to.
func (c herdrClient) cmd(args ...string) *exec.Cmd {
	if c.session != "" {
		args = append([]string{"--session", c.session}, args...)
	}
	cmd := exec.Command(herdrBin(), args...)
	if c.socket != "" {
		cmd.Env = append(os.Environ(), "HERDR_SOCKET_PATH="+c.socket)
	}
	return cmd
}

func (c herdrClient) json(out any, args ...string) error {
	raw, err := c.cmd(args...).Output()
	if err != nil {
		return err
	}
	var env struct{ Result json.RawMessage }
	if err := json.Unmarshal(raw, &env); err != nil {
		return err
	}
	return json.Unmarshal(env.Result, out)
}

// sensitiveProcs are foreground programs Otty treats as remote sessions.
var sensitiveProcs = map[string]bool{"ssh": true, "mosh": true, "mosh-client": true, "autossh": true, "et": true, "telnet": true}

// sendSnippetViaHerdrClient types the snippet into Herdr's focused pane.
func sendSnippetViaHerdrClient(c herdrClient, s Snippet, run bool) string {
	var cur struct {
		Pane struct {
			PaneID string `json:"pane_id"`
		} `json:"pane"`
	}
	if err := c.json(&cur, "pane", "current"); err != nil || cur.Pane.PaneID == "" {
		return tr("send.nopane")
	}
	pane := cur.Pane.PaneID
	var pi struct {
		ProcessInfo struct {
			Foreground []struct {
				Name string `json:"name"`
			} `json:"foreground_processes"`
		} `json:"process_info"`
	}
	if err := c.json(&pi, "pane", "process-info", "--pane", pane); err != nil {
		// Unknown foreground: fail closed rather than risk typing into a remote shell.
		return fmt.Sprintf(tr("send.failed"), strings.TrimSpace(err.Error()))
	}
	for _, p := range pi.ProcessInfo.Foreground {
		if sensitiveProcs[filepath.Base(p.Name)] && !ottyFlag("ipc-allow-sensitive-sessions") {
			return fmt.Sprintf(tr("send.sensitive"), pane)
		}
	}
	if err := c.cmd("pane", "send-text", pane, expand(s.Text)).Run(); err != nil {
		return fmt.Sprintf(tr("send.failed"), strings.TrimSpace(err.Error()))
	}
	if !run {
		return tr("send.filled")
	}
	if err := c.cmd("pane", "send-keys", pane, "enter").Run(); err != nil {
		return fmt.Sprintf(tr("send.failed"), err)
	}
	return tr("send.ran")
}

// ottyFlag reads a boolean Otty setting; anything but "true" counts as off.
func ottyFlag(key string) bool {
	out, err := exec.Command(ottyCLI(), "config", "get", key).Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}
