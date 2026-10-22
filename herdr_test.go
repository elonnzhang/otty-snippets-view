package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHerdrTarget(t *testing.T) {
	t.Setenv("HERDR_PANE_ID", "w1:p2")
	t.Setenv("HERDR_PLUGIN_CONTEXT_JSON", `{"focused_pane_id":"w1:p1"}`)
	if got := herdrTarget(); got != "w1:p1" {
		t.Fatalf("target = %q", got)
	}
	// Focus on the sidebar itself is useless: fall through to the event.
	t.Setenv("HERDR_PLUGIN_CONTEXT_JSON", `{"focused_pane_id":"w1:p2"}`)
	t.Setenv("HERDR_PLUGIN_EVENT_JSON", `{"pane_id":"w1:p3"}`)
	t.Setenv("HERDR_BIN_PATH", "/nonexistent") // keep the CLI fallback out of the test
	if got := herdrTarget(); got != "w1:p3" {
		t.Fatalf("event target = %q", got)
	}
	// Explicit override wins over everything.
	t.Setenv("SNIPPETS_TARGET_PANE", "w9:p9")
	if got := herdrTarget(); got != "w9:p9" {
		t.Fatalf("override target = %q", got)
	}
}

func TestSendSnippetHerdr(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	bin := filepath.Join(dir, "herdr")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + log + "\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HERDR_BIN_PATH", bin)
	t.Setenv("HERDR_PANE_ID", "w1:p2")
	t.Setenv("SNIPPETS_TARGET_PANE", "w1:p1")

	lang = "en"
	if got := sendSnippetHerdr(Snippet{Text: "echo hi"}, true); got != tr("send.ran") {
		t.Fatalf("run result = %q", got)
	}
	raw, _ := os.ReadFile(log)
	calls := strings.Split(strings.TrimSpace(string(raw)), "\n")
	want := []string{"pane send-text w1:p1 echo hi", "pane send-keys w1:p1 enter"}
	if strings.Join(calls, "|") != strings.Join(want, "|") {
		t.Fatalf("calls = %q", calls)
	}

	os.Remove(log)
	if got := sendSnippetHerdr(Snippet{Text: "ls"}, false); got != tr("send.filled") {
		t.Fatalf("fill result = %q", got)
	}
	raw, _ = os.ReadFile(log)
	if strings.TrimSpace(string(raw)) != "pane send-text w1:p1 ls" {
		t.Fatalf("fill calls = %q", raw)
	}
}
