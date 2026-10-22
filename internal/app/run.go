package app

import tea "github.com/charmbracelet/bubbletea"

// Run starts the view: it reads the language flag and runs the Bubble Tea
// program, in the alternate screen unless this is a Herdr sidebar pane.
func Run(args []string) error {
	applyLang(args)
	opts := []tea.ProgramOption{tea.WithMouseCellMotion()}
	if !inHerdr() {
		// Otty's panel is full screen; a Herdr sidebar is a column in the layout,
		// where the alternate screen would erase everything around it.
		opts = append(opts, tea.WithAltScreen())
	}
	_, err := tea.NewProgram(newModel(), opts...).Run()
	return err
}
