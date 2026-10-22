package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

var ctxSnipID string

func ctxTestModel(t *testing.T) *model {
	t.Helper()
	rec := t.TempDir()
	t.Setenv("OTTY_RECIPES_DIR", rec)
	t.Setenv("SNIPPETS_HOME", t.TempDir())
	lang = "en"
	id, err := saveSnippet(Snippet{Name: "[db]-ping", Alias: "ping", Text: "echo ping"})
	if err != nil {
		t.Fatal(err)
	}
	ctxSnipID = id
	m := newModel()
	m.w, m.h = 60, 20
	m.setTab(tabGroup)
	return m
}

func rowIndex(m *model, match func(row) bool) int {
	for i, r := range m.rows {
		if match(r) {
			return i
		}
	}
	return -1
}

func rightClick(m *model, x, y int) {
	m.onMouse(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonRight, Action: tea.MouseActionPress})
}

func TestContextMenuGroup(t *testing.T) {
	m := ctxTestModel(t)
	l := m.layout()
	gi := rowIndex(m, func(r row) bool { return r.group && r.gid != "" })
	if gi < 0 {
		t.Fatalf("no named group in rows: %+v", m.rows)
	}
	rightClick(m, 5, l.listTop+gi-m.scroll)
	if m.mode != modeContext || m.ctx == nil {
		t.Fatalf("mode = %v, want context menu", m.mode)
	}
	var keys []string
	for _, it := range m.ctx.items {
		keys = append(keys, it.key)
	}
	if strings.Join(keys, "") != "Mrmdan" {
		t.Fatalf("group items = %q", keys)
	}
	// The box keeps every line at the screen width.
	for i, line := range strings.Split(m.View(), "\n") {
		if w := ansi.StringWidth(line); w != m.w {
			t.Fatalf("line %d width %d, want %d", i, w, m.w)
		}
	}
	// Picking "Rename" with the mouse opens the rename prompt for that group.
	x, y, _, _ := m.ctxBox()
	m.onMouse(tea.MouseMsg{X: x + 2, Y: y + 2, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if m.mode != modePrompt || m.promptGroup != m.rows[gi].gid {
		t.Fatalf("after rename: mode %v, promptGroup %q", m.mode, m.promptGroup)
	}
}

func TestContextMenuSnippetAndEmpty(t *testing.T) {
	m := ctxTestModel(t)
	l := m.layout()
	si := rowIndex(m, func(r row) bool { return !r.group })
	rightClick(m, 5, l.listTop+si-m.scroll)
	if m.mode != modeContext || m.ctx.items[0].key != "o" {
		t.Fatalf("snippet menu = %+v", m.ctx)
	}
	// The item's own shortcut picks it: d asks to delete the snippet.
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if m.mode != modeConfirm || m.confirmSnip != ctxSnipID {
		t.Fatalf("after d: mode %v, confirmSnip %q", m.mode, m.confirmSnip)
	}
	m.onKey(tea.KeyMsg{Type: tea.KeyEsc})

	// Empty space below the rows: only "new" items; Esc closes.
	rightClick(m, 5, l.listTop+len(m.rows))
	if m.mode != modeContext || len(m.ctx.items) != 2 || m.ctx.items[0].key != "n" {
		t.Fatalf("empty-space menu = %+v", m.ctx)
	}
	m.onKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != modeNormal || m.ctx != nil {
		t.Fatal("Esc did not close the menu")
	}
}

func TestContextMenuHover(t *testing.T) {
	m := ctxTestModel(t)
	l := m.layout()
	gi := rowIndex(m, func(r row) bool { return r.group && r.gid != "" })
	m.Update(tea.MouseMsg{X: 5, Y: l.listTop + gi - m.scroll, Button: tea.MouseButtonRight, Action: tea.MouseActionPress})
	if !m.hover {
		t.Fatal("opening the menu should turn on pointer motion reports")
	}
	x, y, _, _ := m.ctxBox()
	m.Update(tea.MouseMsg{X: x + 3, Y: y + 4, Button: tea.MouseButtonNone, Action: tea.MouseActionMotion})
	if m.ctx.sel != 3 || m.ctx.items[3].key != "d" {
		t.Fatalf("hover sel = %d", m.ctx.sel)
	}
	// Moving off the items keeps the last highlight.
	m.Update(tea.MouseMsg{X: 0, Y: 0, Button: tea.MouseButtonNone, Action: tea.MouseActionMotion})
	if m.ctx.sel != 3 {
		t.Fatalf("sel moved off-menu to %d", m.ctx.sel)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.hover {
		t.Fatal("closing the menu should go back to motion-while-pressed")
	}
}

func TestContextMenuListTab(t *testing.T) {
	m := ctxTestModel(t)
	m.setTab(tabList)
	l := m.layout()
	si := rowIndex(m, func(r row) bool { return !r.group })
	rightClick(m, 5, l.listTop+si-m.scroll)
	for _, it := range m.ctx.items {
		if it.key == "n" {
			t.Fatalf("List tab snippet menu offers a new group: %+v", m.ctx.items)
		}
	}
	m.onKey(tea.KeyMsg{Type: tea.KeyEsc})
	rightClick(m, 5, l.listTop+len(m.rows))
	if len(m.ctx.items) != 1 || m.ctx.items[0].key != "a" {
		t.Fatalf("List tab empty-space menu = %+v", m.ctx.items)
	}
}
