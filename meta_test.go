package main

import (
	"os"
	"testing"
)

func TestSplitPrefix(t *testing.T) {
	cases := []struct {
		in, g, rest string
		ok          bool
	}{
		{"[mysql]-login", "mysql", "login", true},
		{"[ db ] - dump", "db", "dump", true},
		{"[工作]查询", "工作", "查询", true},
		{"relay", "", "relay", false},
		{"[]-x", "", "[]-x", false},
		{"a [b]-c", "", "a [b]-c", false},
	}
	for _, c := range cases {
		g, rest, ok := splitPrefix(c.in)
		if g != c.g || rest != c.rest || ok != c.ok {
			t.Errorf("splitPrefix(%q) = %q,%q,%v; want %q,%q,%v", c.in, g, rest, ok, c.g, c.rest, c.ok)
		}
	}
}

func TestAutoGroup(t *testing.T) {
	t.Setenv("SNIPPETS_HOME", t.TempDir())
	snips := []Snippet{{ID: "a", Name: "[mysql]-login"}, {ID: "b", Name: "[MySQL]-dump"}, {ID: "c", Name: "plain"}}
	m := loadMeta()
	if !m.resolve(snips) || len(m.Groups) != 1 {
		t.Fatalf("want one group created, got %+v", m.Groups)
	}
	gid := m.Groups[0].ID
	if m.groupOf("a") != gid || m.groupOf("b") != gid || m.groupOf("c") != "" {
		t.Fatalf("auto assignment wrong: a=%q b=%q c=%q", m.groupOf("a"), m.groupOf("b"), m.groupOf("c"))
	}

	// Dragging out to 未分组 sticks across a re-resolve.
	m.move("a", "")
	m.resolve(snips)
	if m.groupOf("a") != "" {
		t.Fatalf("explicit 未分组 lost: %q", m.groupOf("a"))
	}
	// Moving back to the prefix group clears the override.
	m.move("a", gid)
	if _, ok := m.Assign["a"]; ok || m.groupOf("a") != gid {
		t.Fatalf("move back should drop override: %+v", m.Assign)
	}

	// Renaming keeps members.
	m.rename(gid, "数据库")
	m.resolve(snips)
	if len(m.Groups) != 1 || m.groupOf("a") != gid || m.groupOf("b") != gid {
		t.Fatalf("rename lost members or recreated group: %+v %+v", m.Groups, m.Assign)
	}

	// Deleting does not resurrect the group from prefixes.
	m.remove(gid)
	if m.resolve(snips) || len(m.Groups) != 0 || m.groupOf("a") != "" {
		t.Fatalf("deleted group came back: %+v %+v", m.Groups, m.Assign)
	}

	// Metadata round-trips, including the UI state.
	m.UI = UIState{Tab: "group", DetailHeight: 7}
	if err := m.save(); err != nil {
		t.Fatal(err)
	}
	got := loadMeta()
	if got.UI != m.UI || len(got.Assign) != len(m.Assign) {
		t.Fatalf("round trip: %+v vs %+v", got, m)
	}
	_ = os.Remove(metaPath())
}

func TestLang(t *testing.T) {
	for _, c := range []struct {
		in, want string
		ok       bool
	}{
		{"en", "en", true}, {"CN", "zh", true}, {"cn", "zh", true},
		{"zh-CN", "zh", true}, {"中文", "zh", true}, {"fr", "", false},
	} {
		if got, ok := parseLang(c.in); got != c.want || ok != c.ok {
			t.Errorf("parseLang(%q) = %q,%v", c.in, got, ok)
		}
	}
	lang = "en"
	if tabNames()[0] != "List" || ungroupedName() != "Ungrouped" {
		t.Errorf("en strings: %v %q", tabNames(), ungroupedName())
	}
	lang = "zh"
	if tabNames()[0] != "列表" || ungroupedName() != "未分组" {
		t.Errorf("zh strings: %v %q", tabNames(), ungroupedName())
	}
	if !isUngroupedLabel("未分组") || !isUngroupedLabel("ungrouped") || isUngroupedLabel("mysql") {
		t.Error("isUngroupedLabel")
	}
	lang = "en"
}

func TestNestedGroups(t *testing.T) {
	t.Setenv("SNIPPETS_HOME", t.TempDir())
	m := loadMeta()
	db := m.add("db")
	my := m.addUnder("mysql", db)
	ops := m.add("ops")
	_ = ops

	if !m.setParent(my, ops) { // mysql under ops
		t.Fatal("setParent refused a valid move")
	}
	if m.find(my).Parent != ops {
		t.Fatalf("parent = %q, want %q", m.find(my).Parent, ops)
	}
	if m.path(my) != "ops / mysql" {
		t.Fatalf("path = %q", m.path(my))
	}
	// Cycles are refused: ops under mysql would loop.
	if m.setParent(ops, my) || m.find(ops).Parent != "" {
		t.Fatal("cycle accepted")
	}
	if !m.isAncestor(my, ops) || m.isAncestor(ops, my) {
		t.Fatal("isAncestor wrong")
	}
	// Back to the top level.
	if !m.setParent(my, "") || m.find(my).Parent != "" {
		t.Fatal("could not move back to top")
	}
	// A missing parent is ignored.
	if m.setParent(db, "nope") {
		t.Fatal("accepted an unknown parent")
	}

	// Snippets follow their group's subtree and deleting a parent deletes the children.
	m.setParent(my, db)
	snips := []Snippet{{ID: "s1", Name: "[mysql]-q"}}
	m.resolve(snips)
	for _, sid := range []string{"s1"} {
		if m.path(m.groupOf(sid)) != "db / mysql" {
			t.Fatalf("snippet path = %q", m.path(m.groupOf(sid)))
		}
	}
	if m.descendants(db) != 1 {
		t.Fatalf("descendants(db) = %d", m.descendants(db))
	}
	m.remove(db)
	if m.find(my) != nil || m.find(db) != nil {
		t.Fatalf("children survived: %+v", m.Groups)
	}
	if m.groupOf("s1") != "" {
		t.Fatalf("member of a deleted subtree still grouped")
	}
}

func TestSiblingShift(t *testing.T) {
	t.Setenv("SNIPPETS_HOME", t.TempDir())
	m := loadMeta()
	a, b := m.add("a"), m.add("b")
	c := m.addUnder("c", a) // child of a, so a sibling of b only at the top level
	if !m.shift(b, -1) || m.children("")[0].ID != b || m.children("")[1].ID != a {
		t.Fatalf("shift up failed: %+v", m.children(""))
	}
	if m.shift(b, -1) {
		t.Fatal("shifted past the first sibling")
	}
	if m.shift(c, 1) || m.shift(c, -1) {
		t.Fatal("only child should not move")
	}
	if m.shift(a, 1) {
		t.Fatal("last sibling shifted down")
	}
}
