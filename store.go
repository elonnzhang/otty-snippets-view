package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml/v2"
)

const (
	recipeExt  = ".ottyrecipe"
	fileHeader = "# Otty recipe — edit paths/commands freely; the terminal will apply on open.\n\n"
)

// Snippet is one text-snippet recipe. Foreign is set when the file holds
// fields this tool does not know; such files are never rewritten.
type Snippet struct {
	ID, Name, Alias, Text, Content string
	Foreign                        bool
}

func home() string { h, _ := os.UserHomeDir(); return h }

func recipesDir() string {
	if d := os.Getenv("OTTY_RECIPES_DIR"); d != "" {
		return d
	}
	return filepath.Join(home(), ".config/otty/recipes")
}

func viewDir() string {
	if d := os.Getenv("SNIPPETS_HOME"); d != "" {
		return d
	}
	return filepath.Join(home(), ".config/otty/views/otty-snippets-view")
}

func recipePath(id string) string { return filepath.Join(recipesDir(), id+recipeExt) }

func str(v any) string { s, _ := v.(string); return s }

func parseSnippet(path string) (Snippet, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Snippet{}, false
	}
	var top map[string]any
	if toml.Unmarshal(raw, &top) != nil {
		return Snippet{}, false
	}
	rec, _ := top["recipe"].(map[string]any)
	snip, _ := top["snippet"].(map[string]any)
	text, ok := snip["text"].(string)
	if rec == nil || !ok || str(rec["scope"]) != "text_snippet" {
		return Snippet{}, false
	}
	known := map[string]bool{"name": true, "version": true, "scope": true, "content": true, "alias": true}
	foreign := len(top) != 2 || len(snip) != 1
	for k := range rec {
		foreign = foreign || !known[k]
	}
	if v, _ := rec["version"].(int64); v != 1 {
		foreign = true
	}
	id := strings.TrimSuffix(filepath.Base(path), recipeExt)
	s := Snippet{ID: id, Name: str(rec["name"]), Alias: str(rec["alias"]), Text: text,
		Content: str(rec["content"]), Foreign: foreign}
	if s.Name == "" {
		s.Name = id
	}
	if s.Content == "" {
		s.Content = "layout"
	}
	return s, true
}

func loadSnippets() []Snippet {
	paths, _ := filepath.Glob(filepath.Join(recipesDir(), "*"+recipeExt))
	var out []Snippet
	for _, p := range paths {
		if s, ok := parseSnippet(p); ok {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := strings.ToLower(out[i].Name), strings.ToLower(out[j].Name)
		return a < b || (a == b && out[i].ID < out[j].ID)
	})
	return out
}

// signature is a cheap change detector for the recipes directory.
func signature() string {
	paths, _ := filepath.Glob(filepath.Join(recipesDir(), "*"+recipeExt))
	var b strings.Builder
	for _, p := range paths {
		if st, err := os.Stat(p); err == nil {
			fmt.Fprintf(&b, "%s:%d;", p, st.ModTime().UnixNano())
		}
	}
	return b.String()
}

// tomlStr quotes s the way Otty's own encoder does: a basic string with escapes.
func tomlStr(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '\\':
			b.WriteString(`\\`)
		case r == '"':
			b.WriteString(`\"`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&b, `\u%04X`, r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6], b[8] = b[6]&0x0f|0x40, b[8]&0x3f|0x80
	return strings.ToUpper(fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]))
}

// saveSnippet writes a new recipe (empty ID) or rewrites an existing one, atomically.
func saveSnippet(s Snippet) (string, error) {
	if s.Foreign {
		return "", fmt.Errorf("%s", tr("foreign"))
	}
	if s.ID == "" {
		s.ID = newID()
	} else if _, err := os.Stat(recipePath(s.ID)); err != nil {
		return "", err
	}
	if s.Content == "" {
		s.Content = "layout"
	}
	body := fileHeader + "[recipe]\n" +
		"name = " + tomlStr(s.Name) + "\n" +
		"version = 1\n" +
		"scope = \"text_snippet\"\n" +
		"content = " + tomlStr(s.Content) + "\n" +
		"alias = " + tomlStr(s.Alias) + "\n\n" +
		"[snippet]\n" +
		"text = " + tomlStr(s.Text) + "\n"
	if err := os.MkdirAll(recipesDir(), 0o755); err != nil {
		return "", err
	}
	tmp := filepath.Join(recipesDir(), "."+s.ID+".tmp")
	if err := os.WriteFile(tmp, []byte(body), 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, recipePath(s.ID)); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return s.ID, nil
}

// trashSnippet moves a recipe into our own trash folder so a delete can be undone by hand.
func trashSnippet(id string) error {
	dir := filepath.Join(viewDir(), "trash")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	name := time.Now().Format("20060102-150405") + "-" + id + recipeExt
	return os.Rename(recipePath(id), filepath.Join(dir, name))
}

// ---- group metadata ----------------------------------------------------------

type Group struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Collapsed bool   `json:"collapsed"`
	Parent    string `json:"parent,omitempty"` // "" = top level
}

// UIState is what the view remembers between runs.
type UIState struct {
	Tab          string `json:"tab,omitempty"`           // "list" or "group"
	DetailHeight int    `json:"detail_height,omitempty"` // rows below the divider
}

type Meta struct {
	Version int               `json:"version"`
	Groups  []Group           `json:"groups"`
	Assign  map[string]string `json:"assign"`
	UI      UIState           `json:"ui"`

	auto     map[string]string // snippet id -> group from its "[group]-" name prefix
	prefixed map[string]bool   // snippets whose name has such a prefix
}

func metaPath() string { return filepath.Join(viewDir(), "groups.json") }

func loadMeta() *Meta {
	m := &Meta{Version: 1, Assign: map[string]string{}, auto: map[string]string{}}
	raw, err := os.ReadFile(metaPath())
	if err != nil {
		return m
	}
	var got Meta
	if json.Unmarshal(raw, &got) != nil {
		// Unreadable metadata: keep it as evidence instead of overwriting it on the next save.
		_ = os.Rename(metaPath(), metaPath()+".bad")
		return m
	}
	m.Groups, m.UI = got.Groups, got.UI
	if got.Assign != nil {
		m.Assign = got.Assign
	}
	return m
}

func (m *Meta) save() error {
	if m.Groups == nil {
		m.Groups = []Group{}
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(viewDir(), 0o755); err != nil {
		return err
	}
	tmp := metaPath() + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, metaPath())
}
