package main

import (
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Token colors use ANSI palette indices, like the rest of the view, so they follow the Otty theme.
var (
	hlKeyword  = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	hlString   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	hlNumber   = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	hlVariable = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	hlFunc     = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	hlOperator = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

func tokenStyle(t chroma.TokenType) *lipgloss.Style {
	switch {
	case t.InCategory(chroma.Comment):
		return &stDim
	case t.InCategory(chroma.Keyword), t == chroma.NameBuiltin:
		return &hlKeyword
	case t.InSubCategory(chroma.LiteralString):
		return &hlString
	case t.InSubCategory(chroma.LiteralNumber):
		return &hlNumber
	case t == chroma.NameVariable, t.InSubCategory(chroma.NameVariable):
		return &hlVariable
	case t == chroma.NameFunction, t == chroma.NameAttribute:
		return &hlFunc
	case t.InCategory(chroma.Operator):
		return &hlOperator
	}
	return nil
}

// hlCache keeps the last highlighted body; detail() runs on every frame.
type hlCache struct {
	key   string
	lines []string
}

// highlight returns text colored by its guessed language (shell when unsure),
// hard-wrapped to w cells. Each line carries its own styling.
func (c *hlCache) highlight(text string, w int) []string {
	text = strings.ReplaceAll(text, "\t", "    ")
	key := strconv.Itoa(w) + "\x00" + text
	if c.key == key {
		return c.lines
	}
	lexer := lexers.Analyse(text)
	if lexer == nil {
		lexer = lexers.Get("bash")
	}
	it, err := chroma.Coalesce(lexer).Tokenise(nil, text)
	if err != nil {
		return strings.Split(ansi.Hardwrap(text, w, true), "\n")
	}

	var lines []string
	var cur strings.Builder
	used := 0
	flush := func() { lines = append(lines, cur.String()); cur.Reset(); used = 0 }
	for _, tok := range it.Tokens() {
		st := tokenStyle(tok.Type)
		for i, part := range strings.Split(tok.Value, "\n") {
			if i > 0 {
				flush()
			}
			for part != "" {
				chunk := ansi.Truncate(part, w-used, "")
				if chunk == "" {
					if used > 0 {
						flush()
						continue
					}
					chunk = string([]rune(part)[:1]) // wider than the whole line
				}
				part = part[len(chunk):]
				if st != nil {
					cur.WriteString(st.Render(chunk))
				} else {
					cur.WriteString(chunk)
				}
				if used += ansi.StringWidth(chunk); used >= w && part != "" {
					flush()
				}
			}
		}
	}
	flush()
	// Lexers append a final newline; drop the empty line it adds.
	if !strings.HasSuffix(text, "\n") && len(lines) > 1 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	c.key, c.lines = key, lines
	return lines
}
