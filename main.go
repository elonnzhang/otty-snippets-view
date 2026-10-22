// Command otty-snippets-view is the Snippets view for Otty's details panel: it lists
// the text-snippet recipes in ~/.config/otty/recipes, organised into groups
// kept in groups.json, and sends the chosen one to the focused terminal pane.
package main

import (
	"fmt"
	"os"

	_ "a/otty-snippets-view/noquery" // must stay: see the package comment
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	applyLang(os.Args[1:])
	p := tea.NewProgram(newModel(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "otty-snippets-view:", err)
		os.Exit(1)
	}
}
