// Command otty-snippets-view is a view for Otty's details panel: it lists the
// text-snippet recipes in ~/.config/otty/recipes, organised into groups kept in
// groups.json, and sends the chosen one to the focused terminal pane.
//
// With HERDR_PANE_ID set it also runs as a Herdr plugin pane, started by
// herdr-plugin/run.sh: Herdr hands the process its invocation in environment
// variables, and snippets go to the pane the plugin was invoked for.
package main

import (
	"fmt"
	"os"

	_ "github.com/elonnzhang/otty-snippets-view/noquery" // must stay: see the package comment
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	applyLang(os.Args[1:])
	opts := []tea.ProgramOption{tea.WithMouseCellMotion()}
	if !inHerdr() {
		// Otty's panel is full screen; a Herdr sidebar is a column in the layout,
		// where the alternate screen would erase everything around it.
		opts = append(opts, tea.WithAltScreen())
	}
	p := tea.NewProgram(newModel(), opts...)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "otty-snippets-view:", err)
		os.Exit(1)
	}
}
