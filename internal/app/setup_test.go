package app

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// fakeOtty puts an `otty` script on PATH that logs its arguments and answers
// `config get details-view` with current, one entry per line, the way Otty
// prints a list key. It is the only otty the tests see.
func fakeOtty(t *testing.T, current ...string) (calls string) {
	t.Helper()
	dir := t.TempDir()
	state := filepath.Join(dir, "details")
	if err := os.WriteFile(state, []byte(strings.Join(current, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	calls = filepath.Join(dir, "calls")
	// A list value spans lines; the log keeps one call per line by turning the
	// newlines inside a call into \036, which ottyCalls turns back.
	script := fmt.Sprintf(`#!/bin/sh
printf '%%s' "$*" | /usr/bin/tr '\n' '\036' >> %q
printf '\n' >> %q
case "$*" in
  *"config get details-view") /bin/cat %q 2>/dev/null ;;
  *"config get ipc-allow-send-keys") printf 'true\n' ;;
esac
`, calls, calls, state)
	if err := os.WriteFile(filepath.Join(dir, "otty"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	return calls
}

// ottyCalls is the arguments of each otty invocation, one call each.
func ottyCalls(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	s := strings.TrimRight(string(raw), "\n")
	if s == "" {
		return nil
	}
	got := strings.Split(s, "\n")
	for i, c := range got {
		got[i] = strings.ReplaceAll(c, "\036", "\n")
	}
	return got
}

// written is the details-view list a run set, and whether it set one.
func written(t *testing.T, calls string) ([]string, bool) {
	t.Helper()
	for _, c := range ottyCalls(t, calls) {
		if _, v, ok := strings.Cut(c, " config set details-view "); ok {
			return strings.Split(v, "\n"), true
		}
	}
	return nil, false
}

// unsetRan reports whether a run removed the details-view key.
func unsetRan(t *testing.T, calls string) bool {
	t.Helper()
	for _, c := range ottyCalls(t, calls) {
		if strings.Contains(c, " config unset ") {
			return true
		}
	}
	return false
}

func freshView(lang string) string {
	return `id="snippets" kind="tui" title="Snippets" icon="command" command=` +
		tomlString(commandFor(lang)) + ` mobile="false"`
}

func initCfg(t *testing.T, extra ...string) {
	t.Helper()
	if err := runInit(append([]string{"--config-file", "/tmp/cfg.toml"}, extra...)); err != nil {
		t.Fatal(err)
	}
}

func TestInitWritesDetailsView(t *testing.T) {
	calls := fakeOtty(t)
	initCfg(t)
	if got := ottyCalls(t, calls); len(got) == 0 ||
		got[0] != "--config-file /tmp/cfg.toml config get details-view" {
		t.Fatalf("calls = %q", got)
	}
	got, ok := written(t, calls)
	if want := []string{freshView("en")}; !ok || !reflect.DeepEqual(got, want) {
		t.Fatalf("written = %q, want %q", got, want)
	}
}

func TestInitKeepsLanguageAndFields(t *testing.T) {
	calls := fakeOtty(t, `id="snippets" kind="tui" title="Mine" icon="file" command="/old/otty-snippets-view --lang=cn" mobile="true"`)
	initCfg(t)
	got, ok := written(t, calls)
	if !ok || len(got) != 1 {
		t.Fatalf("written = %q", got)
	}
	for _, want := range []string{`title="Mine"`, `icon="file"`, `mobile="true"`, `command="` + commandFor("cn") + `"`} {
		if !strings.Contains(got[0], want) {
			t.Fatalf("entry %q does not contain %s", got[0], want)
		}
	}
	if strings.Contains(got[0], "/old/otty-snippets-view") {
		t.Fatalf("old command kept: %q", got[0])
	}
}

// details-view is a list: the other views survive, in their order, and their
// commands are not rewritten.
func TestInitKeepsOtherViews(t *testing.T) {
	git := `id="git" kind="tui" command="/x/lazygit --lang en" mobile="false"`
	calls := fakeOtty(t,
		`id="magpie" kind="web" title="Magpie" url="http://127.0.0.1:3430/"`,
		git,
		`id="snippets" kind="tui" title="Snippets" command="/old/otty-snippets-view --lang cn" mobile="false"`,
		`id="info"`,
	)
	initCfg(t)
	got, ok := written(t, calls)
	want := []string{
		`id="magpie" kind="web" title="Magpie" url="http://127.0.0.1:3430/"`,
		git,
		`id="snippets" kind="tui" title="Snippets" command=` + tomlString(commandFor("cn")) + ` mobile="false"`,
		`id="info"`,
	}
	if !ok || !reflect.DeepEqual(got, want) {
		t.Fatalf("written:\n got %q\nwant %q", got, want)
	}
}

func TestInitAddsToOtherViews(t *testing.T) {
	calls := fakeOtty(t, `id="info"`, `id="git"`)
	initCfg(t, "--lang", "cn")
	got, ok := written(t, calls)
	if want := []string{`id="info"`, `id="git"`, freshView("cn")}; !ok || !reflect.DeepEqual(got, want) {
		t.Fatalf("written:\n got %q\nwant %q", got, want)
	}
}

func TestInitPrintWritesNothing(t *testing.T) {
	for _, extra := range [][]string{{"--print"}, {"--print", "--reset"}} {
		calls := fakeOtty(t, `id="info"`, `id="snippets" command="/x/otty-snippets-view"`)
		initCfg(t, extra...)
		if _, ok := written(t, calls); ok || unsetRan(t, calls) {
			t.Fatalf("%q wrote: %q", extra, ottyCalls(t, calls))
		}
	}
}

func TestInitReset(t *testing.T) {
	// Only our entry goes; the rest is written back.
	calls := fakeOtty(t, `id="info"`, `id="snippets" command="/x/otty-snippets-view --lang en"`, `id="git"`)
	initCfg(t, "--reset")
	got, ok := written(t, calls)
	if want := []string{`id="info"`, `id="git"`}; !ok || !reflect.DeepEqual(got, want) || unsetRan(t, calls) {
		t.Fatalf("written = %q, calls = %q", got, ottyCalls(t, calls))
	}

	// Our entry was the only one: the key is removed, not set to nothing.
	calls = fakeOtty(t, `id="snippets" command="/x/otty-snippets-view"`)
	initCfg(t, "--reset")
	if _, ok := written(t, calls); ok || !unsetRan(t, calls) {
		t.Fatalf("calls = %q", ottyCalls(t, calls))
	}

	// Nothing of ours, or nothing at all: nothing is written.
	for _, cur := range [][]string{{`id="info"`, `id="other" command="/x/other"`}, nil} {
		calls = fakeOtty(t, cur...)
		initCfg(t, "--reset")
		if _, ok := written(t, calls); ok || unsetRan(t, calls) {
			t.Fatalf("reset of %q wrote: %q", cur, ottyCalls(t, calls))
		}
	}
}

func TestInitArgs(t *testing.T) {
	for _, bad := range [][]string{{"--lang", "xx"}, {"--nope"}, {"--force"}, {"--config-file"}} {
		if _, err := parseInitArgs(bad); err == nil {
			t.Fatalf("%q was accepted", bad)
		}
	}
	o, err := parseInitArgs([]string{"--lang=zh", "--print", "--reset", "--no-reload", "--config-file", "/c.toml"})
	if err != nil {
		t.Fatal(err)
	}
	if o.lang != "zh" || !o.print || !o.reset || !o.noReload || o.configFile != "/c.toml" {
		t.Fatalf("opts = %+v", o)
	}
}

func TestWithDetailsView(t *testing.T) {
	if got := withDetailsView(nil, ""); !reflect.DeepEqual(got, []string{freshView("en")}) {
		t.Fatalf("fresh = %q", got)
	}
	// An explicit language wins over the one in the current command.
	cur := []string{`id="snippets" command="/x/otty-snippets-view --lang cn" mobile="false"`}
	if got := withDetailsView(cur, "en"); !strings.Contains(got[0], `--lang en"`) {
		t.Fatalf("lang not replaced: %q", got)
	}
	// Our entry without a command is replaced by a fresh one.
	if got := withDetailsView([]string{`id="snippets"`}, "cn"); !reflect.DeepEqual(got, []string{freshView("cn")}) {
		t.Fatalf("no-command entry = %q", got)
	}
	// currentLang reads both spellings and ignores junk.
	for in, want := range map[string]string{
		`command="/x --lang=cn"`:      "zh",
		`command="/x --lang zh-CN"`:   "zh",
		`command="/x --lang en"`:      "en",
		`command="/x"`:                "en",
		`command="/x --lang klingon"`: "en",
		`command="/x --lang 中文"`:      "zh",
	} {
		if got := currentLang(in); got != want {
			t.Fatalf("currentLang(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDetailsViews(t *testing.T) {
	got := detailsViews("id=\"a\"\n\n  id=\"b\" kind=\"tui\"  \n")
	if want := []string{`id="a"`, `id="b" kind="tui"`}; !reflect.DeepEqual(got, want) {
		t.Fatalf("detailsViews = %q", got)
	}
	if got := detailsViews("\n"); got != nil {
		t.Fatalf("empty output = %q", got)
	}
}

func TestDetailsViewOurs(t *testing.T) {
	for _, v := range []string{
		`id="snippets" kind="tui"`,
		`command="/x/otty-snippets-view --lang en"`,
	} {
		if !detailsViewOurs(v) {
			t.Fatalf("%q is not recognised as ours", v)
		}
	}
	for _, v := range []string{"", `id="other" command="/x/other"`} {
		if detailsViewOurs(v) {
			t.Fatalf("%q is recognised as ours", v)
		}
	}
}
