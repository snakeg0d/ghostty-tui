package ui

import "charm.land/bubbles/v2/key"

// keys holds every user-visible binding. Each one is handled by the root
// update loop; the help overlay is generated from the same values.
type keys struct {
	Stage   key.Binding
	Apply   key.Binding
	Undo    key.Binding
	Preview key.Binding
	Search  key.Binding
	Help    key.Binding
	Quit    key.Binding
	Tune    key.Binding
	Inspect key.Binding

	Next key.Binding
	Prev key.Binding

	Up     key.Binding
	Down   key.Binding
	Top    key.Binding
	Bottom key.Binding
	PgDown key.Binding
	PgUp   key.Binding

	Fav      key.Binding
	Filter   key.Binding
	Pin      key.Binding
	Discover key.Binding
	Install  key.Binding
	Save     key.Binding
	Export   key.Binding
	Import   key.Binding
	Toggle   key.Binding
	Clear    key.Binding
	Refresh  key.Binding
	PanePrev key.Binding
	PaneNext key.Binding

	SectionThemes key.Binding
	SectionFonts  key.Binding
	SectionLooks  key.Binding
	SectionMore   key.Binding
}

func newKeys() keys {
	return keys{
		Stage:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "stage")),
		Apply:   key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "apply")),
		Undo:    key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "undo")),
		Preview: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "native preview")),
		Search:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:    key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		Tune:    key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "tune")),
		Inspect: key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "inspect")),

		Next: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next section")),
		Prev: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev section")),

		Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("j/k", "move")),
		Down:   key.NewBinding(key.WithKeys("down", "j")),
		Top:    key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g/G", "top/bottom")),
		Bottom: key.NewBinding(key.WithKeys("G", "end")),
		PgDown: key.NewBinding(key.WithKeys("ctrl+d", "pgdown"), key.WithHelp("ctrl+d/u", "page")),
		PgUp:   key.NewBinding(key.WithKeys("ctrl+u", "pgup")),

		Fav:      key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "favorite")),
		Filter:   key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "filter")),
		Pin:      key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "pin compare")),
		Discover: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "discover")),
		Install:  key.NewBinding(key.WithKeys("I"), key.WithHelp("I", "install")),
		Save:     key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "save look")),
		Export:   key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "export")),
		Import:   key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "import")),
		Toggle:   key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "toggle")),
		Clear:    key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "clear stack")),
		Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
		PanePrev: key.NewBinding(key.WithKeys("["), key.WithHelp("[ ]", "more panes")),
		PaneNext: key.NewBinding(key.WithKeys("]")),

		SectionThemes: key.NewBinding(key.WithKeys("1"), key.WithHelp("1-4", "sections")),
		SectionFonts:  key.NewBinding(key.WithKeys("2")),
		SectionLooks:  key.NewBinding(key.WithKeys("3")),
		SectionMore:   key.NewBinding(key.WithKeys("4")),
	}
}

// helpKeys adapts a binding set to the bubbles help model.
type helpKeys struct {
	short []key.Binding
	full  [][]key.Binding
}

func (h helpKeys) ShortHelp() []key.Binding  { return h.short }
func (h helpKeys) FullHelp() [][]key.Binding { return h.full }

// shortHelp lists the always-visible footer bindings, followed by bindings that
// only apply to the active focus.
func (k keys) shortHelp(extra ...key.Binding) []key.Binding {
	base := []key.Binding{k.Stage, k.Apply, k.Preview, k.Search, k.Help}
	return append(base, extra...)
}

// fullHelp groups every binding for the help overlay.
func (k keys) fullHelp(extra [][]key.Binding) [][]key.Binding {
	groups := [][]key.Binding{
		{k.Stage, k.Apply, k.Undo, k.Preview},
		{k.Search, k.Help, k.Tune, k.Inspect},
		{k.SectionThemes, k.Next, k.Up, k.Top, k.PgDown},
		{k.Quit},
	}
	return append(groups, extra...)
}
