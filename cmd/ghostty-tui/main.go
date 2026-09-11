package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"ghostty.studio/internal/core"
	"ghostty.studio/internal/ui"
)

var version = "0.2.0"

const usage = `Ghostty Studio — make the terminal yours

  ghostty-tui                               Open the studio
  ghostty-tui themes [query]                Browse installed palettes
  ghostty-tui fonts [--discover]            List available font families
  ghostty-tui fonts --install FONT          Install one discovery item
  ghostty-tui diff --theme NAME             Inspect a proposed change
  ghostty-tui apply --theme NAME            Validate, back up, and save
  ghostty-tui apply --font NAME             Select a font, retain fallbacks
  ghostty-tui apply --look NAME             Apply a composed Look
  ghostty-tui apply --set KEY=VALUE          Set one or more values
  ghostty-tui undo                          Restore the previous change
  ghostty-tui preview --theme NAME          Open a native Ghostty specimen
  ghostty-tui doctor                        Inspect configuration health
  ghostty-tui look list                     List composed Looks
  ghostty-tui look save NAME                Save current appearance
  ghostty-tui look export NAME FILE         Export a Look as JSON
  ghostty-tui look import FILE              Add an exported Look to the library
  ghostty-tui export --format fzf           Print a palette (fzf/ghostty/json)
  ghostty-tui snapshot --width 120 --height 36

Global options go before the command:
  --config FILE       Manage an explicit config (default: discover Ghostty config)
  --state-dir DIR     Keep history, Looks, and favorites in this directory
  --ghostty FILE      Use this Ghostty executable
  --assets DIR       Use this asset directory
  --version          Print the installed Studio version

Selections are staged in the TUI. Apply saves; Ghostty reloads with Cmd+Shift+,.
Fonts and some window settings may require a new terminal or restart.
`

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "ghostty-tui:", err)
		os.Exit(1)
	}
}

func flags(name string, stderr io.Writer) *flag.FlagSet {
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(stderr)
	return f
}

func run(args []string, out, stderr io.Writer) error {
	opts := core.Options{}
	f := flags("ghostty-tui", stderr)
	f.StringVar(&opts.ConfigPath, "config", "", "Ghostty configuration file")
	f.StringVar(&opts.StateDir, "state-dir", "", "Studio state directory")
	f.StringVar(&opts.GhosttyPath, "ghostty", "", "Ghostty executable")
	f.StringVar(&opts.AssetDir, "assets", "", "Bundled assets directory")
	showVersion := f.Bool("version", false, "Print version")
	f.Usage = func() { fmt.Fprint(out, usage) }
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *showVersion {
		fmt.Fprintln(out, "Ghostty Studio", version)
		return nil
	}
	args = f.Args()
	if len(args) > 0 && args[0] == "help" {
		fmt.Fprint(out, usage)
		return nil
	}
	// A native specimen must use ANSI colors from its own Ghostty window, not
	// RGB colors fetched from the user's unrelated default configuration.
	if len(args) > 0 && args[0] == "specimen" {
		return specimen(out)
	}
	e, err := core.Open(opts)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		_, err := tea.NewProgram(ui.New(e, opts)).Run()
		return err
	}
	command, rest := args[0], args[1:]
	switch command {
	case "themes":
		return printChoices(out, e.Catalog("themes"), strings.Join(rest, " "))
	case "fonts":
		ff := flags(command, stderr)
		discover := ff.Bool("discover", false, "Show installable fonts")
		install := ff.String("install", "", "Install one font name or cask")
		if err := ff.Parse(rest); err != nil {
			return err
		}
		if *install != "" {
			result, err := e.InstallFont(*install)
			return printResult(out, result, err)
		}
		section := "fonts"
		if *discover {
			section = "discover"
		}
		return printChoices(out, e.Catalog(section), strings.Join(ff.Args(), " "))
	case "apply", "diff", "preview":
		change, err := parseChange(e, rest, stderr)
		if err != nil {
			return err
		}
		if command == "diff" {
			fmt.Fprint(out, e.Diff(change))
			return nil
		}
		if command == "preview" {
			r, err := e.Preview(change)
			return printResult(out, r, err)
		}
		r, err := e.Apply(change)
		return printResult(out, r, err)
	case "undo":
		if len(rest) != 0 {
			return errors.New("undo takes no arguments")
		}
		r, err := e.Undo()
		return printResult(out, r, err)
	case "doctor":
		for _, d := range e.Diagnose() {
			fmt.Fprintf(out, "%s  %s\n    %s\n", strings.ToUpper(d.Level), d.Title, d.Detail)
		}
		return nil
	case "look":
		return looks(e, rest, out)
	case "export":
		xf := flags(command, stderr)
		format := xf.String("format", "ghostty", "fzf, ghostty, or json")
		if err := xf.Parse(rest); err != nil {
			return err
		}
		if len(xf.Args()) != 0 {
			return errors.New("export prints to stdout; use shell redirection for a file")
		}
		text, err := e.ExportTheme(e.State().Current, *format)
		if err == nil {
			fmt.Fprint(out, text)
		}
		return err
	case "snapshot":
		sf := flags(command, stderr)
		width := sf.Int("width", 120, "Terminal columns")
		height := sf.Int("height", 36, "Terminal rows")
		section := sf.String("section", "themes", "themes, fonts, looks, effects, settings, workflows, health")
		query := sf.String("query", "", "Search the displayed collection")
		if err := sf.Parse(rest); err != nil {
			return err
		}
		if *width < 1 || *height < 1 || *width > 500 || *height > 200 {
			return errors.New("snapshot size must be 1–500 columns and 1–200 rows")
		}
		fmt.Fprint(out, ui.Snapshot(e, opts, *width, *height, *section, *query))
		return nil
	default:
		return fmt.Errorf("unknown command %q; run ghostty-tui --help", command)
	}
}

func printResult(out io.Writer, r core.Result, err error) error {
	if err != nil {
		return err
	}
	fmt.Fprintln(out, r.Message)
	if r.BackupPath != "" {
		fmt.Fprintln(out, "Backup:", r.BackupPath)
	}
	for _, w := range r.Warnings {
		fmt.Fprintln(out, "Note:", w)
	}
	return nil
}

func printChoices(out io.Writer, choices []core.Choice, query string) error {
	count := 0
	for _, c := range choices {
		if query != "" && !strings.Contains(strings.ToLower(c.Name+" "+c.ID+" "+strings.Join(c.Tags, " ")), strings.ToLower(query)) {
			continue
		}
		fmt.Fprintf(out, "%s\n  %s\n", c.Name, c.Description)
		count++
	}
	if count == 0 {
		return fmt.Errorf("no matches for %q", query)
	}
	return nil
}

type repeated []string

func (r *repeated) String() string     { return strings.Join(*r, ", ") }
func (r *repeated) Set(s string) error { *r = append(*r, s); return nil }

func parseChange(e *core.Engine, args []string, stderr io.Writer) (core.Change, error) {
	f := flags("change", stderr)
	theme := f.String("theme", "", "Theme name or ID")
	font := f.String("font", "", "Installed font name or ID")
	look := f.String("look", "", "Look name or ID")
	var sets repeated
	f.Var(&sets, "set", "KEY=VALUE (repeatable)")
	if err := f.Parse(args); err != nil {
		return core.Change{}, err
	}
	if len(f.Args()) != 0 {
		return core.Change{}, fmt.Errorf("unexpected arguments: %s", strings.Join(f.Args(), " "))
	}
	values := core.Values{}
	var labels []string
	for _, pair := range [][2]string{{"looks", *look}, {"themes", *theme}, {"fonts", *font}} {
		if pair[1] == "" {
			continue
		}
		c, err := findChoice(e.Catalog(pair[0]), pair[1])
		if err != nil {
			return core.Change{}, err
		}
		for k, v := range c.Values {
			values[k] = append([]string(nil), v...)
		}
		labels = append(labels, c.Name)
	}
	seen := map[string]bool{}
	for _, s := range sets {
		k, v, ok := strings.Cut(s, "=")
		k = strings.TrimSpace(k)
		if !ok || k == "" {
			return core.Change{}, fmt.Errorf("expected --set KEY=VALUE, got %q", s)
		}
		if !seen[k] {
			values[k] = nil
			if core.IsRepeatable(k) && v != "" {
				values[k] = []string{""}
			}
			seen[k] = true
		}
		values[k] = append(values[k], v)
		labels = append(labels, k)
	}
	if len(values) == 0 {
		return core.Change{}, errors.New("choose --theme, --font, --look, or --set KEY=VALUE")
	}
	return core.Change{Values: values, Revision: e.State().Revision, Label: strings.Join(labels, " + ")}, nil
}

func findChoice(choices []core.Choice, name string) (core.Choice, error) {
	var matches []core.Choice
	for _, c := range choices {
		if strings.EqualFold(c.ID, name) || strings.EqualFold(c.Name, name) {
			return c, nil
		}
		if strings.Contains(strings.ToLower(c.Name), strings.ToLower(name)) {
			matches = append(matches, c)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) == 0 {
		return core.Choice{}, fmt.Errorf("%q is not in this collection", name)
	}
	var names []string
	for _, c := range matches {
		names = append(names, c.Name)
		if len(names) == 8 {
			break
		}
	}
	return core.Choice{}, fmt.Errorf("%q is ambiguous: %s", name, strings.Join(names, ", "))
}

func looks(e *core.Engine, args []string, out io.Writer) error {
	if len(args) == 0 || args[0] == "list" {
		return printChoices(out, e.Catalog("looks"), "")
	}
	switch args[0] {
	case "save":
		if len(args) != 2 {
			return errors.New("usage: look save NAME")
		}
		if err := e.SaveLook(args[1], e.State().Current); err != nil {
			return err
		}
		fmt.Fprintln(out, "Saved Look:", args[1])
		return nil
	case "export":
		if len(args) != 3 {
			return errors.New("usage: look export NAME FILE")
		}
		c, err := findChoice(e.Catalog("looks"), args[1])
		if err != nil {
			return err
		}
		if err = e.ExportLook(c.Name, c.Values, expandPath(args[2])); err != nil {
			return err
		}
		fmt.Fprintln(out, "Exported:", expandPath(args[2]))
		return nil
	case "import":
		if len(args) != 2 {
			return errors.New("usage: look import FILE")
		}
		c, err := e.ImportLook(expandPath(args[1]))
		if err != nil {
			return err
		}
		if err = e.SaveLook(c.Name, c.Values); err != nil {
			return err
		}
		fmt.Fprintln(out, "Imported Look:", c.Name)
		return nil
	default:
		return errors.New("look commands: list, save, export, import")
	}
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func specimen(out io.Writer) error {
	color := func(n int, s string) string { return fmt.Sprintf("\x1b[%dm%s\x1b[0m", n, s) }
	fmt.Fprintln(out, "\n  "+color(1, "Ghostty Studio")+"   /   native specimen\n")
	fmt.Fprint(out, "  This window uses the chosen font, palette, and effects.\n\n")
	fmt.Fprintln(out, "  "+color(36, "package")+" atelier\n")
	fmt.Fprintln(out, "  "+color(90, "// A small place to make things."))
	fmt.Fprintln(out, "  "+color(35, "func")+" greeting() "+color(36, "string")+" {")
	fmt.Fprintln(out, "      "+color(35, "return")+" "+color(32, "\"Hello, beautifully.\""))
	fmt.Fprint(out, "  }\n\n")
	fmt.Fprintln(out, "  "+color(31, "- theme: yesterday")+"\n  "+color(32, "+ theme: yours")+"\n")
	fmt.Fprintln(out, "  "+color(32, "~/atelier")+" "+color(90, "main")+" ❯ "+color(37, "go test ./..."))
	fmt.Fprintln(out, "  "+color(32, "✓")+" "+color(90, "sample output · ready to make something\n"))
	fmt.Fprintln(out, "  0 O o   1 l I |   { [ (   != == => <= >=   www mmm iii")
	fmt.Fprintln(out, "  "+color(1, "Bold / 0O1lI")+"   "+color(3, "Italic / fjkt")+"   Regular / AaBbCc")
	fmt.Fprintln(out, "  ┌──────────┐   → ← ↑ ↓   λ ∑ π   日本語 中文   ⚑  ☀  ✦")
	fmt.Fprintln(out, "  └──────────┘")
	fmt.Fprint(out, "\n  ")
	for n := 0; n < 16; n++ {
		code := 40 + n
		if n >= 8 {
			code = 100 + n - 8
		}
		fmt.Fprint(out, color(code, "   "))
	}
	fmt.Fprint(out, "\n\n")
	if info, err := os.Stdin.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 {
		fmt.Fprintln(out, "  Press Enter to close this specimen.")
		_, err = bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
	}
	return nil
}
