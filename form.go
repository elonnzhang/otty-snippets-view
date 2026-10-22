package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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
}

func newForm(s Snippet, gid string) *form {
	f := &form{orig: s, gid: gid, name: newTextInput(s.Name), alias: newTextInput(s.Alias)}
	f.alias.Blur()
	f.text = textarea.New()
	f.text.ShowLineNumbers = false
	f.text.Prompt = ""
	f.text.CharLimit = 0
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
// the form dialog's top border.
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

// formBox is where the snippet form sits: centred, as wide as the terminal
// allows and as tall as the body needs to keep the cursor in view. It stays
// below the tab bar and above the hint rows when the screen is that small.
func (m *model) formBox() (x, y, w, h int) {
	w = max(16, min(formW, m.w-2))
	inner := w - 2
	textRows := m.formTextRows(inner)
	// top border, title, name, alias, text label, body, bottom border
	h = formTextTop + textRows + 1
	x = max(0, (m.w-w)/2)
	y = max(0, (m.h-h)/2-1)
	if bottom := m.h - hintRows; y+h > bottom {
		y = max(0, bottom-h)
	}
	return x, y, w, h
}

// formTextRows is how many body lines fit: as many as the snippet has, three
// at least, and never more than the screen leaves between the tab bar and the
// hint rows.
func (m *model) formTextRows(inner int) int {
	want := max(3, strings.Count(m.form.text.Value(), "\n")+1)
	room := m.h - 1 - hintRows - formTextTop - 1
	if room < 1 {
		return 1 // the textarea needs at least one row to exist
	}
	return min(want, room)
}

// formLayout sizes the inputs and the textarea for the box they are drawn in.
func (m *model) formLayout() {
	_, _, w, _ := m.formBox()
	inner := w - 2
	f := m.form
	label := max(ansi.StringWidth(tr("form.name")), ansi.StringWidth(tr("form.alias")))
	f.name.Width = max(4, inner-2*dialogPad-label-1) // 1: the cursor cell
	f.alias.Width = f.name.Width
	f.text.SetWidth(max(4, inner-2*dialogPad))
	f.text.SetHeight(m.formTextRows(inner))
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
