package main

import (
	"fmt"
	"strings"
)

func (m *model) toggle(gid string) {
	if gid == "" {
		m.foldNone = !m.foldNone
	} else if g := m.meta.find(gid); g != nil {
		g.Collapsed = !g.Collapsed
		m.persist()
	}
	m.rebuild("g:" + gid)
}

func (m *model) drop(sid, gid string) {
	if m.meta.groupOf(sid) == gid {
		m.say(fmt.Sprintf(tr("already.in"), m.meta.groupName(gid)))
		return
	}
	m.meta.move(sid, gid)
	m.persist()
	m.rebuild("s:" + sid)
	m.say(fmt.Sprintf(tr("moved.to"), m.meta.groupName(gid)))
}

func (m *model) send(run bool) {
	if s := m.curSnip(); s != nil {
		m.say(sendSnippet(*s, run))
	}
}

func (m *model) openMenu() {
	s := m.curSnip()
	if s == nil {
		m.say(tr("pick.first"))
		return
	}
	m.menuItems = nil
	cur := m.meta.groupOf(s.ID)
	m.menuSel = 0
	for _, g := range m.meta.Groups {
		m.menuItems = append(m.menuItems, menuItem{g.ID, g.Name})
	}
	m.menuItems = append(m.menuItems, menuItem{"", ungroupedName()}, menuItem{"+", tr("menu.new")})
	for i, it := range m.menuItems {
		if it.gid == cur {
			m.menuSel = i
		}
	}
	m.menuFor, m.mode = s, modeMenu
}

func (m *model) startPrompt(gid, sid, text string) {
	m.prompt = newTextInput(text)
	m.promptGroup, m.promptSnipID, m.mode = gid, sid, modePrompt
}

func (m *model) commitPrompt() {
	name := strings.TrimSpace(m.prompt.Value())
	if !m.meta.nameOK(name, m.promptGroup) {
		m.say(tr("prompt.bad"))
		return
	}
	gid := m.promptGroup
	if gid == "" {
		gid = m.meta.add(name)
		if m.promptSnipID != "" {
			m.meta.move(m.promptSnipID, gid)
		}
	} else {
		m.meta.rename(gid, name)
	}
	m.persist()
	m.mode = modeNormal
	if m.promptSnipID != "" {
		m.rebuild("s:" + m.promptSnipID)
	} else {
		m.rebuild("g:" + gid)
	}
	m.say(fmt.Sprintf(tr("group.saved"), name))
}

func (m *model) deleteSnippet(id string) {
	s := m.findSnip(id)
	if s == nil {
		return
	}
	name := s.Name
	if err := trashSnippet(id); err != nil {
		m.say(fmt.Sprintf(tr("del.failed"), err))
		return
	}
	m.meta.move(id, "")
	m.persist()
	m.reload()
	m.say(fmt.Sprintf(tr("snip.deleted"), name))
}

// setDetail resizes the detail area so its top border sits on screen row y.
func (m *model) setDetailFromDivider(y int) {
	d := m.h - hintRows - 1 - y
	m.meta.UI.DetailHeight = max(1, min(d, m.maxDetail()))
	m.fit()
}

func (m *model) growDetail(d int) {
	m.meta.UI.DetailHeight = max(1, min(m.layout().detH+d, m.maxDetail()))
	m.persist()
	m.fit()
}
