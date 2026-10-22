package main

import (
	"github.com/charmbracelet/lipgloss"
)

// ---- item colors ----------------------------------------------------------------
//
// Snippets and groups can carry a color. It lives in groups.json (snippet colors
// keyed by id, group colors on the group), never in Otty's recipe files. Named
// colors map to the terminal's ANSI palette so they follow the Otty theme; a
// hand-edited "#rrggbb" also works.

// itemColors is the picker order; the empty name clears the color.
var itemColors = []struct{ name, ansi string }{
	{"red", "1"}, {"green", "2"}, {"yellow", "3"}, {"blue", "4"}, {"magenta", "5"}, {"cyan", "6"}, {"", ""},
}

func colorValue(name string) lipgloss.Color {
	for _, c := range itemColors {
		if c.name == name {
			return lipgloss.Color(c.ansi)
		}
	}
	return lipgloss.Color(name) // "#rrggbb" or a palette index
}

// itemColor is the color of a snippet or group row; "" when it has none.
func (m *model) itemColor(r row) string {
	if r.group {
		if g := m.meta.find(r.gid); g != nil {
			return g.Color
		}
		return ""
	}
	return m.meta.Colors[r.snip.ID]
}

// setItemColor stores c ("" clears it) on the snippet or group of row r.
func (m *model) setItemColor(r row, c string) {
	if r.group {
		g := m.meta.find(r.gid)
		if g == nil {
			return
		}
		g.Color = c
	} else {
		if c == "" {
			delete(m.meta.Colors, r.snip.ID)
		} else {
			if m.meta.Colors == nil {
				m.meta.Colors = map[string]string{}
			}
			m.meta.Colors[r.snip.ID] = c
		}
	}
	m.persist()
}

// paint renders s in the row's color, or with st when the row has none.
func (m *model) paint(r row, s string, st lipgloss.Style) string {
	if c := m.itemColor(r); c != "" {
		return st.Foreground(colorValue(c)).Render(s)
	}
	return st.Render(s)
}

// colorable: Ungrouped is a fixed bucket, not an item of its own.
func colorable(r *row) bool { return r != nil && (!r.group || r.gid != "") }

// openColorMenu shows the color picker for row i at x, y.
func (m *model) openColorMenu(i, x, y int) {
	if i < 0 || i >= len(m.rows) || !colorable(&m.rows[i]) {
		m.say(tr("color.none"))
		return
	}
	r := m.rows[i]
	cur := m.itemColor(r)
	var items []ctxItem
	sel := 0
	for j, c := range itemColors {
		c := c
		label := tr("color.clear")
		if c.name != "" {
			label = lipgloss.NewStyle().Foreground(colorValue(c.name)).Render("● ") + tr("color."+c.name)
		}
		if c.name == cur {
			sel = j
		}
		key := string(rune('1' + j))
		if c.name == "" {
			key = "0"
		}
		items = append(items, ctxItem{key: key, label: label, do: func() { m.setItemColor(r, c.name) }})
	}
	m.ctx = &ctxMenu{x: x, y: y, items: items, sel: sel}
	m.mode = modeContext
}
