package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func TestHighlight(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI)
	text := "curl -sS \"https://example.com/a/very/long/path\" \\\n  -H \"Authorization: Bearer ${KEY}\"\n# done"
	var c hlCache
	lines := c.highlight(text, 20)
	var plain []string
	for _, l := range lines {
		if w := ansi.StringWidth(l); w > 20 {
			t.Errorf("line wider than 20: %d %q", w, l)
		}
		plain = append(plain, ansi.Strip(l))
	}
	if got := strings.Join(plain, ""); got != strings.ReplaceAll(text, "\n", "") {
		t.Errorf("text changed:\n%q\n%q", got, text)
	}
	if !strings.Contains(strings.Join(lines, ""), "\x1b[") {
		t.Errorf("no color applied: %q", lines)
	}
	if again := c.highlight(text, 20); &again[0] != &lines[0] {
		t.Error("cache miss on identical input")
	}
}
