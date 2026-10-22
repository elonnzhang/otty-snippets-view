package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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

func newForm(s Snippet, gid string, w, h int) *form {
	f := &form{orig: s, gid: gid, name: newTextInput(s.Name), alias: newTextInput(s.Alias)}
	f.alias.Blur()
	f.text = textarea.New()
	f.text.ShowLineNumbers = false
	f.text.Prompt = ""
	f.text.CharLimit = 0
	f.text.SetValue(s.Text)
	f.resize(w, h)
	return f
}

func (f *form) resize(w, h int) {
	f.name.Width, f.alias.Width = max(4, w-8), max(4, w-8)
	f.text.SetWidth(max(4, w))
	f.text.SetHeight(max(2, h-formTextTop-hintRows))
}

// Screen rows of the form, below the tab bar.
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

// focusAt maps a click on screen row y to a field.
func (f *form) focusAt(y int) {
	switch {
	case y == formName:
		f.setFocus(0)
	case y == formAlias:
		f.setFocus(1)
	case y >= formLabel:
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
