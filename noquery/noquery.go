// Package noquery stops lipgloss from asking the terminal for its background
// colour (OSC 11 + cursor-position report) at start-up.
//
// Bubble Tea v1 calls lipgloss.HasDarkBackground() from its own init(), which
// writes that query and then blocks for up to five seconds waiting for a reply.
// A terminal that does not answer leaves the program blank for that long, and a
// late reply can swallow the first keystroke. The details panel is one such
// terminal: Otty gives a view a tty, but a view that finds nothing listening is
// left waiting.
//
// Import order alone cannot be relied on for this: Go runs init functions in
// import-path order, and this module's path sorts after github.com/charmbracelet
// once it carries its real name. Whatever runs first here just answers the query
// early; the renderer's answer is fixed once made, so the second call is a no-op.
package noquery

import "github.com/charmbracelet/lipgloss"

func init() { lipgloss.SetHasDarkBackground(false) }
