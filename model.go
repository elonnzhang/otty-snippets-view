package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type mode int

const (
	modeNormal mode = iota
	modeSearch
	modePrompt  // new / rename group
	modeMenu    // move snippet to group
	modeConfirm // delete group / snippet
	modeForm    // new / edit snippet
	modeContext // right-click menu
)

type tab int

const (
	tabList  tab = iota // every snippet, flat
	tabGroup            // snippets under their groups; group management
)

type row struct {
	group  bool
	depth  int    // nesting level in the group tree
	gid    string // group id; "" = ungrouped
	name   string // group name (header rows)
	count  int
	folded bool
	snip   *Snippet
}

func (r row) key() string {
	if r.group {
		return "g:" + r.gid
	}
	return "s:" + r.snip.ID
}

type drag struct {
	sid, gid string // gid set when a group is dragged
	name     string
	from     int
	hover    int // row index under the pointer, -1 if none
	active   bool
}

type menuItem struct{ gid, label string } // gid "+" = new group, "-" = no entry

type model struct {
	meta  *Meta
	snips []Snippet
	sig   string
	rows  []row
	tab   tab

	// sel is the selected row; -1 after a click on the empty part of the list,
	// where down selects the first row and up the last.
	sel, scroll, w, h int
	detScroll         int
	detKey            string // row the detail scroll belongs to
	hl                hlCache
	mode              mode
	query             string
	foldNone          bool
	msg               string
	msgUntil          time.Time

	drag        *drag
	resizing    bool
	pressRow    int
	lastClickID string
	lastClickAt time.Time

	menuFor   *Snippet // snippet being moved, nil when moving a group
	menuGroup string   // group being moved
	menuItems []menuItem
	menuSel   int
	ctx       *ctxMenu // right-click menu, when mode is modeContext
	hover     bool     // terminal reports pointer motion without a button

	prompt       textinput.Model
	promptGroup  string // group being renamed; "" for a new one
	promptSnipID string // snippet to move into the new group
	promptParent string // group the new group goes under

	confirmGroup, confirmSnip string

	form *form
}

type tickMsg time.Time

func tick() tea.Cmd { return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) }) }

func newModel() *model {
	m := &model{meta: loadMeta(), pressRow: -1}
	if m.meta.UI.Tab == "group" {
		m.tab = tabGroup
	}
	m.reload()
	return m
}

func (m *model) Init() tea.Cmd { return tick() }

func (m *model) reload() {
	m.snips, m.sig = loadSnippets(), signature()
	pruned := m.meta.prune(m.snips)
	if m.meta.resolve(m.snips) || pruned {
		m.persist()
	}
	m.rebuild("")
}

func (m *model) say(s string) { m.msg, m.msgUntil = s, time.Now().Add(4*time.Second) }

func (m *model) persist() {
	if err := m.meta.save(); err != nil {
		m.say(fmt.Sprintf(tr("save.failed"), err))
	}
}

func (m *model) cur() *row {
	if m.sel < 0 || m.sel >= len(m.rows) {
		return nil
	}
	return &m.rows[m.sel]
}

func (m *model) curSnip() *Snippet {
	if r := m.cur(); r != nil && !r.group {
		return r.snip
	}
	return nil
}

func (m *model) findSnip(id string) *Snippet {
	for i := range m.snips {
		if m.snips[i].ID == id {
			return &m.snips[i]
		}
	}
	return nil
}

func (m *model) setTab(t tab) {
	m.tab = t
	m.meta.UI.Tab = map[tab]string{tabList: "list", tabGroup: "group"}[t]
	m.persist()
	m.rebuild("")
}

// rebuild recomputes the visible rows for the current tab, keeping the row
// with key `keep` (or the current one) selected.
func (m *model) rebuild(keep string) {
	if keep == "" {
		if r := m.cur(); r != nil {
			keep = r.key()
		}
	}
	q := strings.ToLower(m.query)
	var hits []*Snippet
	for i := range m.snips {
		s := &m.snips[i]
		if q == "" || strings.Contains(strings.ToLower(s.Name+" "+s.Alias+" "+s.Text), q) {
			hits = append(hits, s)
		}
	}
	var rows []row
	if m.tab == tabList {
		for _, s := range hits {
			rows = append(rows, row{gid: m.meta.groupOf(s.ID), snip: s})
		}
	} else {
		buckets := map[string][]*Snippet{}
		for _, s := range hits {
			gid := m.meta.groupOf(s.ID)
			buckets[gid] = append(buckets[gid], s)
		}
		// Walking by children, not list order, keeps a parent above its children
		// even when the stored order has drifted.
		var walk func(parent string, depth int)
		walk = func(parent string, depth int) {
			for _, g := range m.meta.children(parent) {
				items, hasKids := buckets[g.ID], len(m.meta.children(g.ID)) > 0
				if q != "" && len(items) == 0 && !hasKids {
					continue
				}
				folded := g.Collapsed && q == ""
				rows = append(rows, row{group: true, depth: depth, gid: g.ID, name: g.Name,
					count: len(items), folded: folded})
				if folded {
					continue
				}
				for _, s := range items {
					rows = append(rows, row{depth: depth + 1, gid: g.ID, snip: s})
				}
				walk(g.ID, depth+1)
			}
		}
		walk("", 0)
		items := buckets[""]
		if q == "" || len(items) > 0 {
			folded := m.foldNone && q == ""
			rows = append(rows, row{group: true, gid: "", name: ungroupedName(), count: len(items), folded: folded})
			if !folded {
				for _, s := range items {
					rows = append(rows, row{gid: "", snip: s})
				}
			}
		}
	}
	m.rows = rows

	found := -1
	for i, r := range rows {
		if r.key() == keep {
			found = i
			break
		}
	}
	if found < 0 && (q != "" || strings.HasPrefix(keep, "g:")) {
		for i, r := range rows {
			if !r.group {
				found = i
				break
			}
		}
	}
	if found >= 0 {
		m.sel = found
	}
	m.fit()
}

// layout splits the screen, top to bottom: tab bar, list, divider, detail, hints.
type layout struct {
	listTop, listH int
	divY           int // the detail's top border, draggable; -1 when there is no room
	detTop, detH   int
	hintTop, hintH int
}

const hintRows = 2 // key row + tip row

func (m *model) maxDetail() int { return m.h - 1 - hintRows - 1 - 2 } // keep ≥2 list rows

func (m *model) layout() layout {
	l := layout{listTop: 1, divY: -1, hintH: hintRows, hintTop: m.h - hintRows}
	avail := m.h - 1 - hintRows
	if avail < 6 {
		l.listH = max(0, avail)
		return l
	}
	d := m.meta.UI.DetailHeight
	if d <= 0 {
		d = avail / 3
	}
	l.detH = max(1, min(d, m.maxDetail()))
	l.divY = m.h - hintRows - l.detH - 1
	l.detTop = l.divY + 1
	l.listH = l.divY - l.listTop
	return l
}

func (m *model) fit() {
	lh := max(1, m.layout().listH)
	// sel -1 means nothing is selected (a click on the empty part of the list).
	if m.sel >= 0 {
		m.sel = min(m.sel, len(m.rows)-1)
	}
	m.scroll = max(0, min(m.scroll, len(m.rows)-lh))
	if m.sel >= 0 && m.sel < m.scroll {
		m.scroll = m.sel
	} else if m.sel >= m.scroll+lh {
		m.scroll = m.sel - lh + 1
	}
	if r := m.cur(); r == nil || r.key() != m.detKey {
		m.detScroll = 0
		m.detKey = ""
		if r != nil {
			m.detKey = r.key()
		}
	}
}

func (m *model) moveSel(d int) {
	if len(m.rows) == 0 {
		return
	}
	if m.sel < 0 {
		// Nothing is selected after a click on empty space: down starts at the
		// top row, up starts at the bottom one.
		if d > 0 {
			m.sel = 0
		} else {
			m.sel = len(m.rows) - 1
		}
		m.fit()
		return
	}
	m.sel = max(0, min(len(m.rows)-1, m.sel+d))
	m.fit()
}
