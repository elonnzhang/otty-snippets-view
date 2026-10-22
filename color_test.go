package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestItemColors(t *testing.T) {
	m := ctxTestModel(t)
	gi := rowIndex(m, func(r row) bool { return r.group && r.gid != "" })
	si := rowIndex(m, func(r row) bool { return !r.group })
	gid := m.rows[gi].gid

	// Snippet: c opens the picker, 1 picks red.
	m.sel = si
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if m.mode != modeContext || len(m.ctx.items) != len(itemColors) {
		t.Fatalf("picker not open: mode %v", m.mode)
	}
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	if m.meta.Colors[ctxSnipID] != "red" || m.mode != modeNormal {
		t.Fatalf("snippet color = %q", m.meta.Colors[ctxSnipID])
	}

	// Group: via the right-click menu, then green.
	l := m.layout()
	rightClick(m, 5, l.listTop+gi-m.scroll)
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	if m.meta.find(gid).Color != "green" {
		t.Fatalf("group color = %q", m.meta.find(gid).Color)
	}

	// Both survive a reload of groups.json and show up in the rows.
	m2 := newModel()
	m2.w, m2.h = 60, 20
	m2.setTab(tabGroup)
	if m2.meta.Colors[ctxSnipID] != "red" || m2.meta.find(gid).Color != "green" {
		t.Fatal("colors not persisted")
	}
	lipgloss.SetColorProfile(termenv.ANSI)
	defer lipgloss.SetColorProfile(termenv.Ascii)
	m2.sel = -1 // no row reversed, so the colors are visible
	v := m2.View()
	if !strings.Contains(v, "31m") || !strings.Contains(v, "32") {
		t.Fatalf("red snippet / green group missing from view: %q", v)
	}

	// 0 clears; Ungrouped refuses a color.
	m.sel = si
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("0")})
	if _, ok := m.meta.Colors[ctxSnipID]; ok {
		t.Fatal("0 did not clear the color")
	}
	m.sel = rowIndex(m, func(r row) bool { return r.group && r.gid == "" })
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if m.mode == modeContext {
		t.Fatal("Ungrouped got a color picker")
	}
}
