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

func TestParseHerdrClient(t *testing.T) {
	line := "herdr --session work TERM=xterm OTTY_PANE_ID=1a_22 HERDR_SOCKET_PATH=/tmp/h.sock"
	c, ok := parseHerdrClient(line, "OTTY_PANE_ID=1a_22")
	if !ok || c.session != "work" || c.socket != "/tmp/h.sock" {
		t.Fatalf("got %+v %v", c, ok)
	}
	if _, ok := parseHerdrClient(line, "OTTY_PANE_ID=1a_2"); ok {
		t.Fatal("prefix of another pane id must not match")
	}
	if !isHerdrServer([]string{"server", "--handoff-import"}) || isHerdrServer(nil) {
		t.Fatal("server detection")
	}
}

func TestSendViaHerdrClient(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	fg := filepath.Join(dir, "fg")
	bin := filepath.Join(dir, "herdr")
	script := `#!/bin/sh
printf '%s\n' "$*" >> ` + log + `
case "$*" in
*"pane current"*) echo '{"result":{"pane":{"pane_id":"w1:p5"}}}' ;;
*"process-info"*) read n < ` + fg + `; printf '{"result":{"process_info":{"foreground_processes":[{"name":"%s"}]}}}' "$n" ;;
esac
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HERDR_BIN_PATH", bin)
	// A fake otty that reports every setting as off.
	if err := os.WriteFile(filepath.Join(dir, "otty"), []byte("#!/bin/sh\necho false\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	lang = "en"
	c := herdrClient{session: "work"}

	os.WriteFile(fg, []byte("zsh\n"), 0o644)
	if got := sendSnippetViaHerdrClient(c, Snippet{Text: "echo hi"}, true); got != tr("send.ran") {
		t.Fatalf("run = %q", got)
	}
	raw, _ := os.ReadFile(log)
	if !strings.Contains(string(raw), "--session work pane send-text w1:p5 echo hi") ||
		!strings.Contains(string(raw), "--session work pane send-keys w1:p5 enter") {
		t.Fatalf("calls = %s", raw)
	}

	os.Remove(log)
	os.WriteFile(fg, []byte("ssh\n"), 0o644)
	if got := sendSnippetViaHerdrClient(c, Snippet{Text: "rm -rf x"}, true); !strings.Contains(got, "SSH") {
		t.Fatalf("ssh target = %q", got)
	}
	raw, _ = os.ReadFile(log)
	if strings.Contains(string(raw), "send-text") {
		t.Fatalf("must not type into an SSH pane: %s", raw)
	}
}
