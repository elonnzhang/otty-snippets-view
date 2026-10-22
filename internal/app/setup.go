package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// init writes (or with --reset removes) this binary's entry in Otty's
// `details-view` list so that it opens in the details panel. It is the Go
// counterpart of what install.sh does to the config. details-view is a list
// key, one view per line, and `otty config set` takes the whole list, so init
// reads the list, changes only the entry that is ours and writes the rest
// back as it was.
//
// The view itself cannot be started from there: a running panel keeps its old
// process until it is reopened, so init only edits the config and asks Otty to
// reload it.

const initUsage = `Write or remove this binary's entry in Otty's details-view list.

Usage:
  otty-snippets-view init [flags]

Only the entry for this binary (id="snippets", or a command that runs
otty-snippets-view) is changed; the other details views are kept as they are.

  --lang en|cn       language for the view's command (default: keep the current one)
  --print            print the details-view list that would be written, and exit
  --reset            remove this binary's entry instead of writing it
  --config-file <p>  edit this config file instead of Otty's own (skips reload)
  --no-reload        do not ask a running Otty to reload its config
`

const usage = `Usage:
  otty-snippets-view              open the view (Otty's details panel or a Herdr pane)
  otty-snippets-view init [flags] write or remove Otty's details-view entry
  otty-snippets-view --version    print the build version

Run "otty-snippets-view init -h" for the init flags.
`

// errHelp tells runInit that the usage was already handled.
var errHelp = errors.New("help")

type initOpts struct {
	lang       string // "" keeps the language the current entry uses
	configFile string
	print      bool
	reset      bool
	noReload   bool
}

func runInit(args []string) error {
	o, err := parseInitArgs(args)
	if err == errHelp {
		fmt.Print(initUsage)
		return nil
	}
	if err != nil {
		return err
	}

	cur, err := ottyConfig(o.configFile, "get", "details-view")
	if err != nil {
		return err
	}
	views := detailsViews(cur)

	if o.reset {
		return resetDetailsView(o, views)
	}
	next := withDetailsView(views, o.lang)
	if o.print {
		printDetailsViews(next)
		return nil
	}
	if err := writeDetailsViews(o.configFile, next); err != nil {
		return err
	}
	for _, v := range next {
		if detailsViewOurs(v) {
			fmt.Printf("details-view = %s\n", v)
		}
	}
	noteKept(views)
	reloadOtty(o)
	fmt.Println("\nOpen the details panel (⌘⇧R twice) to use it.")
	noteSendKeys(o.configFile)
	return nil
}

// resetDetailsView removes our entries and keeps the other views.
func resetDetailsView(o initOpts, views []string) error {
	var rest []string
	for _, v := range views {
		if !detailsViewOurs(v) {
			rest = append(rest, v)
		}
	}
	if len(rest) == len(views) {
		fmt.Println("details-view has no entry for otty-snippets-view")
		return nil
	}
	if o.print {
		printDetailsViews(rest)
		return nil
	}
	if err := writeDetailsViews(o.configFile, rest); err != nil {
		return err
	}
	fmt.Println("removed the otty-snippets-view entry from details-view")
	noteKept(views)
	reloadOtty(o)
	return nil
}

// detailsViews splits `otty config get details-view` output into its entries,
// which Otty prints one per line.
func detailsViews(out string) []string {
	var views []string
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			views = append(views, l)
		}
	}
	return views
}

// writeDetailsViews stores views as the whole details-view list. An empty list
// removes the key rather than leaving an empty entry behind.
func writeDetailsViews(configFile string, views []string) error {
	var err error
	if len(views) == 0 {
		_, err = ottyConfig(configFile, "unset", "details-view")
	} else {
		_, err = ottyConfig(configFile, "set", "details-view", strings.Join(views, "\n"))
	}
	return err
}

// printDetailsViews prints views the way `otty config get` does, so the output
// can be handed back to `otty config set details-view`.
func printDetailsViews(views []string) {
	if len(views) > 0 {
		fmt.Println(strings.Join(views, "\n"))
	}
}

// noteKept says how many other views were left in place.
func noteKept(views []string) {
	n := 0
	for _, v := range views {
		if !detailsViewOurs(v) {
			n++
		}
	}
	if n > 0 {
		fmt.Printf("kept the other %d details-view entries\n", n)
	}
}

func parseInitArgs(args []string) (initOpts, error) {
	var o initOpts
	for i := 0; i < len(args); i++ {
		a := args[i]
		value := func(name string) (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("%s needs a value", name)
			}
			i++
			return args[i], nil
		}
		switch {
		case a == "--lang" || a == "-lang":
			v, err := value(a)
			if err != nil {
				return o, err
			}
			if o.lang, err = initLang(v); err != nil {
				return o, err
			}
		case strings.HasPrefix(a, "--lang="):
			var err error
			if o.lang, err = initLang(strings.TrimPrefix(a, "--lang=")); err != nil {
				return o, err
			}
		case a == "--config-file":
			v, err := value(a)
			if err != nil {
				return o, err
			}
			o.configFile = v
		case a == "--print":
			o.print = true
		case a == "--reset" || a == "--remove":
			o.reset = true
		case a == "--no-reload":
			o.noReload = true
		case a == "-h" || a == "--help":
			return o, errHelp
		default:
			return o, fmt.Errorf("unknown option: %s", a)
		}
	}
	return o, nil
}

func initLang(v string) (string, error) {
	l, ok := parseLang(v)
	if !ok {
		return "", fmt.Errorf("unsupported --lang %q (use en or cn)", v)
	}
	return l, nil
}

var (
	// command= inside one entry. Otty only accepts double-quoted field values.
	commandRE = regexp.MustCompile(`command="[^"]*"`)
	// --lang en or --lang=en inside the command.
	langRE = regexp.MustCompile(`--lang[= ]([^\s"']+)`)
)

// withDetailsView is views with this binary's entry brought up to date: each
// entry of ours gets the command for lang, in place, and a fresh entry goes at
// the end when there is none. The other views are kept as they are. lang ""
// keeps the language our entry passes.
func withDetailsView(views []string, lang string) []string {
	if lang == "" {
		lang = "en"
		for _, v := range views {
			if detailsViewOurs(v) {
				lang = currentLang(v)
				break
			}
		}
	}
	cmd := "command=" + tomlString(commandFor(configLang(lang)))
	fresh := `id="snippets" kind="tui" title="Snippets" icon="command" ` + cmd + ` mobile="false"`

	next := make([]string, 0, len(views)+1)
	found := false
	for _, v := range views {
		if detailsViewOurs(v) {
			found = true
			if commandRE.MatchString(v) {
				v = commandRE.ReplaceAllLiteralString(v, cmd)
			} else {
				v = fresh
			}
		}
		next = append(next, v)
	}
	if !found {
		next = append(next, fresh)
	}
	return next
}

// configLang is the spelling written into Otty's config, en or cn, the way
// install.sh writes it. parseLang returns "zh" for Chinese internally.
func configLang(l string) string {
	if l == "zh" {
		return "cn"
	}
	return l
}

// detailsViewOurs reports whether a details-view entry is this tool's, matched
// the same ways install.sh does.
func detailsViewOurs(v string) bool {
	return strings.Contains(v, "otty-snippets-view") || strings.Contains(v, `id="snippets"`)
}

// currentLang is the language the current entry's command passes, "en" when it
// has none or an unknown one.
func currentLang(v string) string {
	m := langRE.FindStringSubmatch(v)
	if m == nil {
		return "en"
	}
	if l, ok := parseLang(m[1]); ok {
		return l
	}
	return "en"
}

// commandFor is the command Otty should run: this binary, by the path it was
// started from with symlinks resolved, and the language flag.
func commandFor(lang string) string {
	exe, err := os.Executable()
	if err != nil || exe == "" {
		exe = os.Args[0]
	}
	if p, err := filepath.EvalSymlinks(exe); err == nil && p != "" {
		exe = p
	}
	return exe + " --lang " + lang
}

// tomlString quotes s as a TOML basic string.
func tomlString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// ottyConfig runs `otty [--config-file p] config <args...>` and returns its
// output. An empty config file means Otty's own.
func ottyConfig(configFile string, args ...string) (string, error) {
	full := append([]string{"config"}, args...)
	if configFile != "" {
		full = append([]string{"--config-file", configFile}, full...)
	}
	cmd := exec.Command(ottyCLI(), full...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err == nil {
		return string(out), nil
	}
	if msg := strings.TrimSpace(stderr.String()); msg != "" {
		return "", fmt.Errorf("otty %s: %s", strings.Join(args, " "), msg)
	}
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("otty is not on PATH (looked for %s): install Otty or put its CLI on PATH", ottyCLI())
	}
	return "", fmt.Errorf("otty %s: %w", strings.Join(args, " "), err)
}

// reloadOtty asks a running Otty to pick up the config it just gained.
func reloadOtty(o initOpts) {
	if o.noReload || o.configFile != "" {
		return
	}
	if _, err := ottyConfig("", "reload"); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not ask Otty to reload: %v\n", err)
	}
}

// noteSendKeys repeats install.sh's reminder: sending to a pane is the user's
// security switch, so the command never turns it on.
func noteSendKeys(configFile string) {
	v, err := ottyConfig(configFile, "get", "ipc-allow-send-keys")
	if err == nil && strings.TrimSpace(v) == "true" {
		return
	}
	fmt.Println("\nnote: ipc-allow-send-keys is off, so run/fill will answer \"rejected\".")
	fmt.Println("      Turning it on is your decision: set ipc-allow-send-keys = true in the Otty config.")
}
