package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

func (m *Meta) find(gid string) *Group {
	for i := range m.Groups {
		if m.Groups[i].ID == gid {
			return &m.Groups[i]
		}
	}
	return nil
}

// groupOf returns the snippet's group id, or "" when it is ungrouped. An
// explicit assignment (drag, move menu) wins over the name's [group]- prefix.
func (m *Meta) groupOf(sid string) string {
	if gid, ok := m.Assign[sid]; ok {
		if gid == noGroup {
			return ""
		}
		if m.find(gid) != nil {
			return gid
		}
	}
	if gid := m.auto[sid]; m.find(gid) != nil {
		return gid
	}
	return ""
}

// noGroup in Assign pins a snippet to 未分组 even though its name has a prefix.
const noGroup = "none"

var prefixRe = regexp.MustCompile(`^\[([^\[\]]+)\]\s*-?\s*(.*)$`)

// splitPrefix parses "[group]-name" into ("group", "name").
func splitPrefix(name string) (group, rest string, ok bool) {
	sm := prefixRe.FindStringSubmatch(name)
	if sm == nil || strings.TrimSpace(sm[1]) == "" {
		return "", name, false
	}
	return strings.TrimSpace(sm[1]), sm[2], true
}

func (m *Meta) findByName(name string) *Group {
	for i := range m.Groups {
		if strings.EqualFold(m.Groups[i].Name, name) {
			return &m.Groups[i]
		}
	}
	return nil
}

// resolve maps every "[group]-name" snippet to its group, creating groups that
// don't exist yet. It reports whether it created any.
func (m *Meta) resolve(snips []Snippet) bool {
	m.auto, m.prefixed = map[string]string{}, map[string]bool{}
	created := false
	for _, s := range snips {
		g, _, ok := splitPrefix(s.Name)
		if !ok {
			continue
		}
		m.prefixed[s.ID] = true
		gr := m.findByName(g)
		if gr == nil {
			if _, pinned := m.Assign[s.ID]; pinned {
				continue // placed by hand; don't resurrect a group for it
			}
			m.add(g)
			gr = &m.Groups[len(m.Groups)-1]
			created = true
		}
		m.auto[s.ID] = gr.ID
	}
	return created
}

func (m *Meta) groupName(gid string) string {
	if g := m.find(gid); g != nil {
		return g.Name
	}
	return ungroupedName()
}

func (m *Meta) nameOK(name, skip string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" || isUngroupedLabel(n) {
		return false
	}
	for _, g := range m.Groups {
		if g.ID != skip && strings.ToLower(g.Name) == n {
			return false
		}
	}
	return true
}

func (m *Meta) add(name string) string { return m.addUnder(name, "") }

func (m *Meta) addUnder(name, parent string) string {
	gid := strings.ToLower(newID()[:8])
	m.Groups = append(m.Groups, Group{ID: gid, Name: strings.TrimSpace(name), Parent: parent})
	return gid
}

// children returns the groups directly under parent ("" = top level), in list order.
func (m *Meta) children(parent string) []*Group {
	var out []*Group
	for i := range m.Groups {
		p := m.Groups[i].Parent
		if m.find(p) == nil {
			p = "" // a stale parent means top level
		}
		if p == parent {
			out = append(out, &m.Groups[i])
		}
	}
	return out
}

// treeOrder lists group ids depth-first, parents before their children.
func (m *Meta) treeOrder(parent string) []string {
	var out []string
	for _, g := range m.children(parent) {
		out = append(out, g.ID)
		out = append(out, m.treeOrder(g.ID)...)
	}
	return out
}

func (m *Meta) descendants(gid string) int {
	n := 0
	for _, g := range m.children(gid) {
		n += 1 + m.descendants(g.ID)
	}
	return n
}

// isAncestor reports whether a is gid itself or lies above it in the tree.
// Moving gid under a would then create a cycle.
func (m *Meta) isAncestor(a, gid string) bool {
	if gid == "" {
		return false // the top level is every group's ancestor, not the reverse
	}
	for n := m.find(a); n != nil; n = m.find(n.Parent) {
		if n.ID == gid {
			return true
		}
	}
	return false
}

// setParent moves a group under parent ("" = top level). It refuses moves that
// would create a cycle, and goes nowhere if the parent is unchanged.
func (m *Meta) setParent(gid, parent string) bool {
	g := m.find(gid)
	if g == nil || m.find(parent) == nil && parent != "" || m.isAncestor(parent, gid) {
		return false
	}
	if g.Parent == parent {
		return false
	}
	g.Parent = parent
	return true
}

// path is the group's name together with its ancestors, e.g. "db / mysql".
func (m *Meta) path(gid string) string {
	var parts []string
	for gid != "" {
		g := m.find(gid)
		if g == nil {
			break
		}
		parts = append([]string{g.Name}, parts...)
		gid = g.Parent
	}
	return strings.Join(parts, " / ")
}

func (m *Meta) remove(gid string) {
	// Members would recreate the group from their prefix on the next resolve,
	// so pin every member that has one to 未分组; the rest simply lose their entry.
	// Sub-groups are deleted with it, one level at a time.
	for _, g := range m.children(gid) {
		m.remove(g.ID)
	}
	for sid := range m.prefixed {
		if m.groupOf(sid) == gid {
			m.Assign[sid] = noGroup
		}
	}
	out := m.Groups[:0]
	for _, g := range m.Groups {
		if g.ID != gid {
			out = append(out, g)
		}
	}
	m.Groups = out
	for k, v := range m.Assign {
		if v == gid {
			delete(m.Assign, k)
		}
	}
}

// rename keeps prefix-matched members in the group even though their
// "[old]-" prefix no longer matches its name.
func (m *Meta) rename(gid, name string) {
	for sid, a := range m.auto {
		if _, explicit := m.Assign[sid]; a == gid && !explicit {
			m.Assign[sid] = gid
		}
	}
	m.find(gid).Name = strings.TrimSpace(name)
}

func (m *Meta) move(sid, gid string) {
	switch {
	case gid == m.auto[sid] && m.find(gid) != nil:
		delete(m.Assign, sid) // back where its prefix puts it
	case gid == "" && m.auto[sid] != "":
		m.Assign[sid] = noGroup
	case gid == "":
		delete(m.Assign, sid)
	default:
		m.Assign[sid] = gid
	}
}

// shift swaps a group with its previous or next sibling, so ordering is per level.
func (m *Meta) shift(gid string, d int) bool {
	g := m.find(gid)
	if g == nil {
		return false
	}
	// Positions of the siblings in the flat list, in list order.
	var slots []int
	for i := range m.Groups {
		if m.Groups[i].Parent == g.Parent || m.find(m.Groups[i].Parent) == nil && g.Parent == "" {
			slots = append(slots, i)
		}
	}
	k := -1
	for i, idx := range slots {
		if m.Groups[idx].ID == gid {
			k = i
		}
	}
	j := k + d
	if k < 0 || j < 0 || j >= len(slots) {
		return false
	}
	a, b := slots[k], slots[j]
	m.Groups[a], m.Groups[b] = m.Groups[b], m.Groups[a]
	return true
}

// prune drops assignments to snippets that no longer exist.
func (m *Meta) prune(snips []Snippet) bool {
	live := map[string]bool{}
	for _, s := range snips {
		live[s.ID] = true
	}
	changed := false
	for k := range m.Assign {
		if !live[k] {
			delete(m.Assign, k)
			changed = true
		}
	}
	return changed
}

// ---- talking to Otty ---------------------------------------------------------

// ottyCLI: Otty puts `otty` on PATH only in shells with its integration loaded;
// a view's program doesn't get it, so fall back to the CLI inside the app bundle.
func ottyCLI() string {
	if p, err := exec.LookPath("otty"); err == nil {
		return p
	}
	return "/Applications/Otty.app/Contents/MacOS/otty-cli"
}

// focusedPane is the terminal pane the user works in; "" if it is this view itself.
func focusedPane() string {
	out, err := exec.Command(ottyCLI(), "pane", "show", "--json").Output()
	if err != nil {
		return ""
	}
	var r struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(out, &r) != nil || r.Data.ID == "" {
		return ""
	}
	if strings.TrimPrefix(r.Data.ID, "p_") == strings.TrimPrefix(os.Getenv("OTTY_PANE_ID"), "p_") {
		return ""
	}
	return r.Data.ID
}

// expand fills Otty's snippet placeholders; {{cursor}} only means something in Otty's own expander.
func expand(text string) string {
	now := time.Now()
	if strings.Contains(text, "{{clipboard}}") {
		clip, _ := exec.Command("pbpaste").Output()
		text = strings.ReplaceAll(text, "{{clipboard}}", string(clip))
	}
	return strings.NewReplacer("{{date}}", now.Format("2006-01-02"),
		"{{time}}", now.Format("15:04:05"), "{{cursor}}", "").Replace(text)
}

func sendSnippet(s Snippet, run bool) string {
	pane := focusedPane()
	if pane == "" {
		return tr("send.nopane")
	}
	args := []string{"pane", "send-text", "--pane", pane, "--stdin", "--no-escape", "-q"}
	if run {
		args = append(args, "--submit=now")
	} else {
		args = append(args, "--raw-newlines")
	}
	cmd := exec.Command(ottyCLI(), args...)
	cmd.Stdin = strings.NewReader(expand(s.Text))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 7 {
			return tr("send.denied")
		}
		return fmt.Sprintf(tr("send.failed"), firstNonEmpty(strings.TrimSpace(stderr.String()), err.Error()))
	}
	if run {
		return tr("send.ran")
	}
	return tr("send.filled")
}

func copySnippet(s Snippet) string {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(expand(s.Text))
	if err := cmd.Run(); err != nil {
		return fmt.Sprintf(tr("copy.failed"), err)
	}
	return tr("copy.done")
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}
	return ""
}
