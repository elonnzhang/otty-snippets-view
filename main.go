// Command otty-snippets-view is a view for Otty's details panel: it lists the
// text-snippet recipes in ~/.config/otty/recipes, organised into groups kept in
// groups.json, and sends the chosen one to the focused terminal pane.
//
// With HERDR_PANE_ID set it also runs as a Herdr plugin pane, started by
// herdr-plugin/run.sh: Herdr hands the process its invocation in environment
// variables, and snippets go to the pane the plugin was invoked for.
//
// The view itself lives in internal/app; this file only starts it.
package main

import (
	"fmt"
	"os"

	"github.com/elonnzhang/otty-snippets-view/internal/app"
	_ "github.com/elonnzhang/otty-snippets-view/internal/noquery" // must stay: see the package comment
)

func main() {
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "otty-snippets-view:", err)
		os.Exit(1)
	}
}
