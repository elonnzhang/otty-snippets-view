package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
)

// ---- detail -----------------------------------------------------------------

// detail is the area below the divider. With nothing selected it holds one dim
// line rather than the last row's content.
func (m *model) detail(n int) []string {
	r := m.cur()
	if r == nil {
		// No rows at all: the list already says so, no placeholder needed.
		if n <= 0 || len(m.rows) == 0 {
			return nil
		}
		return []string{stDim.Render(tr("detail.none"))}
	}
	if r.group {
		out := []string{stBold.Render(r.name) + stDim.Render("  "+fmt.Sprintf(tr("group.count"), r.count))}
		if r.gid != "" {
			if p := m.meta.find(r.gid).Parent; p != "" {
				out = append(out, stDim.Render(fmt.Sprintf(tr("group.parent"), m.meta.path(p))))
			}
			out = append(out, stDim.Render(fmt.Sprintf(tr("group.auto"), r.name)))
		}
		return out
	}
	s := r.snip
	head := stBold.Render(s.Name) + stDim.Render("  alias: "+s.Alias+" · "+m.meta.groupName(r.gid))
	body := m.hl.highlight(s.Text, max(1, m.w))
	room := max(0, n-1)
	m.detScroll = max(0, min(m.detScroll, len(body)-room))
	end := min(len(body), m.detScroll+room)
	view := append([]string(nil), body[m.detScroll:end]...)
	if end < len(body) && len(view) > 0 {
		view[len(view)-1] = fitw(view[len(view)-1], m.w-1) + stDim.Render("↓")
	}
	return append([]string{head}, view...)
}

// ---- hints: two rows, each key above its tip ------------------------------

type hint struct{ key, tip string }

func (m *model) hintSet() []hint {
	switch m.mode {
	case modeSearch:
		return []hint{{tr("h.input"), tr("h.filter")}, {"↑↓", tr("h.select")}, {"⏎", tr("h.done")}, {"Esc", tr("h.clear")}}
	case modePrompt:
		return []hint{{"⏎", tr("h.save")}, {"Esc", tr("h.cancel")}}
	case modeMenu:
		return []hint{{"↑↓", tr("h.select")}, {"⏎/click", tr("h.move")}, {"Esc", tr("h.cancel")}}
	case modeContext:
		return []hint{{"↑↓", tr("h.select")}, {"⏎/click", tr("h.ok")}, {"Esc", tr("h.cancel")}}
	case modeConfirm:
		return []hint{{"y", tr("h.ok")}, {pick("other key", "其他键"), tr("h.cancel")}}
	case modeForm:
		return []hint{{"Tab", tr("h.next")}, {"⌃S", tr("h.save")}, {"Esc", tr("h.cancel")}}
	}
	r := m.cur()
	switch {
	case r == nil:
		if m.tab == tabGroup {
			return []hint{{"a", tr("h.new")}, {"n", tr("h.newg")}, {"Tab", tr("h.tab")}, {"/", tr("h.find")}}
		}
		return []hint{{"a", tr("h.new")}, {"Tab", tr("h.tab")}, {"/", tr("h.find")}}
	case r.group:
		h := []hint{{"⏎", tr("h.fold")}, {"n", tr("h.newg")}}
		if r.gid != "" {
			h = append(h, hint{"r", tr("h.rename")}, hint{"m", tr("h.moveg")}, hint{"c", tr("h.color")}, hint{"d", tr("h.del")}, hint{"[ ]", tr("h.sort")})
		}
		h = append(h, hint{"M", tr("h.subg")})
		return append(h, hint{"Tab", tr("h.tab")}, hint{"/", tr("h.search")})
	}
	h := []hint{{"⏎", tr("h.run")}, {"o", tr("h.fill")}, {"y", tr("h.copy")}, {"a", tr("h.new")}, {"e", tr("h.edit")},
		{"m", tr("h.move")}, {"c", tr("h.color")}, {"d", tr("h.delete")}}
	if m.tab == tabGroup {
		h = append(h, hint{"n", tr("h.newg")})
	}
	return append(h, hint{"Tab", tr("h.tab")}, hint{"/", tr("h.find")})
}

func (m *model) hints() []string {
	rows := m.hintRow(m.hintSet())
	return rows[:]
}

// status is the prompt, confirm question or a recent message, shown on the
// line just above the hints so the keys stay visible; "" when there is none.
func (m *model) status() string {
	switch {
	case m.mode == modePrompt:
		label := tr("prompt.new")
		if m.promptGroup != "" {
			label = tr("prompt.ren")
		}
		return stLabel.Render(label) + m.prompt.View()
	case m.mode == modeConfirm:
		return stErr.Render(m.confirmText())
	case m.msg != "" && time.Now().Before(m.msgUntil):
		st := stErr
		if strings.HasPrefix(m.msg, "✓") {
			st = stOK
		}
		return st.Render(m.msg)
	}
	return ""
}

// hintRow lays hints out in columns: keys on the first row, tips right below.
func (m *model) hintRow(hs []hint) [2]string {
	var keys, tips strings.Builder
	used := 0
	for _, h := range hs {
		cw := max(ansi.StringWidth(h.key), ansi.StringWidth(h.tip))
		sep := 0
		if used > 0 {
			sep = 2
		}
		if used+sep+cw > m.w {
			break
		}
		if sep > 0 {
			keys.WriteString("  ")
			tips.WriteString("  ")
		}
		keys.WriteString(stKey.Render(fitw(h.key, cw)))
		tips.WriteString(stDim.Render(fitw(h.tip, cw)))
		used += sep + cw
	}
	return [2]string{keys.String(), tips.String()}
}

func (m *model) confirmText() string {
	if m.confirmGroup != "" {
		n := 0
		for _, s := range m.snips {
			if m.meta.groupOf(s.ID) == m.confirmGroup {
				n++
			}
		}
		if kids := m.meta.descendants(m.confirmGroup); kids > 0 {
			return fmt.Sprintf(tr("group.del.tree"), m.meta.groupName(m.confirmGroup), n, kids)
		}
		return fmt.Sprintf(tr("group.del"), m.meta.groupName(m.confirmGroup), n)
	}
	name := m.confirmSnip
	if s := m.findSnip(m.confirmSnip); s != nil {
		name = s.Name
	}
	return fmt.Sprintf(tr("snip.del"), name)
}

// ---- form -------------------------------------------------------------------

func (m *model) viewForm() string {
	f := m.form
	title := tr("form.edit")
	if f.orig.ID == "" {
		title = fmt.Sprintf(tr("form.new"), m.meta.groupName(f.gid))
	}
	label := func(i int, s string) string {
		if f.focus == i {
			return stFocused.Render(s)
		}
		return stLabel.Render(s)
	}
	lines := []string{
		m.tabBar(),
		stBold.Render(title),
		label(0, tr("form.name")) + f.name.View(),
		label(1, tr("form.alias")) + f.alias.View(),
		label(2, tr("form.text")) + stDim.Render(tr("form.tiphint")),
	}
	lines = append(lines, strings.Split(f.text.View(), "\n")...)
	for len(lines) < m.h-hintRows {
		lines = append(lines, "")
	}
	lines = lines[:max(0, m.h-hintRows)]
	if s := m.status(); s != "" && len(lines) > 0 {
		lines[len(lines)-1] = s
	}
	hs := m.hintRow(m.hintSet())
	lines = append(lines, hs[0], hs[1])
	for i := range lines {
		lines[i] = fitw(lines[i], m.w)
	}
	return strings.Join(lines, "\n")
}
