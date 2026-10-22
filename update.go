package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const doubleClick = 450 * time.Millisecond

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.fit()
		if m.form != nil {
			m.formLayout() // the box and the body window follow the terminal
		}
	case tickMsg:
		// Pick up snippets added or edited in Otty's own Recipes settings.
		if m.mode == modeNormal && m.drag == nil && !m.resizing && signature() != m.sig {
			m.reload()
		}
		return m, tick()
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		return m, tea.Batch(m.onKey(msg), m.mouseMode())
	case tea.MouseMsg:
		return m, tea.Batch(m.onMouse(msg), m.mouseMode())
	}
	return m, nil
}

// mouseMode asks the terminal for plain pointer motion while the right-click
// menu is open, so its highlight follows the pointer, and drops back to
// motion-while-pressed afterwards; nil when nothing changes.
func (m *model) mouseMode() tea.Cmd {
	want := m.mode == modeContext
	if want == m.hover {
		return nil
	}
	m.hover = want
	if want {
		return tea.EnableMouseAllMotion
	}
	return tea.EnableMouseCellMotion
}

func (m *model) onKey(k tea.KeyMsg) tea.Cmd {
	switch m.mode {
	case modeSearch:
		return m.keySearch(k)
	case modePrompt:
		return m.keyPrompt(k)
	case modeMenu:
		m.keyMenu(k.String())
		return nil
	case modeContext:
		return m.keyCtx(k.String())
	case modeConfirm:
		m.keyConfirm(k.String())
		return nil
	case modeForm:
		return m.keyForm(k)
	}
	r := m.cur()
	isG := r != nil && r.group
	named := isG && r.gid != "" // a real group, not 未分组
	lh := max(1, m.layout().listH)
	switch k.String() {
	case "up", "k":
		m.moveSel(-1)
	case "down", "j":
		m.moveSel(1)
	case "pgup":
		m.moveSel(-lh)
	case "pgdown":
		m.moveSel(lh)
	case "home", "g":
		m.moveSel(-len(m.rows))
	case "end", "G":
		m.moveSel(len(m.rows))
	case "tab", "shift+tab":
		m.setTab(1 - m.tab)
	case "1":
		m.setTab(tabList)
	case "2":
		m.setTab(tabGroup)
	case "+", "=":
		m.growDetail(1)
	case "-":
		m.growDetail(-1)
	case "enter":
		if isG {
			m.toggle(r.gid)
		} else {
			m.send(true)
		}
	case " ":
		if isG {
			m.toggle(r.gid)
		}
	case "left":
		if m.tab == tabGroup && r != nil && !r.group {
			for i := m.sel; i >= 0; i-- {
				if m.rows[i].group {
					m.sel = i
					break
				}
			}
			m.fit()
		} else if isG && !r.folded {
			m.toggle(r.gid)
		}
	case "right":
		if isG && r.folded {
			m.toggle(r.gid)
		}
	case "o", "ctrl+o":
		m.send(false)
	case "y":
		if s := m.curSnip(); s != nil {
			m.say(copySnippet(*s))
		}
	case "a": // new snippet, in the current group when there is one
		gid := ""
		if r != nil && m.tab == tabGroup {
			gid = r.gid
		}
		m.form, m.mode = newForm(Snippet{}, gid), modeForm
		m.formMsg = ""
		m.formLayout()
	case "e":
		if s := m.curSnip(); s != nil {
			if s.Foreign {
				m.say(tr("foreign"))
			} else {
				m.form, m.mode = newForm(*s, ""), modeForm
				m.formMsg = ""
				m.formLayout()
			}
		} else if named {
			m.startPrompt(r.gid, "", r.name)
		}
	case "n":
		m.startPrompt("", "", "")
	case "r":
		if named {
			m.startPrompt(r.gid, "", r.name)
		}
	case "m":
		m.openMenu(isG)
	case "c":
		m.openColorMenu(m.sel, 2, m.layout().listTop+m.sel-m.scroll)
	case "M":
		if isG && r.gid != "" {
			m.startChildPrompt(r.gid, "")
		}
	case "d", "x", "delete":
		if named {
			m.confirmGroup, m.confirmSnip, m.mode = r.gid, "", modeConfirm
		} else if s := m.curSnip(); s != nil {
			m.confirmGroup, m.confirmSnip, m.mode = "", s.ID, modeConfirm
		}
	case "[", "]":
		if named {
			d := 1
			if k.String() == "[" {
				d = -1
			}
			if m.meta.shift(r.gid, d) {
				m.persist()
				m.rebuild("g:" + r.gid)
			}
		}
	case "/":
		m.mode = modeSearch
	case "esc":
		if m.query != "" {
			m.query = ""
			m.rebuild("")
		}
	case "R", "ctrl+r":
		m.reload()
		m.say(tr("refreshed"))
	}
	return nil
}

func (m *model) keySearch(k tea.KeyMsg) tea.Cmd {
	switch k.Type {
	case tea.KeyEsc:
		m.query, m.mode = "", modeNormal
	case tea.KeyEnter:
		m.mode = modeNormal
		return nil
	case tea.KeyUp, tea.KeyDown:
		if k.Type == tea.KeyUp {
			m.moveSel(-1)
		} else {
			m.moveSel(1)
		}
		return nil
	case tea.KeyBackspace:
		if r := []rune(m.query); len(r) > 0 {
			m.query = string(r[:len(r)-1])
		}
	case tea.KeyRunes, tea.KeySpace:
		m.query += string(k.Runes)
	default:
		return nil
	}
	m.rebuild("")
	return nil
}

func (m *model) keyPrompt(k tea.KeyMsg) tea.Cmd {
	switch k.Type {
	case tea.KeyEsc:
		m.mode = modeNormal
		return nil
	case tea.KeyEnter:
		m.commitPrompt()
		return nil
	}
	var cmd tea.Cmd
	m.prompt, cmd = m.prompt.Update(k)
	if m.promptMsg != "" {
		// Typing again is the fix; drop the error line as soon as it changes.
		m.promptMsg = ""
	}
	return cmd
}

func (m *model) keyMenu(k string) {
	switch k {
	case "up", "k":
		m.menuSel = max(0, m.menuSel-1)
	case "down", "j":
		m.menuSel = min(len(m.menuItems)-1, m.menuSel+1)
	case "esc", "q":
		m.mode = modeNormal
	case "enter":
		m.mode = modeNormal
		if m.menuSel < 0 || m.menuSel >= len(m.menuItems) {
			return
		}
		it := m.menuItems[m.menuSel]
		subject := ""
		if m.menuFor != nil {
			subject = m.menuFor.ID
		}
		switch {
		case it.gid == "+" && m.menuFor != nil:
			m.startPrompt("", subject, "")
		case it.gid == "+":
			m.startChildPrompt(m.menuGroup, "") // the new group is not the parent
			m.promptParent = ""
		case m.menuFor != nil:
			m.drop(subject, it.gid)
		case it.gid == "-":
			m.dropGroup(m.menuGroup, "")
		default:
			if m.meta.isAncestor(it.gid, m.menuGroup) {
				m.say(tr("pick.cycle"))
				return
			}
			m.dropGroup(m.menuGroup, it.gid)
		}
	}
}

func (m *model) keyConfirm(k string) {
	m.mode = modeNormal
	if k != "y" && k != "Y" {
		return
	}
	if m.confirmGroup != "" {
		name := m.meta.groupName(m.confirmGroup)
		kids := m.meta.descendants(m.confirmGroup)
		m.meta.remove(m.confirmGroup)
		m.persist()
		m.rebuild("")
		if kids > 0 {
			m.say(fmt.Sprintf(tr("group.deleted.tree"), name))
		} else {
			m.say(fmt.Sprintf(tr("group.deleted"), name))
		}
	} else {
		m.deleteSnippet(m.confirmSnip)
	}
}

func (m *model) keyForm(k tea.KeyMsg) tea.Cmd {
	f := m.form
	switch k.String() {
	case "esc":
		m.form, m.mode, m.formMsg = nil, modeNormal, ""
		return nil
	case "tab":
		f.setFocus(f.focus + 1)
		return nil
	case "shift+tab":
		f.setFocus(f.focus - 1)
		return nil
	case "ctrl+s":
		m.saveForm()
		return nil
	case "enter":
		if f.focus < 2 {
			f.setFocus(f.focus + 1)
			return nil
		}
	}
	cmd := f.update(k)
	if k.String() != "esc" {
		// Every edit can add or remove rows; keep the caret in view.
		m.formFollow()
	}
	return cmd
}

func (m *model) saveForm() {
	s, problem := m.form.result(m.snips)
	if problem != "" {
		// The dialog stays open with the reason inside it, like the group
		// name dialog.
		m.formMsg = problem
		return
	}
	isNew := s.ID == ""
	id, err := saveSnippet(s)
	if err != nil {
		m.formMsg = fmt.Sprintf(tr("save.failed"), err)
		return
	}
	if _, _, prefixed := splitPrefix(s.Name); isNew && m.form.gid != "" && !prefixed {
		m.meta.move(id, m.form.gid)
		m.persist()
	}
	m.form, m.mode, m.formMsg = nil, modeNormal, ""
	m.reload()
	m.rebuild("s:" + id)
	if isNew {
		m.say(fmt.Sprintf(tr("snip.new"), s.Name))
	} else {
		m.say(fmt.Sprintf(tr("snip.saved"), s.Name))
	}
}

// ---- mouse ------------------------------------------------------------------

func (m *model) onMouse(e tea.MouseMsg) tea.Cmd {
	l := m.layout()
	left := e.Button == tea.MouseButtonLeft
	press := e.Action == tea.MouseActionPress

	// The tab bar works in every mode except while a dialog wants an answer.
	if e.Y == 0 && press && left && (m.mode == modeNormal || m.mode == modeSearch) {
		if t, ok := tabAt(e.X); ok && t != m.tab {
			m.mode = modeNormal
			m.setTab(t)
		}
		return nil
	}

	// Resizing the detail area by its top border.
	if m.resizing {
		switch e.Action {
		case tea.MouseActionMotion:
			m.setDetailFromDivider(e.Y)
		case tea.MouseActionRelease:
			m.resizing = false
			m.persist()
		}
		return nil
	}
	if press && left && l.divY >= 0 && e.Y == l.divY && m.mode != modeForm {
		m.resizing = true
		return nil
	}

	switch m.mode {
	case modeMenu:
		m.mouseMenu(e, l)
		return nil
	case modeContext:
		return m.mouseCtx(e)
	case modeForm:
		// Like the group dialog: a click inside picks the field under it (in
		// the body, also the character), a click outside cancels the form. The
		// wheel only scrolls the body, when the pointer is over it.
		if !press {
			return nil
		}
		switch e.Button {
		case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
			x, y, w, _ := m.formBox()
			inBody := e.X > x && e.X < x+w && e.Y >= y+formTextTop && e.Y < y+formTextTop+m.form.rows
			if inBody {
				d := 3
				if e.Button == tea.MouseButtonWheelUp {
					d = -3
				}
				m.form.scroll += d
				m.form.clampScroll()
			}
		default:
			if m.clickOutsideForm(e) {
				m.form, m.formMsg, m.mode = nil, "", modeNormal
			} else if left {
				m.clickForm(e)
			}
		}
		return nil
	case modePrompt:
		// A click outside the box cancels, like Esc; inside, the input keeps
		// its cursor where it was.
		if press && m.ctxClickOutsideDialog(e) {
			m.mode = modeNormal
			m.promptMsg = ""
		}
		return nil
	case modeConfirm, modeSearch:
		return nil
	}

	// The hint rows are a second toolbar: clicking a key runs it.
	if press && left {
		if h, ok := m.hintAt(e.X, e.Y); ok {
			return m.clickHint(h)
		}
	}

	inDetail := l.detH > 0 && e.Y >= l.detTop && e.Y < l.detTop+l.detH
	idx := m.scroll + e.Y - l.listTop
	inList := e.Y >= l.listTop && e.Y < l.listTop+l.listH && idx >= 0 && idx < len(m.rows)

	switch {
	case e.Button == tea.MouseButtonWheelUp || e.Button == tea.MouseButtonWheelDown:
		d := 3
		if e.Button == tea.MouseButtonWheelUp {
			d = -3
		}
		if inDetail {
			m.detScroll = max(0, m.detScroll+d)
		} else {
			m.moveSel(d)
		}
	case e.Action == tea.MouseActionMotion:
		if m.drag != nil {
			m.dragTo(e.Y, l)
		}
	case press:
		if !inList {
			if e.Y >= l.listTop && e.Y < l.listTop+l.listH {
				switch e.Button {
				case tea.MouseButtonRight:
					// Right-click on the empty part of the list: menu for new things only.
					m.openCtx(nil, e.X, e.Y)
				case tea.MouseButtonLeft:
					// Left-click on empty space: drop the selection so the
					// list (and the detail area beside it) is not tied to an
					// item. Moving the pointer is not a click.
					m.clearSel()
				}
			}
			return nil
		}
		m.sel = idx
		m.fit()
		r := m.rows[idx]
		switch e.Button {
		case tea.MouseButtonLeft:
			m.pressRow, m.drag = idx, nil
			if m.tab == tabGroup {
				if r.group && r.gid != "" {
					m.drag = &drag{gid: r.gid, name: r.name, from: idx, hover: idx}
				} else if !r.group {
					m.drag = &drag{sid: r.snip.ID, name: r.snip.Name, from: idx, hover: idx}
				}
			}
		case tea.MouseButtonRight:
			m.openCtx(&r, e.X, e.Y)
		}
	case e.Action == tea.MouseActionRelease:
		m.release(idx, inList)
	}
	return nil
}

// ctxClickOutsideDialog reports whether e lands outside the group-name box.
func (m *model) ctxClickOutsideDialog(e tea.MouseMsg) bool {
	x, y, w := m.dialogBox()
	return e.X < x || e.X >= x+w || e.Y < y || e.Y >= y+m.dialogHeight()
}

// clickOutsideForm reports whether e lands outside the snippet form.
func (m *model) clickOutsideForm(e tea.MouseMsg) bool {
	x, y, w, h := m.formBox()
	return e.X < x || e.X >= x+w || e.Y < y || e.Y >= y+h
}

func (m *model) clearSel() {
	if m.sel < 0 {
		return
	}
	m.sel = -1
	m.lastClickID = "" // a cleared row is not a double-click target
	m.drag, m.pressRow = nil, -1
	m.fit() // resets the detail scroll; the divider now shows the placeholder
}

func (m *model) dragTo(y int, l layout) {
	d := m.drag
	// Dragging past the top or bottom edge of the list scrolls it.
	if y < l.listTop && m.scroll > 0 {
		m.scroll--
	} else if y >= l.listTop+l.listH && m.scroll+l.listH < len(m.rows) {
		m.scroll++
	}
	h := m.scroll + min(max(y, l.listTop), l.listTop+l.listH-1) - l.listTop
	if h >= len(m.rows) {
		h = -1
	}
	if h != d.from {
		d.active = true
	}
	d.hover = h
}

func (m *model) release(idx int, inList bool) {
	d := m.drag
	m.drag = nil
	if d != nil && d.active {
		if d.hover < 0 || d.hover >= len(m.rows) {
			return
		}
		h := m.rows[d.hover]
		if d.gid != "" {
			// A group dropped on another group becomes its child; on a snippet
			// or on 未分组 it goes back to the top level.
			parent := ""
			if h.group {
				parent = h.gid
			}
			if m.meta.isAncestor(parent, d.gid) {
				m.say(tr("pick.cycle"))
				return
			}
			m.dropGroup(d.gid, parent)
			return
		}
		m.drop(d.sid, h.gid)
		return
	}
	if !inList || idx != m.pressRow {
		return
	}
	r := m.rows[idx]
	if r.group {
		m.toggle(r.gid)
		return
	}
	now := time.Now()
	if m.lastClickID == r.snip.ID && now.Sub(m.lastClickAt) < doubleClick {
		m.lastClickID = ""
		m.send(true)
	} else {
		m.lastClickID, m.lastClickAt = r.snip.ID, now
	}
}

func (m *model) mouseMenu(e tea.MouseMsg, l layout) {
	switch {
	case e.Button == tea.MouseButtonWheelUp:
		m.keyMenu("up")
	case e.Button == tea.MouseButtonWheelDown:
		m.keyMenu("down")
	case e.Action == tea.MouseActionPress && e.Button == tea.MouseButtonLeft:
		first, _ := m.menuWindow()
		j := first + e.Y - l.listTop - 1 // the menu title takes the first list row
		if j >= 0 && j < len(m.menuItems) {
			m.menuSel = j
			m.keyMenu("enter")
		}
	}
}

// menuWindow returns the first visible menu item and how many fit.
func (m *model) menuWindow() (int, int) {
	n := max(1, m.layout().listH-1)
	first := max(0, min(m.menuSel-n+1, len(m.menuItems)-n))
	return first, n
}
