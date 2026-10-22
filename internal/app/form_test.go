package app

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// openForm opens the snippet form on a new snippet with body text.
func openForm(t *testing.T, text string) *model {
	t.Helper()
	m := ctxTestModel(t)
	m.w, m.h = 90, 30
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m.onKey(tea.KeyMsg{Type: tea.KeyTab})
	m.onKey(tea.KeyMsg{Type: tea.KeyTab})
	if text != "" {
		m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)})
	}
	return m
}

// formTop is the screen row of the form's top border.
func formTop(m *model) int {
	_, y, _, _ := m.formBox()
	return y
}

func TestLongTextFitsTheBox(t *testing.T) {
	var lines []string
	for i := 0; i < 60; i++ {
		lines = append(lines, fmt.Sprintf("line %02d", i))
	}
	m := openForm(t, strings.Join(lines, "\n"))
	x, y, w, h := m.formBox()
	if w != formW {
		t.Fatalf("box width %d, want %d", w, formW)
	}
	if x+w > m.w || y < 1 || y+h > m.h-hintRows {
		t.Fatalf("box off the safe area: x=%d y=%d w=%d h=%d", x, y, w, h)
	}
	if m.form.rows >= 60 {
		t.Fatalf("body grew to %d rows for 60 lines; it should scroll instead", m.form.rows)
	}
	if m.form.total() != 60 {
		t.Fatalf("total rows = %d, want 60", m.form.total())
	}
	// The caret is at the end of the text, so the window opens there and only
	// the last body rows are drawn, with a mark that more text is above.
	plain := strings.Split(ansi.Strip(m.View()), "\n")
	last := plain[y+formTextTop+m.form.rows-1]
	if !strings.Contains(last, "line 59") {
		t.Fatalf("last body row does not show the end of the text: %q", last)
	}
	if first := plain[y+formTextTop]; !strings.Contains(first, "↑") {
		t.Fatalf("no scroll mark on the first body row: %q", first)
	}
}

func TestClickPlacesCaret(t *testing.T) {
	m := openForm(t, "one\ntwo\nthree")
	x, y, _, _ := m.formBox()
	m.form.scroll = 0 // the three lines fit; nothing to scroll
	// Click on the second row, two cells in.
	click(m, x+1+dialogPad+2, y+formTextTop+1)
	li := m.form.text.LineInfo()
	if m.form.text.Line() != 1 || li.RowOffset != 0 || li.ColumnOffset != 2 {
		t.Fatalf("caret at line %d row %d col %d, want 1/0/2", m.form.text.Line(), li.RowOffset, li.ColumnOffset)
	}
	if m.form.focus != 2 {
		t.Fatalf("click on the body did not focus the text: focus %d", m.form.focus)
	}
	// Typing goes in at the clicked spot.
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})
	if got := m.form.text.Value(); got != "one\ntwXo\nthree" {
		t.Fatalf("text after typing = %q", got)
	}
	// Past the end of a line the caret goes to its end.
	click(m, x+1+dialogPad+30, y+formTextTop+2)
	if li := m.form.text.LineInfo(); m.form.text.Line() != 2 || li.ColumnOffset != len("three") {
		t.Fatalf("caret at line %d col %d, want 2/%d", m.form.text.Line(), li.ColumnOffset, len("three"))
	}
}

func TestClickOnWrappedLine(t *testing.T) {
	// A line far longer than the box wraps; the click must land on the
	// displayed row, not just the first one.
	long := strings.Repeat("x", 200)
	m := openForm(t, long+"\nshort")
	m.form.scroll = 0
	if m.form.total() < 3 {
		t.Fatalf("the long line did not wrap: heights %v", m.form.heights)
	}
	x, y, _, _ := m.formBox()
	click(m, x+1+dialogPad, y+formTextTop+1) // second display row of the long line
	li := m.form.text.LineInfo()
	if m.form.text.Line() != 0 || li.RowOffset != 1 {
		t.Fatalf("caret at line %d row %d, want 0/1", m.form.text.Line(), li.RowOffset)
	}
	click(m, x+1+dialogPad, y+formTextTop+m.form.heights[0]) // first row of "short"
	if m.form.text.Line() != 1 {
		t.Fatalf("caret at line %d, want 1", m.form.text.Line())
	}
}

func TestClickInScrolledBody(t *testing.T) {
	var lines []string
	for i := 0; i < 40; i++ {
		lines = append(lines, fmt.Sprintf("row %02d", i))
	}
	m := openForm(t, strings.Join(lines, "\n"))
	if m.form.scroll == 0 {
		t.Fatalf("a 40-line text should open scrolled to the caret")
	}
	// Scroll to the top with the wheel, then click the top body row.
	top := formTop(m)
	x, _, _, _ := m.formBox()
	for m.form.scroll > 0 {
		m.onMouse(tea.MouseMsg{X: x + 3, Y: top + formTextTop, Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress})
	}
	if m.form.scroll != 0 {
		t.Fatalf("the wheel did not reach the top: scroll %d", m.form.scroll)
	}
	click(m, x+1+dialogPad+5, top+formTextTop+2)
	if m.form.text.Line() != 2 {
		t.Fatalf("click on row 2 of the window put the caret on line %d", m.form.text.Line())
	}
	if li := m.form.text.LineInfo(); li.ColumnOffset != 5 {
		t.Fatalf("caret column %d, want 5", li.ColumnOffset)
	}
	// The view follows: the clicked row is still on screen after the move.
	if row := m.form.cursorRow(); row < m.form.scroll || row >= m.form.scroll+m.form.rows {
		t.Fatalf("caret row %d left the window [%d,%d)", row, m.form.scroll, m.form.scroll+m.form.rows)
	}
}

func TestTypingFollowsCaret(t *testing.T) {
	m := openForm(t, "")
	if m.form.rows != 3 {
		t.Fatalf("an empty body got %d rows, want 3", m.form.rows)
	}
	for i := 0; i < 20; i++ {
		m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("line\n")})
	}
	if m.form.total() < m.form.rows {
		t.Fatalf("20 lines should not fit in %d rows", m.form.rows)
	}
	row := m.form.cursorRow()
	if row < m.form.scroll || row >= m.form.scroll+m.form.rows {
		t.Fatalf("caret row %d is off the window [%d,%d)", row, m.form.scroll, m.form.scroll+m.form.rows)
	}
	plain := strings.Split(ansi.Strip(m.View()), "\n")
	y := formTop(m)
	// The text ends with a newline, so the caret is on an empty last row: the
	// line typed just before it is the one above.
	last := plain[y+formTextTop+m.form.rows-2]
	if !strings.Contains(last, "line") {
		t.Fatalf("the newest line is not visible: %q", last)
	}
}

func TestClickInNameAndAlias(t *testing.T) {
	m := ctxTestModel(t)
	m.w, m.h = 90, 30
	s := Snippet{Name: "alpha", Alias: "beta", Text: "x"}
	m.form, m.mode = newForm(s, ""), modeForm
	m.formLayout()
	x, y, _, _ := m.formBox()
	label := m.formLabelWidth()
	// Click three cells into the name, then two into the alias.
	click(m, x+1+dialogPad+label+3, y+formName)
	if m.form.focus != 0 || m.form.name.Position() != 3 {
		t.Fatalf("name: focus %d pos %d, want 0/3", m.form.focus, m.form.name.Position())
	}
	click(m, x+1+dialogPad+label+2, y+formAlias)
	if m.form.focus != 1 || m.form.alias.Position() != 2 {
		t.Fatalf("alias: focus %d pos %d, want 1/2", m.form.focus, m.form.alias.Position())
	}
	// Typing lands at the clicked spot.
	m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})
	if got := m.form.alias.Value(); got != "beXta" {
		t.Fatalf("alias after typing = %q", got)
	}
	// Past the end of a field the caret goes to its end.
	click(m, x+1+dialogPad+label+40, y+formName)
	if m.form.focus != 0 || m.form.name.Position() != len("alpha") {
		t.Fatalf("name: focus %d pos %d, want 0/%d", m.form.focus, m.form.name.Position(), len("alpha"))
	}
	// Clicking the label focuses the field but leaves the caret where it was.
	namePos, aliasPos := m.form.name.Position(), m.form.alias.Position()
	click(m, x+2, y+formName) // the "Name" label, left of the field
	if m.form.focus != 0 || m.form.name.Position() != namePos || m.form.alias.Position() != aliasPos {
		t.Fatalf("label click: focus %d, pos %d/%d, want 0/%d/%d",
			m.form.focus, m.form.name.Position(), m.form.alias.Position(), namePos, aliasPos)
	}
	// A long name scrolls inside its field; a click still lands on the
	// character that is displayed there.
	long := strings.Repeat("abcdefghij", 10)
	m.form.name.SetValue(long)
	m.formLayout()
	click(m, x+1+dialogPad+label+5, y+formName)
	pos := m.form.name.Position()
	if pos < 1 || pos > len(long) {
		t.Fatalf("caret pos %d out of range for a %d-rune name", pos, len(long))
	}
}
