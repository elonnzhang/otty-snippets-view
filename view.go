package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// ANSI palette indices so the view follows the user's Otty theme.
var (
	stDim     = lipgloss.NewStyle().Faint(true)
	stBold    = lipgloss.NewStyle().Bold(true)
	stGroup   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
	stTag     = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	stSel     = lipgloss.NewStyle().Reverse(true)
	stDrop    = lipgloss.NewStyle().Background(lipgloss.Color("2")).Foreground(lipgloss.Color("0"))
	stOK      = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	stErr     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	stLabel   = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	stFocused = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	stTabOn   = lipgloss.NewStyle().Bold(true).Reverse(true)
	stTabOff  = lipgloss.NewStyle().Faint(true)
	stKey     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
	stHandle  = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
)

// fitw truncates s (which may contain ANSI) to w cells and pads it to exactly w.
func fitw(s string, w int) string {
	s = ansi.Truncate(s, w, "…")
	if pad := w - ansi.StringWidth(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

func oneLine(s string) string { return strings.Join(strings.Split(s, "\n"), " ⏎ ") }

// ---- tab bar -----------------------------------------------------------------

const tabPad = 1 // spaces either side of a tab label

// tabSpans returns the [start, end) columns of each tab label.
func tabSpans() [][2]int {
	var out [][2]int
	x := 1
	for _, n := range tabNames() {
		w := ansi.StringWidth(n) + 2*tabPad
		out = append(out, [2]int{x, x + w})
		x += w + 1
	}
	return out
}

func tabAt(x int) (tab, bool) {
	for i, s := range tabSpans() {
		if x >= s[0] && x < s[1] {
			return tab(i), true
		}
	}
	return 0, false
}

func (m *model) tabBar() string {
	var b strings.Builder
	b.WriteString(" ")
	pad := strings.Repeat(" ", tabPad)
	for i, n := range tabNames() {
		st := stTabOff
		if tab(i) == m.tab {
			st = stTabOn
		}
		b.WriteString(st.Render(pad + n + pad))
		b.WriteByte(' ')
	}
	right := fmt.Sprintf(tr("count"), len(m.snips), len(m.meta.Groups))
	switch {
	case m.mode == modeSearch:
		right = "/" + m.query + "▏"
	case m.query != "":
		right = "/" + m.query
	}
	left := b.String()
	gap := m.w - ansi.StringWidth(left) - ansi.StringWidth(right) - 1
	if gap < 1 {
		return left
	}
	return left + strings.Repeat(" ", gap) + stDim.Render(right)
}

// ---- whole screen -----------------------------------------------------------

func (m *model) View() string {
	if m.w == 0 || m.h == 0 {
		return ""
	}
	l := m.layout()
	lines := make([]string, 0, m.h)
	lines = append(lines, m.tabBar())
	body := m.listLines(l.listH)
	for len(body) < l.listH {
		body = append(body, "")
	}
	lines = append(lines, body...)
	if l.divY >= 0 {
		lines = append(lines, m.divider())
		det := m.detail(l.detH)
		for len(det) < l.detH {
			det = append(det, "")
		}
		lines = append(lines, det...)
	}
	if s := m.status(); s != "" {
		lines[len(lines)-1] = s
	}
	lines = append(lines, m.hints()...)
	for i := range lines {
		lines[i] = fitw(lines[i], m.w)
	}
	if len(lines) > m.h {
		lines = lines[:m.h]
	}
	if m.mode == modeContext {
		m.drawCtx(lines)
	}
	if m.mode == modePrompt {
		m.drawDialog(lines)
	}
	if m.mode == modeForm {
		m.drawForm(lines)
	}
	return strings.Join(lines, "\n")
}

// divider is the detail area's top border; dragging it resizes the area.
func (m *model) divider() string {
	if d := m.drag; d != nil && d.active {
		target := ""
		if d.hover >= 0 && d.hover < len(m.rows) {
			target = m.meta.groupName(m.rows[d.hover].gid)
		}
		return stOK.Render(fitw(fmt.Sprintf(tr("drag.target"), d.name, target), m.w))
	}
	st := stDim
	if m.resizing {
		st = stHandle
	}
	label, handle := tr("divider"), "═══"
	mid := max(0, (m.w-ansi.StringWidth(label)-len(handle))/2)
	rest := max(0, m.w-mid-ansi.StringWidth(label)-len(handle))
	return st.Render(strings.Repeat("─", mid)+label) + stHandle.Render(handle) + st.Render(strings.Repeat("─", rest))
}

// ---- list -------------------------------------------------------------------

func (m *model) listLines(lh int) []string {
	if m.mode == modeMenu {
		var title, cur string
		if m.menuFor != nil {
			title, cur = fmt.Sprintf(tr("menu.title"), m.menuFor.Name), m.meta.groupOf(m.menuFor.ID)
		} else if g := m.meta.find(m.menuGroup); g != nil {
			title, cur = fmt.Sprintf(tr("menu.group.title"), g.Name), g.Parent
		}
		out := []string{stBold.Render(title)}
		first, n := m.menuWindow()
		for j := first; j < min(len(m.menuItems), first+n); j++ {
			it := m.menuItems[j]
			mark := "  "
			switch {
			case it.gid == cur && it.gid != "+":
				mark = "✓ "
			case it.gid == "-" || it.gid == "+":
			case m.menuFor == nil && m.meta.isAncestor(it.gid, m.menuGroup):
				mark = "✗ "
			}
			line := fitw(mark+it.label, m.w)
			if j == m.menuSel {
				line = stSel.Render(line)
			}
			out = append(out, line)
		}
		return out
	}
	if len(m.rows) == 0 {
		if m.query != "" {
			return []string{stDim.Render("  " + fmt.Sprintf(tr("empty.query"), m.query))}
		}
		return []string{stDim.Render("  " + tr("empty.all"))}
	}
	var out []string
	for i := m.scroll; i < min(len(m.rows), m.scroll+lh); i++ {
		out = append(out, m.rowLine(i))
	}
	return out
}

// shortName drops a "[group]-" prefix when the group is already shown next to it.
func (m *model) shortName(r row) string {
	if g, rest, ok := splitPrefix(r.snip.Name); ok && rest != "" && strings.EqualFold(g, m.meta.groupName(r.gid)) {
		return rest
	}
	return r.snip.Name
}

func (m *model) rowLine(i int) string {
	r := m.rows[i]
	var line string
	switch {
	case r.group:
		arrow := "▾"
		if r.folded {
			arrow = "▸"
		}
		line = strings.Repeat("  ", r.depth) + m.paint(r, fitw(fmt.Sprintf("%s %s  (%d)", arrow, r.name, r.count), m.w-2*r.depth), stGroup)
	case m.tab == tabGroup:
		line = fitw(strings.Repeat("  ", r.depth+1)+m.paint(r, m.shortName(r), lipgloss.NewStyle())+stDim.Render("  "+oneLine(r.snip.Text)), m.w)
	default:
		tag := ""
		if r.gid != "" {
			tag = stTag.Render(" " + m.meta.groupName(r.gid))
		}
		line = fitw(" "+m.paint(r, m.shortName(r), lipgloss.NewStyle())+tag+stDim.Render("  "+oneLine(r.snip.Text)), m.w)
	}
	if d := m.drag; d != nil && d.active && d.hover == i {
		return stDrop.Render(ansi.Strip(line))
	}
	if i == m.sel {
		if c := m.itemColor(r); c != "" {
			return stSel.Foreground(colorValue(c)).Render(ansi.Strip(line))
		}
		return stSel.Render(ansi.Strip(line))
	}
	return line
}
