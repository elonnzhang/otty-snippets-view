package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// status is a recent message, shown on the line just above the hints so the
// keys stay visible; "" when there is none. The prompt and the delete
// confirmation carry their own text inside their box.
func (m *model) status() string {
	switch {
	case m.mode == modePrompt || m.mode == modeConfirm:
		return ""
	case m.msg != "" && time.Now().Before(m.msgUntil):
		st := stErr
		if strings.HasPrefix(m.msg, "✓") {
			st = stOK
		}
		return st.Render(m.msg)
	}
	return ""
}

// drawDialog paints the group-name dialog over the finished screen. Its keys
// are handled before the view is built, like the right-click menu.
func (m *model) drawDialog(lines []string) {
	x, y, w := m.dialogBox()
	inner := w - 2
	label := tr("prompt.new")
	if m.promptGroup != "" {
		label = tr("prompt.ren")
	}
	cells := []string{
		stLabel.Render("┌" + strings.Repeat("─", inner) + "┐"),
		boxCell(stBold.Render(label)+m.prompt.View(), inner),
	}
	if m.promptMsg != "" {
		cells = append(cells, boxCell(stErr.Render(m.promptMsg), inner))
	}
	cells = append(cells, stLabel.Render("└"+strings.Repeat("─", inner)+"┘"))
	for i, c := range cells {
		row := y + i
		if row < 0 || row >= len(lines) {
			continue
		}
		line := lines[row]
		lines[row] = ansi.Truncate(line, x, "") + c + ansi.TruncateLeft(line, x+w, "")
	}
}

// boxCell draws one inner row of a dialog box: borders on both sides and
// exactly `inner` cells between them, so the columns line up.
func boxCell(s string, inner int) string {
	return stLabel.Render("│") + fitw(strings.Repeat(" ", dialogPad)+s, inner) + stLabel.Render("│")
}

// hintRow lays hints out in columns: keys on the first row, tips right below.
func (m *model) hintRow(hs []hint) [2]string {
	return m.hintRows(hs)
}

func (m *model) hintRows(hs []hint) [2]string {
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

// hintSpan is a clickable stretch of the key row: [start, end) columns.
type hintSpan struct {
	key  string
	col  int
	wid  int
	hint hint
}

// hintSpans lists the hints that fit, in the same order and column arithmetic
// as hintRow, so a click on a key can replay it. Both rows of a hint are the
// click target, from the keys row down to the bottom line.
func (m *model) hintSpans() []hintSpan {
	var out []hintSpan
	used := 0
	for _, h := range m.hintSet() {
		cw := max(ansi.StringWidth(h.key), ansi.StringWidth(h.tip))
		sep := 0
		if used > 0 {
			sep = 2
		}
		if used+sep+cw > m.w {
			break
		}
		out = append(out, hintSpan{key: h.key, col: used + sep, wid: cw, hint: h})
		used += sep + cw
	}
	return out
}

// hintAt returns the hint whose key was clicked at x on the hint rows, if any.
// Both rows count: the keys sit on the top one and the tips right below it.
func (m *model) hintAt(x, y int) (hint, bool) {
	if y < m.h-hintRows || y >= m.h {
		return hint{}, false
	}
	for _, s := range m.hintSpans() {
		if x >= s.col && x < s.col+s.wid {
			return s.hint, true
		}
	}
	return hint{}, false
}

// clickHint replays the key of a clicked hint, which runs the same action as
// pressing it.
func (m *model) clickHint(h hint) tea.Cmd {
	switch h.key {
	case "⏎", "⏎/click":
		return m.onKey(tea.KeyMsg{Type: tea.KeyEnter})
	case "Tab":
		return m.onKey(tea.KeyMsg{Type: tea.KeyTab})
	case "Esc":
		return m.onKey(tea.KeyMsg{Type: tea.KeyEsc})
	case "⌃S":
		return m.onKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	}
	if len([]rune(h.key)) != 1 {
		// "↑↓" and the like name several keys, so there is nothing unambiguous
		// to replay.
		return nil
	}
	return m.onKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(h.key)})
}

// confirmText is the delete confirmation: the title asks the question, the
// first body line is what goes, quoted, and the second says what the delete
// touches beyond it. The name gets a line of its own so a long one wraps
// instead of pushing the question out of the box.
func (m *model) confirmText() []string {
	if m.confirmGroup != "" {
		n := 0
		for _, s := range m.snips {
			if m.meta.groupOf(s.ID) == m.confirmGroup {
				n++
			}
		}
		lines := []string{quoted(m.meta.groupName(m.confirmGroup))}
		if kids := m.meta.descendants(m.confirmGroup); kids > 0 {
			return append(lines, fmt.Sprintf(tr("confirm.group.tree"), n, kids))
		}
		return append(lines, fmt.Sprintf(tr("confirm.group.body"), n))
	}
	name := m.confirmSnip
	if s := m.findSnip(m.confirmSnip); s != nil {
		name = s.Name
	}
	return []string{quoted(name), tr("confirm.snip.body")}
}

// quoted wraps a name the way the rest of the UI does in each language.
func quoted(s string) string { return pick(`"`+s+`"`, "「"+s+"」") }

// drawConfirm paints the delete confirmation over the finished screen: the
// question sits in a box of its own, like the other dialogs, instead of on the
// status line above the hints where it is easy to miss.
func (m *model) drawConfirm(lines []string) {
	x, y, w, body := m.confirmBox()
	inner := w - 2
	title := tr("confirm.snip")
	if m.confirmGroup != "" {
		title = tr("confirm.group")
	}
	cells := []string{stLabel.Render("┌" + strings.Repeat("─", inner) + "┐")}
	cells = append(cells, boxCell(stBold.Render(title), inner))
	for i, l := range body {
		st := stDim
		if i == 0 {
			st = stBold
		}
		cells = append(cells, boxCell(st.Render(l), inner))
	}
	var row strings.Builder
	for i, b := range confirmButtons() {
		if i > 0 {
			row.WriteString(strings.Repeat(" ", confirmGap))
		}
		row.WriteString(stKey.Render(b.key) + " " + b.st.Render(b.label))
	}
	cells = append(cells, boxCell("", inner), boxCell(row.String(), inner))
	cells = append(cells, stLabel.Render("└"+strings.Repeat("─", inner)+"┘"))
	paintBox(lines, x, y, w, cells)
}

// confirmButton is one answer on the confirmation's last row: drawn as its key
// and a label, clickable over both.
type confirmButton struct {
	key, label string
	st         lipgloss.Style
	answer     string // the key a click replays
}

const confirmGap = 3 // cells between the buttons

func confirmButtons() []confirmButton {
	return []confirmButton{
		{key: "y", label: tr("confirm.yes"), st: stErr, answer: "y"},
		{key: tr("confirm.anykey"), label: tr("h.cancel"), st: stDim, answer: "esc"},
	}
}

// confirmButtonAt returns the key the button under screen cell (cx, cy)
// answers with, or "" when the cell is not on a button.
func (m *model) confirmButtonAt(cx, cy int) string {
	x, y, _, body := m.confirmBox()
	if cy != y+confirmHeight(body)-2 { // the row above the bottom border
		return ""
	}
	col := x + 1 + dialogPad
	for _, b := range confirmButtons() {
		w := ansi.StringWidth(b.key) + 1 + ansi.StringWidth(b.label)
		if cx >= col && cx < col+w {
			return b.answer
		}
		col += w + confirmGap
	}
	return ""
}

// paintBox lays the rows of a dialog over the finished screen at x, y.
func paintBox(lines []string, x, y, w int, cells []string) {
	for i, c := range cells {
		row := y + i
		if row < 0 || row >= len(lines) {
			continue
		}
		lines[row] = ansi.Truncate(lines[row], x, "") + c + ansi.TruncateLeft(lines[row], x+w, "")
	}
}

// ---- form -------------------------------------------------------------------

// drawForm paints the snippet form over the finished screen, in the middle of
// it: the list stays visible around it, and the box sits where the eye is.
func (m *model) drawForm(lines []string) {
	f := m.form
	x, y, w, _ := m.formBox()
	inner := w - 2

	title := tr("form.edit")
	if f.orig.ID == "" {
		title = fmt.Sprintf(tr("form.new"), m.meta.groupName(f.gid))
	}
	cells := []string{stLabel.Render("┌" + strings.Repeat("─", inner) + "┐")}
	cells = append(cells, boxCell(stBold.Render(title), inner))
	cells = append(cells, m.formField(inner, 0, tr("form.name"), f.name))
	cells = append(cells, m.formField(inner, 1, tr("form.alias"), f.alias))
	text := tr("form.text")
	if f.focus == 2 {
		text = stFocused.Render(text)
	} else {
		text = stLabel.Render(text)
	}
	cells = append(cells, boxCell(text+" "+stDim.Render(tr("form.tiphint")), inner))
	// Body: only the window the box shows; the cursor block goes on the cell
	// the cursor is at. The ↑/↓ marks say the text goes on above or below.
	body := make([]string, f.rows)
	for i := range body {
		if j := f.scroll + i; j >= 0 && j < len(f.lines) {
			body[i] = f.lines[j]
		}
	}
	if row, cell, ch := f.cursorSpot(); row >= f.scroll && row < f.scroll+f.rows {
		i := row - f.scroll
		body[i] = ansi.Truncate(body[i], cell, "") + stCursor.Render(ch) +
			ansi.TruncateLeft(fitw(body[i], max(cell+1, ansi.StringWidth(body[i]))), cell+1, "")
	}
	more := f.scroll+f.rows < f.total()
	for i := range body {
		mark := " "
		switch {
		case i == 0 && f.scroll > 0:
			mark = stDim.Render("↑")
		case i == f.rows-1 && more:
			mark = stDim.Render("↓")
		}
		// 1 cell goes to the scroll mark, the rest to text and padding.
		cells = append(cells, stLabel.Render("│")+fitw(strings.Repeat(" ", dialogPad)+body[i], inner-1)+mark+stLabel.Render("│"))
	}
	if m.formMsg != "" {
		// The reason sits on the last body line, inside the box, so it cannot
		// be mistaken for a status message.
		cells[len(cells)-1] = boxCell(stErr.Render(m.formMsg), inner)
	}
	cells = append(cells, stLabel.Render("└"+strings.Repeat("─", inner)+"┘"))

	for i, c := range cells {
		row := y + i
		if row < 0 || row >= len(lines) {
			continue
		}
		lines[row] = ansi.Truncate(lines[row], x, "") + c + ansi.TruncateLeft(lines[row], x+w, "")
	}
}

// formField draws one label + input row of the form.
func (m *model) formField(inner, i int, name string, in textinput.Model) string {
	st := stLabel
	if m.form.focus == i {
		st = stFocused
	}
	return boxCell(st.Render(name)+strings.Repeat(" ", dialogPad)+in.View(), inner)
}
