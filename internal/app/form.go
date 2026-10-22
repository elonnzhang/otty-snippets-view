package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func newTextInput(text string) textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.SetValue(text)
	ti.CursorEnd()
	ti.Focus()
	return ti
}

// form edits a snippet: name, alias and a multi-line body.
type form struct {
	orig  Snippet // ID "" for a new snippet
	gid   string  // group a new snippet lands in
	name  textinput.Model
	alias textinput.Model
	text  textarea.Model
	focus int // 0 name, 1 alias, 2 text

	// The body window is ours, not the textarea's: its viewport offset cannot
	// be read back, so a click could not be mapped to a text position. The
	// textarea shows all of its text in the box's window; these say which part
	// of it is on screen.
	heights []int    // display rows per logical line at the current width
	tops    []int    // display row each logical line starts at; one extra entry for the total
	lines   []string // the wrapped text, one entry per display row
	mvalue  string   // text and width heights/tops were measured at
	mwidth  int
	rows    int // body rows the dialog shows
	scroll  int // first body row on screen
}

func newForm(s Snippet, gid string) *form {
	f := &form{orig: s, gid: gid, name: newTextInput(s.Name), alias: newTextInput(s.Alias)}
	f.alias.Blur()
	f.text = textarea.New()
	f.text.ShowLineNumbers = false
	f.text.Prompt = ""
	f.text.CharLimit = 0
	f.text.MaxHeight = 0
	// No text or cursor-line styling inside the dialog; the cursor block is
	// drawn by drawForm, on the cell the cursor is at.
	f.text.FocusedStyle.CursorLine = lipgloss.NewStyle()
	f.text.FocusedStyle.Text = lipgloss.NewStyle()
	f.text.BlurredStyle.Text = lipgloss.NewStyle()
	f.text.SetValue(s.Text)
	return f
}

// Screen rows of the form's dialog, relative to its top border.
const (
	formTitle   = 1
	formName    = 2
	formAlias   = 3
	formLabel   = 4
	formTextTop = 5
)

func (f *form) setFocus(i int) {
	f.focus = (i + 3) % 3
	f.name.Blur()
	f.alias.Blur()
	f.text.Blur()
	switch f.focus {
	case 0:
		f.name.Focus()
	case 1:
		f.alias.Focus()
	default:
		f.text.Focus()
	}
}

// setFocusAt maps a click on screen row y to a field; rows are 0-based from
// the form dialog's top border. A click on the body only switches to the text;
// form.clickText puts the cursor where it landed.
func (f *form) setFocusAt(row int) {
	switch {
	case row == formName:
		f.setFocus(0)
	case row == formAlias:
		f.setFocus(1)
	case row >= formLabel:
		f.setFocus(2)
	}
}

func (f *form) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch f.focus {
	case 0:
		f.name, cmd = f.name.Update(msg)
	case 1:
		f.alias, cmd = f.alias.Update(msg)
	default:
		f.text, cmd = f.text.Update(msg)
	}
	return cmd
}

// ---- dialogs -----------------------------------------------------------------
//
// Creating or renaming a group, and creating or editing a snippet, all happen
// in a box in the middle of the screen: the one-line prompt used to sit on the
// status line, easy to miss, and the form took over the whole view.

const (
	dialogW   = 44 // the group-name box: outer width, clamped to the terminal
	dialogPad = 1
	formW     = 72 // the snippet form: outer width, clamped to the terminal
	confirmW  = 56 // the delete confirmation: outer width, clamped to the terminal
)

// dialogBox is where the group-name dialog sits: centred horizontally, a
// little above the middle so the eye lands on it. The box grows a row when it
// has an error to show.
func (m *model) dialogBox() (x, y, w int) {
	w = max(12, min(dialogW, m.w-2))
	x = max(0, (m.w-w)/2)
	y = max(0, (m.h-m.dialogHeight())/2-1)
	return x, y, w
}

func (m *model) dialogHeight() int {
	if m.promptMsg != "" {
		return 4
	}
	return 3
}

// dialogInputWidth is the room the prompt has inside the box. The label is the
// wider of the two prompts, so the input width does not jump between new and
// rename.
func (m *model) dialogInputWidth() int {
	_, _, w := m.dialogBox()
	label := max(ansi.StringWidth(tr("prompt.new")), ansi.StringWidth(tr("prompt.ren")))
	return max(4, w-2-2*dialogPad-label)
}

// confirmBox is where the delete confirmation sits: centred like the other
// dialogs, with the question as its title, what the delete touches as its
// body and the keys that answer it on the last row. The body wraps to the box
// width, so a long name still fits.
func (m *model) confirmBox() (x, y, w int, body []string) {
	w = max(16, min(confirmW, m.w-2))
	inner := w - 2 - 2*dialogPad
	for _, l := range m.confirmText() {
		for _, part := range strings.Split(ansi.Wrap(l, inner, ""), "\n") {
			body = append(body, strings.TrimRight(part, " "))
		}
	}
	h := confirmHeight(body)
	x = max(0, (m.w-w)/2)
	y = max(0, (m.h-h)/2-1)
	if bottom := m.h - hintRows; y+h > bottom {
		y = max(0, bottom-h)
	}
	return x, y, w, body
}

// confirmHeight: top border, title, body, a blank row, the keys, bottom border.
func confirmHeight(body []string) int { return 5 + len(body) }

// formBox is where the snippet form sits: centred, as wide as the terminal
// allows and as tall as the body needs. It stays below the tab bar and above
// the hint rows when the screen is that small.
func (m *model) formBox() (x, y, w, h int) {
	w = max(16, min(formW, m.w-2))
	textRows := 1
	if m.form != nil {
		textRows = max(1, m.form.rows)
	}
	// top border, title, name, alias, text label, body, bottom border
	h = formTextTop + textRows + 1
	x = max(0, (m.w-w)/2)
	y = max(0, (m.h-h)/2-1)
	if bottom := m.h - hintRows; y+h > bottom {
		y = max(0, bottom-h)
	}
	return x, y, w, h
}

// formTextRows is how many body rows the box shows: as many as the text needs,
// but never past a soft two-thirds of the screen, so the list stays visible
// around the box. Text that does not fit scrolls inside it. On a terminal too
// short for that the box takes the room it gets, and the hint rows go under it.
func (m *model) formTextRows() int {
	f := m.form
	f.measure()
	want := max(3, f.total())
	soft := (m.h-1-hintRows)*2/3 - formTextTop - 1
	room := m.h - formTextTop - 1 // the box never grows past the screen
	rows := min(want, room)
	if soft >= 3 {
		rows = min(rows, soft)
	}
	return max(1, rows)
}

// formLayout sizes the fields, measures the text at the width they got and
// sizes the body for it.
func (m *model) formLayout() {
	f := m.form
	w := max(16, min(formW, m.w-2))
	inner := w - 2
	f.name.Width = max(4, inner-2*dialogPad-m.formLabelWidth()-1) // 1: the cursor cell
	f.alias.Width = f.name.Width
	f.text.SetWidth(max(4, inner-2*dialogPad))
	f.measure()
	f.rows = m.formTextRows()
	f.clampScroll()
}

// measure wraps the text at the current width: display rows per logical line
// and the wrapped lines themselves. The textarea offers no read access to
// either, so the rows are walked on a copy and the lines come from a fresh
// textarea at the same width: it wraps exactly like the focused one, and its
// viewport is at the top with no cursor on screen.
//
// The walk is on a copy because CursorUp and CursorDown move by screen rows and
// the real cursor must stay where the user left it.
func (f *form) measure() {
	value, width := f.text.Value(), f.text.Width()
	if len(f.tops) > 0 && value == f.mvalue && width == f.mwidth {
		return
	}

	walk := f.text
	walk.CursorStart()
	for walk.Line() > 0 {
		walk.CursorUp()
	}
	n := walk.LineCount()
	heights := make([]int, n)
	tops := make([]int, n+1)
	for i := 0; i < n; i++ {
		heights[i] = max(1, walk.LineInfo().Height)
		tops[i+1] = tops[i] + heights[i]
		if i < n-1 {
			// From the end of a line CursorDown moves one logical line exactly,
			// soft wraps included.
			walk.CursorEnd()
			walk.CursorDown()
		}
	}

	c := textarea.New()
	c.ShowLineNumbers = false
	c.Prompt = ""
	c.CharLimit = 0
	c.MaxHeight = 0
	c.SetWidth(width)
	c.SetValue(value)
	c.SetHeight(tops[n])
	rows := strings.Split(c.View(), "\n")
	lines := make([]string, tops[n])
	for i := range lines {
		if i < len(rows) {
			lines[i] = strings.TrimRight(ansi.Strip(rows[i]), " ")
		}
	}

	f.heights, f.tops, f.lines = heights, tops, lines
	f.mvalue, f.mwidth = value, width
}

// total is the number of display rows the whole text takes.
func (f *form) total() int { return f.tops[len(f.tops)-1] }

// cursorRow is the display row the text cursor sits on.
func (f *form) cursorRow() int {
	f.measure()
	row := f.text.Line()
	if row >= len(f.heights) {
		row = len(f.heights) - 1
	}
	return f.tops[row] + f.text.LineInfo().RowOffset
}

// cursorSpot is where the cursor block is drawn: its display row, the cell in
// that row, and the character under it (a space past the end of a line).
func (f *form) cursorSpot() (row, cell int, ch string) {
	f.measure()
	row = f.cursorRow()
	lines := strings.Split(f.text.Value(), "\n")
	l := f.text.Line()
	if l >= len(lines) {
		return row, 0, " "
	}
	runes := []rune(lines[l])
	li := f.text.LineInfo()
	start := min(li.StartColumn, len(runes))
	end := min(start+li.ColumnOffset, len(runes))
	cell = ansi.StringWidth(string(runes[start:end]))
	if end < len(runes) {
		return row, cell, string(runes[end])
	}
	return row, cell, " "
}

func (f *form) clampScroll() {
	f.measure()
	f.scroll = max(0, min(f.scroll, max(0, f.total()-max(1, f.rows))))
}

// clickText puts the cursor on the character under a click in the body:
// viewRow is 0-based from the first visible body row, cell from the body's
// left edge.
func (f *form) clickText(viewRow, cell int) {
	f.measure()
	if viewRow < 0 || f.total() == 0 {
		return
	}
	row := min(f.scroll+viewRow, f.total()-1)
	l := 0
	for l+1 < len(f.heights) && f.tops[l+1] <= row {
		l++
	}
	f.moveTo(l, row-f.tops[l], max(0, cell))
}

// moveTo puts the cursor at `cell` cells into display row `offset` of logical
// line l. The textarea has no cursor-row setter, so this walks there the way a
// person would; CursorEnd followed by CursorDown is exactly one line even
// where the line is soft-wrapped. The cursor ends up at a rune index in the
// line: the display row follows from it, so the index is the row's start plus
// the clicked cell, not the clicked cell alone.
func (f *form) moveTo(l, offset, cell int) {
	f.text.CursorStart()
	for f.text.Line() > 0 {
		f.text.CursorUp()
	}
	f.text.CursorStart()
	for i := 0; i < l; i++ {
		f.text.CursorEnd()
		f.text.CursorDown()
	}
	for i := 0; i < offset; i++ {
		f.text.CursorDown()
	}
	start := f.text.LineInfo().StartColumn
	if all := strings.Split(f.text.Value(), "\n"); l < len(all) {
		f.text.SetCursor(start + cellIn(string([]rune(all[l])[start:]), cell))
	}
}

// formFollow grows or shrinks the body window for the text as edited and
// scrolls it so the cursor stays in view. It runs after every edit, so typing at
// the end of a long text keeps the caret visible.
func (m *model) formFollow() {
	f := m.form
	if f == nil {
		return
	}
	f.measure()
	f.rows = m.formTextRows()
	row := f.cursorRow()
	if row < f.scroll {
		f.scroll = row
	} else if row >= f.scroll+f.rows {
		f.scroll = row - f.rows + 1
	}
	f.clampScroll()
}

// clickForm focuses the field under a click and puts its cursor where the
// click landed: on a character in the name or alias, or in the body.
func (m *model) clickForm(e tea.MouseMsg) {
	x, y, _, _ := m.formBox()
	row := e.Y - y
	m.form.setFocusAt(row)
	switch row {
	case formName:
		if cell := e.X - (x + 1 + dialogPad + m.formLabelWidth()); cell >= 0 {
			m.form.name.SetCursor(fieldPos(m.form.name, cell))
		}
	case formAlias:
		if cell := e.X - (x + 1 + dialogPad + m.formLabelWidth()); cell >= 0 {
			m.form.alias.SetCursor(fieldPos(m.form.alias, cell))
		}
	default:
		if row >= formTextTop {
			m.form.clickText(row-formTextTop, e.X-(x+1+dialogPad))
			m.formFollow()
		}
	}
}

// formLabelWidth is the widest field label, so name and alias line up.
func (m *model) formLabelWidth() int {
	return max(ansi.StringWidth(tr("form.name")), ansi.StringWidth(tr("form.alias")))
}

// fieldPos maps a click `cell` cells into a one-line field to a rune index. A
// field longer than its width scrolls sideways and the textinput does not
// expose where its window starts; the window is taken to end at the caret,
// which is how it sits after typing and is one cell off at most after a few
// cursor lefts.
func fieldPos(in textinput.Model, cell int) int {
	value := []rune(in.Value())
	pos, width := in.Position(), in.Width
	start, used := pos, 0
	for start > 0 && used+ansi.StringWidth(string(value[start-1])) <= width {
		start--
		used += ansi.StringWidth(string(value[start]))
	}
	end := pos
	for end < len(value) && used+ansi.StringWidth(string(value[end])) <= width {
		used += ansi.StringWidth(string(value[end]))
		end++
	}
	return start + cellIn(string(value[start:end]), cell)
}

// cellIn maps a cell offset from the start of a one-line input's text to a
// rune index in it: the click goes before the character it landed on, and past
// the end of the text to its end.
func cellIn(value string, cell int) int {
	runes := []rune(value)
	col, used := 0, 0
	for col < len(runes) && used < cell {
		w := ansi.StringWidth(string(runes[col]))
		if used+w > cell {
			break // the click landed in the middle of a wide character
		}
		used += w
		col++
	}
	return col
}

// result validates the form and returns the snippet to save.
func (f *form) result(meta []Snippet) (Snippet, string) {
	s := f.orig
	s.Name = strings.TrimSpace(f.name.Value())
	s.Alias = strings.TrimSpace(f.alias.Value())
	s.Text = f.text.Value()
	switch {
	case s.Name == "":
		return s, tr("form.noname")
	case s.Alias == "" || strings.ContainsAny(s.Alias, " \t"):
		return s, tr("form.noalias")
	case s.Text == "":
		return s, tr("form.notext")
	}
	for _, o := range meta {
		if o.ID != s.ID && strings.EqualFold(o.Alias, s.Alias) {
			return s, fmt.Sprintf(tr("form.aliasused"), o.Name)
		}
	}
	return s, ""
}
