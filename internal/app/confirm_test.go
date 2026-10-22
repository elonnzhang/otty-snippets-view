package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// openConfirm selects the test snippet and presses d.
func openConfirm(t *testing.T) *model {
	t.Helper()
	m := ctxTestModel(t)
	m.w, m.h = 70, 20
	m.setTab(tabList)
	m.sel = rowIndex(m, func(r row) bool { return !r.group })
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if m.mode != modeConfirm {
		t.Fatalf("d did not ask: mode %v", m.mode)
	}
	return m
}

func TestDeleteConfirmDialog(t *testing.T) {
	m := openConfirm(t)
	x, y, w, body := m.confirmBox()
	if x <= 0 || y <= 1 || x+w > m.w || y+confirmHeight(body) > m.h-hintRows {
		t.Fatalf("box off the safe area: x=%d y=%d w=%d h=%d", x, y, w, confirmHeight(body))
	}
	lines := strings.Split(ansi.Strip(m.View()), "\n")
	if !strings.Contains(lines[y], "┌") || !strings.Contains(lines[y+1], "Delete snippet?") {
		t.Fatalf("box top rows: %q / %q", lines[y], lines[y+1])
	}
	if !strings.Contains(lines[y+2], `"[db]-ping"`) || !strings.Contains(lines[y+3], "trash/") {
		t.Fatalf("box body: %q / %q", lines[y+2], lines[y+3])
	}
	for i, l := range lines {
		if ansi.StringWidth(l) != m.w {
			t.Fatalf("line %d width %d, want %d", i, ansi.StringWidth(l), m.w)
		}
	}
	// The question is in the box, no longer on the status line.
	if s := m.status(); s != "" {
		t.Fatalf("status line still shows %q", s)
	}

	// A click inside the box does nothing; one outside cancels.
	click(m, x+2, y+2)
	if m.mode != modeConfirm {
		t.Fatalf("a click inside the box closed it: mode %v", m.mode)
	}
	click(m, 0, 1)
	if m.mode != modeNormal || m.findSnip(ctxSnipID) == nil {
		t.Fatalf("a click outside did not just cancel: mode %v", m.mode)
	}
}

func TestDeleteConfirmKeys(t *testing.T) {
	m := openConfirm(t)
	m.onKey(tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modeNormal || m.findSnip(ctxSnipID) == nil {
		t.Fatal("Enter deleted the snippet; only y should")
	}
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if m.mode != modeNormal || m.findSnip(ctxSnipID) != nil {
		t.Fatal("y did not delete the snippet")
	}
}

func TestDeleteConfirmHintClick(t *testing.T) {
	m := openConfirm(t)
	click(m, hintColumn(m, "y"), m.h-hintRows)
	if m.mode != modeNormal || m.findSnip(ctxSnipID) != nil {
		t.Fatal("clicking the y hint did not delete")
	}
}

func TestDeleteGroupConfirmWraps(t *testing.T) {
	m := ctxTestModel(t)
	m.w, m.h = 40, 20
	gi := rowIndex(m, func(r row) bool { return r.group && r.gid != "" })
	m.meta.find(m.rows[gi].gid).Name = strings.Repeat("long", 15)
	m.sel = gi
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	_, _, w, body := m.confirmBox()
	if len(body) < 3 {
		t.Fatalf("a 60-char name in a %d-wide box did not wrap: %q", w, body)
	}
	for i, l := range strings.Split(ansi.Strip(m.View()), "\n") {
		if ansi.StringWidth(l) != m.w {
			t.Fatalf("line %d width %d, want %d", i, ansi.StringWidth(l), m.w)
		}
	}
}

func TestDeleteConfirmButtons(t *testing.T) {
	// Cancel button.
	m := openConfirm(t)
	x, y, _, body := m.confirmBox()
	row := y + confirmHeight(body) - 2
	plain := strings.Split(ansi.Strip(m.View()), "\n")[row]
	if !strings.Contains(plain, "y delete") || !strings.Contains(plain, "other key cancel") {
		t.Fatalf("button row: %q", plain)
	}
	cancelCol := strings.Index(plain, "other key")
	click(m, cancelCol+2, row)
	if m.mode != modeNormal || m.findSnip(ctxSnipID) == nil {
		t.Fatalf("cancel button: mode %v", m.mode)
	}
	// Delete button.
	m = openConfirm(t)
	click(m, x+1+dialogPad+2, row)
	if m.mode != modeNormal || m.findSnip(ctxSnipID) != nil {
		t.Fatal("delete button did not delete")
	}
	// The gap between the buttons does nothing.
	m = openConfirm(t)
	click(m, x+1+dialogPad+len("y delete")+1, row)
	if m.mode != modeConfirm {
		t.Fatalf("click in the gap closed the box: mode %v", m.mode)
	}
}
