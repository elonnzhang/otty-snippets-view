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
