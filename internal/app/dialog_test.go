package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// hintColumn is the middle column of a hint key in the key row.
func hintColumn(m *model, key string) int {
	for _, s := range m.hintSpans() {
		if s.key == key {
			return s.col + s.wid/2
		}
	}
	return -1
}

func TestGroupDialog(t *testing.T) {
	m := ctxTestModel(t)
	m.w, m.h = 70, 20
	m.setTab(tabGroup)
	rows := len(m.rows)

	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.mode != modePrompt {
		t.Fatalf("n did not open the dialog: mode %v", m.mode)
	}
	x, y, w := m.dialogBox()
	if x <= 0 || y <= 1 || x+w > m.w {
		t.Fatalf("dialog box off screen: x=%d y=%d w=%d", x, y, w)
	}
	plain := ansi.Strip(m.View())
	for _, want := range []string{"New group name", "┌", "└"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("dialog does not show %q", want)
		}
	}
	line := strings.Split(plain, "\n")[y+1]
	if ansi.StringWidth(line) != m.w {
		t.Fatalf("dialog row width %d, want %d", ansi.StringWidth(line), m.w)
	}
	if !strings.Contains(line, "New group name") {
		t.Fatalf("dialog label missing on row %d: %q", y+1, line)
	}

	for _, r := range "cli" {
		m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m.onKey(tea.KeyMsg{Type: tea.KeyEnter})
	if g := m.meta.findByName("cli"); g == nil {
		t.Fatal("enter did not create the group")
	}
	if m.mode != modeNormal {
		t.Fatalf("dialog stayed open after saving: mode %v", m.mode)
	}
	if len(m.rows) == rows {
		t.Fatal("the new group is not in the list")
	}

	// A duplicate name keeps the dialog open with the reason inside it.
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	for _, r := range "db" {
		m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m.onKey(tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modePrompt || m.promptMsg == "" {
		t.Fatalf("bad name closed the dialog: mode %v, msg %q", m.mode, m.promptMsg)
	}
	if !strings.Contains(ansi.Strip(m.View()), "taken") {
		t.Fatal("the dialog does not show why the name was rejected")
	}
	if m.dialogHeight() != 4 {
		t.Fatalf("dialog height with an error = %d, want 4", m.dialogHeight())
	}
	// Typing again clears the reason.
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if m.promptMsg != "" {
		t.Fatalf("typing kept the error: %q", m.promptMsg)
	}

	// A click outside the box cancels; inside, it is not a cancel.
	x, y, w = m.dialogBox()
	_ = w
	click(m, x+2, y+1)
	if m.mode != modePrompt {
		t.Fatalf("a click inside the dialog cancelled it: mode %v", m.mode)
	}
	click(m, 0, 2)
	if m.mode != modeNormal {
		t.Fatalf("a click outside the dialog did not cancel: mode %v", m.mode)
	}
	if g := m.meta.findByName("db"); g == nil {
		t.Fatal("the test group was not there to begin with")
	}
	if len(m.rows) != rows+1 {
		t.Fatalf("cancel left %d rows, want %d", len(m.rows), rows+1)
	}
}

func TestSnippetFormDialog(t *testing.T) {
	m := ctxTestModel(t)
	m.w, m.h = 80, 24
	m.setTab(tabList)
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.mode != modeForm {
		t.Fatalf("a did not open the form: mode %v", m.mode)
	}
	x, y, w, h := m.formBox()
	if y <= 1 || y+h > m.h-hintRows || x+w > m.w {
		t.Fatalf("form box overlaps the tab bar or the hints: x=%d y=%d w=%d h=%d", x, y, w, h)
	}
	line := strings.Split(ansi.Strip(m.View()), "\n")[y]
	if !strings.Contains(line, "┌") || ansi.StringWidth(line) != m.w {
		t.Fatalf("form box top row: %q", line)
	}

	// A rejected save keeps the form open and shows the reason inside it.
	m.onKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	if m.mode != modeForm || m.formMsg == "" {
		t.Fatalf("empty save closed the form: mode %v, msg %q", m.mode, m.formMsg)
	}
	if !strings.Contains(ansi.Strip(m.View()), "name can't be empty") {
		t.Fatal("the form does not show why saving was refused")
	}

	// Typing, filling it in, then saving puts the snippet in the list.
	for _, r := range "echo" {
		m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m.onKey(tea.KeyMsg{Type: tea.KeyTab})
	for _, r := range "hi" {
		m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m.onKey(tea.KeyMsg{Type: tea.KeyTab})
	for _, r := range "echo hi" {
		m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m.onKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	if m.mode != modeNormal || m.form != nil {
		t.Fatalf("save left the form open: mode %v", m.mode)
	}
	if m.findSnip(m.rows[m.sel].snip.ID) == nil || m.rows[m.sel].snip.Name != "echo" {
		t.Fatalf("the new snippet is not selected: %+v", m.cur())
	}
}

func TestHintClicks(t *testing.T) {
	m := ctxTestModel(t)
	m.w, m.h = 70, 20
	m.setTab(tabList)

	click(m, hintColumn(m, "a"), m.h-hintRows)
	if m.mode != modeForm {
		t.Fatalf("clicking the a hint did not open the form: mode %v", m.mode)
	}
	m.onKey(tea.KeyMsg{Type: tea.KeyEsc})

	// The tip row under a key works too.
	click(m, hintColumn(m, "/"), m.h-hintRows+1)
	if m.mode != modeSearch {
		t.Fatalf("clicking the / hint did not open search: mode %v", m.mode)
	}
	m.onKey(tea.KeyMsg{Type: tea.KeyEsc})

	click(m, hintColumn(m, "Tab"), m.h-hintRows)
	if m.tab != tabGroup {
		t.Fatalf("the Tab hint did not switch tabs: tab %v", m.tab)
	}

	// A form's hint rows are clickable too: Tab moves on, ⌃S saves, Esc
	// cancels. They sit outside the box but must not count as an outside click.
	m.setTab(tabList)
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if m.mode != modeForm {
		t.Fatalf("a did not open the form: mode %v", m.mode)
	}
	click(m, hintColumn(m, "Tab"), m.h-hintRows)
	if m.mode != modeForm || m.form.focus != 1 {
		t.Fatalf("the Tab hint did not move to the next field: mode %v", m.mode)
	}
	click(m, hintColumn(m, "⌃S"), m.h-hintRows)
	if m.mode != modeForm || m.formMsg == "" {
		t.Fatalf("saving an empty form closed it: mode %v, msg %q", m.mode, m.formMsg)
	}
	click(m, hintColumn(m, "Esc"), m.h-hintRows+1)
	if m.mode != modeNormal || m.form != nil {
		t.Fatalf("the Esc hint did not cancel the form: mode %v", m.mode)
	}

	// A click outside the form box, off the hint rows, still cancels it.
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	click(m, 0, 2)
	if m.mode != modeNormal || m.form != nil {
		t.Fatalf("a click outside the form did not cancel it: mode %v", m.mode)
	}
}

// Clicking ⌃S in the form saves the snippet, the same as pressing it.
func TestFormSaveByHintClick(t *testing.T) {
	m := ctxTestModel(t)
	m.w, m.h = 80, 24
	m.setTab(tabList)
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	for _, field := range []string{"echo", "hi", "echo hi"} {
		for _, r := range field {
			m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		}
		m.onKey(tea.KeyMsg{Type: tea.KeyTab})
	}
	click(m, hintColumn(m, "⌃S"), m.h-hintRows)
	if m.mode != modeNormal || m.form != nil {
		t.Fatalf("clicking ⌃S left the form open: mode %v, msg %q", m.mode, m.formMsg)
	}
	var saved bool
	for _, s := range loadSnippets() {
		saved = saved || (s.Name == "echo" && s.Alias == "hi" && s.Text == "echo hi")
	}
	if !saved {
		t.Fatal("clicking ⌃S did not write the snippet to the recipes folder")
	}
}

// The group name box has the same hint rows: ⏎ saves, Esc cancels.
func TestGroupDialogHintClicks(t *testing.T) {
	m := ctxTestModel(t)
	m.w, m.h = 70, 20
	m.setTab(tabGroup)
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	for _, r := range "cli" {
		m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	click(m, hintColumn(m, "⏎"), m.h-hintRows)
	if m.mode != modeNormal || m.meta.findByName("cli") == nil {
		t.Fatalf("clicking ⏎ did not create the group: mode %v", m.mode)
	}
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	click(m, hintColumn(m, "Esc"), m.h-hintRows)
	if m.mode != modeNormal {
		t.Fatalf("clicking Esc did not close the dialog: mode %v", m.mode)
	}
}
