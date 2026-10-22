package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// ---- right-click menu ---------------------------------------------------------
//
// Every item is a shortcut that already exists: picking one runs that key on
// the row that was right-clicked, so the menu cannot drift from the keyboard.

type ctxItem struct {
	key, label string
	do         func() // set: run this instead of replaying key
}

type ctxMenu struct {
	x, y  int // top-left corner of the box on screen
	items []ctxItem
	sel   int
}

// openCtx opens the menu for row r (nil: the empty part of the list) at x, y.
func (m *model) openCtx(r *row, x, y int) {
	var items []ctxItem
	switch {
	case r == nil:
		items = []ctxItem{{key: "n", label: tr("ctx.newgroup")}, {key: "a", label: tr("ctx.newsnip")}}
	case !r.group:
		items = []ctxItem{{key: "o", label: tr("ctx.fill")}, {key: "e", label: tr("ctx.edit")},
			{key: "m", label: tr("ctx.move")}, {key: "c", label: tr("ctx.color")}, {key: "d", label: tr("ctx.delete")},
			{key: "a", label: tr("ctx.newsnip")}, {key: "n", label: tr("ctx.newgroup")}}
	case r.gid != "":
		items = []ctxItem{{key: "M", label: tr("ctx.subgroup")}, {key: "r", label: tr("ctx.rename")},
			{key: "m", label: tr("ctx.move")}, {key: "c", label: tr("ctx.color")}, {key: "d", label: tr("ctx.delete")},
			{key: "a", label: tr("ctx.newsnip")}, {key: "n", label: tr("ctx.newgroup")}}
	default: // Ungrouped can't be renamed, moved or deleted
		items = []ctxItem{{key: "n", label: tr("ctx.newgroup")}, {key: "a", label: tr("ctx.newsnip")}}
	}
	if m.tab == tabList {
		// The flat list shows no groups, so creating one there makes no sense.
		kept := items[:0]
		for _, it := range items {
			if it.key != "n" {
				kept = append(kept, it)
			}
		}
		items = kept
	}
	m.ctx = &ctxMenu{x: x, y: y, items: items}
	m.mode = modeContext
}

// run closes the menu and replays the item's key in normal mode.
func (m *model) runCtx(i int) tea.Cmd {
	c := m.ctx
	m.ctx, m.mode = nil, modeNormal
	if c == nil || i < 0 || i >= len(c.items) {
		return nil
	}
	if do := c.items[i].do; do != nil {
		do()
		return nil
	}
	return m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(c.items[i].key)})
}

func (m *model) closeCtx() { m.ctx, m.mode = nil, modeNormal }

func (m *model) keyCtx(k string) tea.Cmd {
	c := m.ctx
	switch k {
	case "up", "k":
		c.sel = (c.sel + len(c.items) - 1) % len(c.items)
	case "down", "j":
		c.sel = (c.sel + 1) % len(c.items)
	case "enter", " ":
		return m.runCtx(c.sel)
	case "esc", "q":
		m.closeCtx()
	default:
		// The shortcut shown next to an item picks it directly.
		for i, it := range c.items {
			if it.key == k {
				return m.runCtx(i)
			}
		}
	}
	return nil
}

func (m *model) mouseCtx(e tea.MouseMsg) tea.Cmd {
	x, y, w, h := m.ctxBox()
	inside := e.X >= x && e.X < x+w && e.Y > y && e.Y < y+h-1
	switch {
	case e.Button == tea.MouseButtonWheelUp:
		return m.keyCtx("up")
	case e.Button == tea.MouseButtonWheelDown:
		return m.keyCtx("down")
	case e.Action == tea.MouseActionMotion:
		// The highlight follows the pointer; off the items it stays put.
		if inside {
			m.ctx.sel = e.Y - y - 1
		}
		return nil
	case e.Action != tea.MouseActionPress:
		return nil
	}
	switch {
	case inside && e.Button == tea.MouseButtonLeft:
		return m.runCtx(e.Y - y - 1)
	case inside:
		return nil
	}
	// A click outside closes the menu; a right-click opens one where it landed.
	m.closeCtx()
	if e.Button == tea.MouseButtonRight {
		return m.onMouse(e)
	}
	return nil
}

// ctxBox is where the menu is drawn: next to the click, kept on screen and
// above the hint rows.
func (m *model) ctxBox() (x, y, w, h int) {
	c := m.ctx
	for _, it := range c.items {
		w = max(w, ansi.StringWidth(it.label))
	}
	w += 2 + 2 + 3 // borders, padding, key column
	h = len(c.items) + 2
	x = max(0, min(c.x, m.w-w))
	y = c.y + 1 // below the clicked row, which stays readable
	if bottom := m.h - hintRows; y+h > bottom {
		y = max(0, bottom-h)
	}
	return x, y, min(w, m.w), h
}

func (m *model) drawCtx(lines []string) {
	c := m.ctx
	x, y, w, h := m.ctxBox()
	inner := w - 2
	box := make([]string, 0, h)
	box = append(box, stLabel.Render("┌"+strings.Repeat("─", inner)+"┐"))
	for i, it := range c.items {
		label := fitw(" "+it.label, inner-3)
		key := fitw(" "+it.key+" ", 3)
		cell := label + stDim.Render(key)
		if i == c.sel {
			cell = stSel.Render(label + key)
		}
		box = append(box, stLabel.Render("│")+cell+stLabel.Render("│"))
	}
	box = append(box, stLabel.Render("└"+strings.Repeat("─", inner)+"┘"))
	for i, b := range box {
		row := y + i
		if row < 0 || row >= len(lines) {
			continue
		}
		line := lines[row]
		lines[row] = ansi.Truncate(line, x, "") + b + ansi.TruncateLeft(line, x+w, "")
	}
}
