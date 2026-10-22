package app

import (
	"fmt"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"
)

// Run starts the view: it reads the language flag and runs the Bubble Tea
// program, in the alternate screen unless this is a Herdr sidebar pane. The
// init subcommand edits Otty's config instead, and --version prints the build.
func Run(args []string) error {
	if len(args) > 0 && args[0] == "init" {
		return runInit(args[1:])
	}
	if wantsVersion(args) {
		fmt.Println(version())
		return nil
	}
	if wantsHelp(args) {
		fmt.Print(usage)
		return nil
	}
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

// wantsVersion reports whether args ask for the version instead of the view.
func wantsVersion(args []string) bool {
	for _, a := range args {
		switch a {
		case "--version", "-version", "-V":
			return true
		}
	}
	return false
}

func wantsHelp(args []string) bool {
	for _, a := range args {
		if a == "-h" || a == "--help" {
			return true
		}
	}
	return false
}

// version is what this binary was built from: the module version when it came
// from `go install …@vX.Y.Z`, or dev-<revision> for a build from a checkout.
// `go install` does not pass -ldflags, so this reads the build info Go embeds
// in every binary.
func version() string {
	v, rev := "", ""
	if bi, ok := debug.ReadBuildInfo(); ok {
		if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			v = bi.Main.Version
		}
		for _, s := range bi.Settings {
			if s.Key == "vcs.revision" {
				rev = s.Value
			}
		}
	}
	if v == "" {
		v = "dev"
		if len(rev) >= 7 {
			v += "-" + rev[:7]
		}
	}
	return "otty-snippets-view " + v
}
