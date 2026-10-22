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

// openMenu offers group destinations for the selected snippet, or, with
// groupWanted set, parents for the selected group.
func (m *model) openMenu(groupWanted bool) {
	m.menuItems, m.menuSel = nil, 0
	var cur string
	switch r := m.cur(); {
	case r == nil:
		m.say(tr("pick.first"))
		return
	case groupWanted && !r.group:
		m.say(tr("pick.group"))
		return
	case groupWanted && r.gid == "":
		m.say(tr("group.no.move"))
		return
	case groupWanted:
		g := m.meta.find(r.gid)
		if g != nil {
			cur = g.Parent
		}
		m.menuGroup, m.menuFor = r.gid, nil
		m.menuItems = append(m.menuItems, menuItem{"-", tr("menu.no.parent")})
		for _, id := range m.meta.treeOrder("") {
			g := m.meta.find(id)
			depth := strings.Count(m.meta.path(id), " / ")
			mark := ""
			if m.meta.isAncestor(id, r.gid) {
				mark = "  ✗"
			}
			m.menuItems = append(m.menuItems, menuItem{id, strings.Repeat("  ", depth) + g.Name + mark})
		}
		m.menuItems = append(m.menuItems, menuItem{"+", tr("menu.new")})
	case r.group:
		m.say(tr("pick.snip"))
		return
	default:
		cur = m.meta.groupOf(r.snip.ID)
		m.menuFor, m.menuGroup = r.snip, ""
		for _, id := range m.meta.treeOrder("") {
			g := m.meta.find(id)
			depth := strings.Count(m.meta.path(id), " / ")
			m.menuItems = append(m.menuItems, menuItem{id, strings.Repeat("  ", depth) + g.Name})
		}
		m.menuItems = append(m.menuItems, menuItem{"", ungroupedName()}, menuItem{"+", tr("menu.new")})
	}
	for i, it := range m.menuItems {
		if it.gid == cur {
			m.menuSel = i
		}
	}
	m.mode = modeMenu
}

// dropGroup puts group gid under parent ("" = top level).
func (m *model) dropGroup(gid, parent string) {
	name := m.meta.groupName(gid)
	if !m.meta.setParent(gid, parent) {
		m.say(fmt.Sprintf(tr("already.in"), m.meta.groupName(parent)))
		return
	}
	m.persist()
	m.rebuild("g:" + gid)
	if parent == "" {
		m.say(fmt.Sprintf(tr("moved.top"), name))
	} else {
		m.say(fmt.Sprintf(tr("moved.group"), name, m.meta.groupName(parent)))
	}
}

func (m *model) startPrompt(gid, sid, text string) {
	m.prompt = newTextInput(text)
	m.prompt.Width = m.dialogInputWidth()
	m.promptMsg = ""
	m.promptGroup, m.promptSnipID, m.promptParent, m.mode = gid, sid, "", modePrompt
}

// startChildPrompt asks for a new group that will sit under parent.
func (m *model) startChildPrompt(parent, sid string) {
	m.startPrompt("", sid, "")
	m.promptParent = parent
}

func (m *model) commitPrompt() {
	name := strings.TrimSpace(m.prompt.Value())
	if !m.meta.nameOK(name, m.promptGroup) {
		// The dialog stays open with the reason under the input, so the name
		// can be fixed right where it was typed.
		m.promptMsg = tr("prompt.bad")
		return
	}
	gid := m.promptGroup
	if gid == "" {
		gid = m.meta.addUnder(name, m.promptParent)
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
	if m.promptParent != "" && m.promptGroup == "" {
		m.say(fmt.Sprintf(tr("group.under.new"), m.meta.groupName(m.promptParent)))
	} else {
		m.say(fmt.Sprintf(tr("group.saved"), name))
	}
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
