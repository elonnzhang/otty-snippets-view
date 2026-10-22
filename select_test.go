package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func click(m *model, x, y int) {
	m.onMouse(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m.onMouse(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease})
}

func TestClickEmptySpaceClearsSelection(t *testing.T) {
	for _, tab := range []tab{tabList, tabGroup} {
		m := ctxTestModel(t)
		m.setTab(tab)
		l := m.layout()
		si := rowIndex(m, func(r row) bool { return !r.group })
		click(m, 5, l.listTop+si-m.scroll)
		if m.sel != si {
			t.Fatalf("tab %v: click on a row did not select it", tab)
		}

		// Below the last row, inside the list.
		click(m, 5, l.listTop+len(m.rows)-m.scroll+1)
		if m.sel != -1 {
			t.Fatalf("tab %v: click on empty space left sel = %d", tab, m.sel)
		}
		if m.detKey != "" {
			t.Fatalf("tab %v: detail still keyed to %q", tab, m.detKey)
		}
		if m.mode != modeNormal {
			t.Fatalf("tab %v: empty click changed mode to %v", tab, m.mode)
		}

		// The pointer moving over empty space is not a click.
		m.sel = si
		m.Update(tea.MouseMsg{X: 5, Y: l.listTop + len(m.rows) - m.scroll + 1, Button: tea.MouseButtonNone, Action: tea.MouseActionMotion})
		if m.sel != si {
			t.Fatalf("tab %v: pointer motion cleared the selection", tab)
		}

		// Navigation from the cleared state: down starts at the first row, up
		// at the last one.
		click(m, 5, l.listTop+len(m.rows)-m.scroll+1)
		m.onKey(tea.KeyMsg{Type: tea.KeyDown})
		if m.sel != 0 {
			t.Fatalf("tab %v: down from nothing selected sel = %d, want 0", tab, m.sel)
		}
		click(m, 5, l.listTop+len(m.rows)-m.scroll+1)
		m.onKey(tea.KeyMsg{Type: tea.KeyUp})
		if m.sel != len(m.rows)-1 {
			t.Fatalf("tab %v: up from nothing selected sel = %d, want %d", tab, m.sel, len(m.rows)-1)
		}
		click(m, 5, l.listTop+len(m.rows)-m.scroll+1)
		m.onKey(tea.KeyMsg{Type: tea.KeyPgDown})
		if m.sel != 0 {
			t.Fatalf("tab %v: pgdown from nothing selected sel = %d, want 0", tab, m.sel)
		}

		// Clicking a row again selects it.
		click(m, 5, l.listTop+len(m.rows)-m.scroll+1)
		click(m, 5, l.listTop+si-m.scroll)
		if m.sel != si {
			t.Fatalf("tab %v: click on a row after clearing sel = %d", tab, m.sel)
		}
	}
}

func TestNothingSelectedView(t *testing.T) {
	m := ctxTestModel(t)
	lang = "en"
	l := m.layout()
	click(m, 5, l.listTop+len(m.rows)-m.scroll+1)

	if !strings.Contains(m.View(), tr("detail.none")) {
		t.Fatal("detail area does not say nothing is selected")
	}
	if !strings.Contains(m.View(), "a") {
		t.Fatal("hints lost the new-snippet key")
	}

	// Row actions do nothing without a selection.
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if m.mode != modeNormal || m.form != nil {
		t.Fatalf("row action ran with nothing selected: mode %v", m.mode)
	}

	// Creating one still works.
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.mode != modeForm || m.form == nil {
		t.Fatalf("a did not open the form: mode %v", m.mode)
	}

	// A reload keeps the cleared state unless it leaves nothing to show.
	m.mode = modeNormal
	m.form = nil
	m.reload()
	if m.sel != -1 {
		t.Fatalf("reload brought the selection back: sel = %d", m.sel)
	}
}
