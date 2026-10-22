// Package noquery stops lipgloss from asking the terminal for its background
// colour (OSC 11 + cursor-position report) at start-up.
//
// Bubble Tea v1 calls lipgloss.HasDarkBackground() from its own init(), which
// writes that query and then blocks for up to five seconds waiting for a reply.
// A terminal that does not answer (such as a details-panel view) leaves the
// program blank for that long, and a late reply can swallow the first keystroke.
//
// Go runs package init functions in import-path order once their dependencies
// are ready, so this package must sort before "github.com/charmbracelet/bubbletea";
// that is why the module is named "a/otty-snippets-view" rather than "otty-snippets-view".
package noquery

import "github.com/charmbracelet/lipgloss"

func init() { lipgloss.SetHasDarkBackground(false) }
