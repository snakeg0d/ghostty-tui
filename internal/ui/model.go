package ui

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"ghostty.studio/internal/core"
)

// Messages produced by the asynchronous loads and actions.
type (
	stateLoadedMsg   struct{ st core.State }
	catalogLoadedMsg struct {
		section string
		items   []core.Choice
	}
	settingsLoadedMsg struct{ items []core.Setting }
	favsLoadedMsg     struct{ ids []string }
	diagLoadedMsg     struct{ items []core.Diagnostic }
	actionDoneMsg     struct {
		kind  string
		label string
		res   core.Result
		err   error
	}
)

var errEngineUnavailable = errors.New("core engine unavailable")

const (
	minWidth  = 60
	minHeight = 18
)

// Model is the Ghostty Studio terminal atelier.
type Model struct {
	engine *core.Engine
	opts   core.Options

	width, height int
	sec           Section
	more          MorePane

	keys    keys
	styles  *styles
	help    help.Model
	spinner spinner.Model
	busy    bool
	busyFor string

	search   textinput.Model
	searchOn bool

	themes    *choiceList
	fonts     *choiceList
	discover  *choiceList
	looks     *choiceList
	effects   *choiceList
	workflows *choiceList
	settingsL *choiceList
	health    *choiceList
	exports   *choiceList

	fontsDiscover bool

	state      core.State
	stateReady bool
	favs       map[string]bool

	staged    core.Values
	stagedRev string
	stagedFor string
	overrides []string

	pinned string
	stack  []string

	settingItems []core.Setting
	diagItems    []core.Diagnostic

	modal *modal

	status     string
	statusErr  bool
	lastDetail string
}

// New builds the studio model around an opened core engine.
func New(engine *core.Engine, opts core.Options) *Model {
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(lipgloss.NewStyle().Foreground(colAmber)))
	si := textinput.New()
	si.Prompt = ""
	si.Placeholder = "search themes, ids, tags…"
	si.SetStyles(inputStyles())
	si.CharLimit = 0

	hp := help.New()
	hp.Styles = helpStyles()
	hp.ShortSeparator = "   "
	hp.FullSeparator = "    "

	m := &Model{
		engine:    engine,
		opts:      opts,
		width:     100,
		height:    30,
		keys:      newKeys(),
		styles:    newStyles(),
		help:      hp,
		spinner:   sp,
		search:    si,
		favs:      map[string]bool{},
		staged:    core.Values{},
		themes:    newChoiceList("themes"),
		fonts:     newChoiceList("fonts"),
		discover:  newChoiceList("discover"),
		looks:     newChoiceList("looks"),
		effects:   newChoiceList("effects"),
		workflows: newChoiceList("workflows"),
		settingsL: newChoiceList("settings"),
		health:    newChoiceList("health"),
		exports:   newChoiceList("export"),
	}
	m.exports.SetItems(exportFormats())
	m.health.SetItems(nil)
	return m
}

func helpStyles() help.Styles {
	s := help.DefaultDarkStyles()
	s.ShortKey = lipgloss.NewStyle().Foreground(colSage)
	s.ShortDesc = lipgloss.NewStyle().Foreground(colIvoryDim)
	s.ShortSeparator = lipgloss.NewStyle().Foreground(colLine)
	s.Ellipsis = lipgloss.NewStyle().Foreground(colIvoryDim)
	s.FullKey = lipgloss.NewStyle().Foreground(colSage)
	s.FullDesc = lipgloss.NewStyle().Foreground(colIvoryDim)
	s.FullSeparator = lipgloss.NewStyle().Foreground(colLine)
	return s
}

// Init loads every catalog once, asynchronously.
func (m *Model) Init() tea.Cmd {
	if m.engine == nil {
		m.status = "core engine unavailable"
		m.statusErr = true
		return nil
	}
	return tea.Batch(
		m.spinner.Tick,
		loadState(m.engine),
		loadCatalog(m.engine, "themes"),
		loadCatalog(m.engine, "fonts"),
		loadCatalog(m.engine, "discover"),
		loadCatalog(m.engine, "looks"),
		loadCatalog(m.engine, "effects"),
		loadCatalog(m.engine, "workflows"),
		loadSettings(m.engine),
		loadFavorites(m.engine),
		loadDiagnostics(m.engine),
	)
}

func loadState(e *core.Engine) tea.Cmd {
	return func() tea.Msg { return stateLoadedMsg{st: e.State()} }
}

func loadCatalog(e *core.Engine, section string) tea.Cmd {
	return func() tea.Msg {
		return catalogLoadedMsg{section: section, items: e.Catalog(section)}
	}
}

func loadSettings(e *core.Engine) tea.Cmd {
	return func() tea.Msg { return settingsLoadedMsg{items: e.Settings()} }
}

func loadFavorites(e *core.Engine) tea.Cmd {
	return func() tea.Msg { return favsLoadedMsg{ids: e.Favorites()} }
}

func loadDiagnostics(e *core.Engine) tea.Cmd {
	return func() tea.Msg { return diagLoadedMsg{items: e.Diagnose()} }
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil

	case stateLoadedMsg:
		m.state = msg.st
		m.stateReady = true
		if m.stagedRev == "" || len(m.staged) == 0 {
			m.stagedRev = msg.st.Revision
		}
		m.rebuildHealth()
		return m, nil

	case catalogLoadedMsg:
		m.applyCatalog(msg.section, msg.items)
		return m, nil

	case settingsLoadedMsg:
		m.settingItems = msg.items
		m.settingsL.SetItems(settingsAsChoices(msg.items))
		return m, nil

	case favsLoadedMsg:
		next := make(map[string]bool, len(msg.ids))
		for _, id := range msg.ids {
			next[id] = true
		}
		m.favs = next
		m.applyFavorites()
		return m, nil

	case diagLoadedMsg:
		m.diagItems = msg.items
		m.rebuildHealth()
		return m, nil

	case importedLookMsg:
		m.busy = false
		m.busyFor = ""
		if msg.err != nil {
			m.fail("import: " + msg.err.Error())
			return m, nil
		}
		m.stageImportedLook(msg.choice)
		return m, m.reloadCatalogs()

	case actionDoneMsg:
		m.busy = false
		m.busyFor = ""
		m.handleActionDone(msg)
		if msg.err != nil {
			return m, nil
		}
		// Every successful action re-reads the whole surface: installs add
		// font families, saves add looks, applies rewrite the current values.
		return m, m.reloadCatalogs()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.PasteMsg, tea.PasteStartMsg, tea.PasteEndMsg:
		return m, m.routeInput(msg)
	}
	return m, m.routeInput(msg)
}

// routeInput forwards non-key messages to whichever text input owns focus.
func (m *Model) routeInput(msg tea.Msg) tea.Cmd {
	if m.modal.active() {
		if f := m.modal.selectedField(); f != nil {
			var cmd tea.Cmd
			f.input, cmd = f.input.Update(msg)
			return cmd
		}
		return nil
	}
	if m.searchOn {
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		m.applyQuery()
		return cmd
	}
	return nil
}

func (m *Model) applyCatalog(section string, items []core.Choice) {
	switch section {
	case "themes":
		m.themes.SetItems(items)
	case "fonts":
		m.fonts.SetItems(items)
	case "discover":
		m.discover.SetItems(items)
	case "looks":
		m.looks.SetItems(items)
	case "effects":
		m.effects.SetItems(items)
	case "workflows":
		m.workflows.SetItems(items)
	}
	m.applyFavorites()
}

func (m *Model) applyFavorites() {
	m.themes.SetFavorites(m.favs)
	m.fonts.SetFavorites(m.favs)
}

// SetSize stores the terminal size and reflows every list.
func (m *Model) SetSize(w, h int) {
	if w <= 0 {
		w = 100
	}
	if h <= 0 {
		h = 30
	}
	m.width, m.height = w, h
	m.search.SetWidth(max(w-8, 10))
	m.layout()
}

// layout assigns each list its exact extent for the current size.
func (m *Model) layout() {
	m.search.Placeholder = m.searchPlaceholder()
	listW := m.listWidth()
	bodyH := m.bodyHeight()
	height := max(bodyH-m.moreHeaderLines(), 1)
	m.themes.SetExtent(listW, height)
	m.fonts.SetExtent(listW, height)
	m.discover.SetExtent(listW, height)
	m.looks.SetExtent(listW, height)
	m.effects.SetExtent(listW, height)
	m.workflows.SetExtent(listW, height)
	m.settingsL.SetExtent(listW, height)
	m.health.SetExtent(listW, height)
	m.exports.SetExtent(listW, height)
}

func (m *Model) headerHeight() int { return 6 }

func (m *Model) footerHeight() int { return 4 }

func (m *Model) bodyHeight() int {
	return max(m.height-m.headerHeight()-m.footerHeight(), 3)
}

// listWidth keeps the left column near 32 columns while leaving a usable
// specimen on smaller terminals.
func (m *Model) listWidth() int {
	switch {
	case m.width >= 110:
		return 34
	case m.width >= 96:
		return 30
	case m.width >= 76:
		return 27
	default:
		return 24
	}
}

func (m *Model) moreHeaderLines() int {
	if m.sec != SecMore {
		return 1
	}
	return 2
}

// activeList returns the list that owns the cursor and the search.
func (m *Model) activeList() *choiceList {
	switch m.sec {
	case SecThemes:
		return m.themes
	case SecFonts:
		if m.fontsDiscover {
			return m.discover
		}
		return m.fonts
	case SecLooks:
		return m.looks
	default:
		switch m.more {
		case PaneEffects:
			return m.effects
		case PaneSettings:
			return m.settingsL
		case PaneWorkflows:
			return m.workflows
		case PaneHealth:
			return m.health
		default:
			return m.exports
		}
	}
}

// ----- staging -----

// stagePatch merges an explicit patch into the staged appearance. Absent keys
// stay untouched; an empty slice stages a reset of the managed value, exactly
// like core's patch semantics.
func (m *Model) stagePatch(patch core.Values, label string) {
	if m.stagedRev != "" && m.stagedRev != m.state.Revision {
		// Re-staging after a refresh begins a new transaction. Carrying the
		// old patch forward would silently rebase edits the user has not seen.
		m.clearStaged()
	}
	if m.stagedRev == "" {
		m.stagedRev = m.state.Revision
	}
	m.overrides = m.overrides[:0]
	for _, k := range patch.Keys() {
		v := patch[k]
		m.staged[k] = append([]string(nil), v...)
		if cur, ok := m.state.Current[k]; ok && !slices.Equal(cur, v) {
			m.overrides = append(m.overrides, k)
		}
	}
	slices.Sort(m.overrides)
	if label != "" {
		m.stagedFor = label
	}
}

// mergedValues layers staged overrides on top of the live configuration using
// core's own patch operation, so staged resets behave identically here and at
// apply time.
func (m *Model) mergedValues() core.Values {
	return core.Patch(m.state.Current, m.staged)
}

func (m *Model) dirty() bool { return len(m.staged) > 0 }

func (m *Model) clearStaged() {
	m.staged = core.Values{}
	m.stagedFor = ""
	m.overrides = nil
	m.stagedRev = m.state.Revision
}

// pendingSummary renders the staged-change line.
func (m *Model) pendingSummary() string {
	if !m.dirty() {
		return ""
	}
	keys := stagedKeys(m.staged)
	show := keys
	if len(show) > 6 {
		show = show[:6]
	}
	parts := make([]string, len(show))
	for i, k := range show {
		parts[i] = k
	}
	summary := strings.Join(parts, ", ")
	if len(keys) > len(show) {
		summary += " +" + strconv.Itoa(len(keys)-len(show))
	}
	label := m.stagedFor
	if label == "" {
		label = "staged"
	}
	return label + "  ·  " + summary
}

// ----- status -----

func (m *Model) say(s string) {
	m.status = s
	m.statusErr = false
}

func (m *Model) fail(s string) {
	m.status = s
	m.statusErr = true
}

// handleActionDone turns a finished action into short, truthful feedback. The
// verbose core message is kept for Inspect, where there is room for it.
func (m *Model) handleActionDone(msg actionDoneMsg) {
	if msg.err != nil {
		m.lastDetail = msg.label + ": " + msg.err.Error()
		switch msg.kind {
		case "apply":
			m.fail("apply rejected: " + msg.err.Error() + "  — r reloads and re-stages")
		case "refresh":
			m.fail("reload failed: " + msg.err.Error())
		default:
			m.fail(msg.label + ": " + msg.err.Error())
		}
		return
	}

	if detail := detailText(msg.res); detail != "" {
		m.lastDetail = detail
	}

	switch msg.kind {
	case "apply":
		m.clearStaged()
		m.say(reloadHint("Saved", msg.res))
	case "undo":
		m.clearStaged()
		m.say(reloadHint("Restored", msg.res))
	case "refresh":
		m.say("Reloaded  ·  configuration re-read")
	default:
		s := msg.res.Message
		if s == "" {
			s = msg.label
		}
		if len(msg.res.Warnings) > 0 {
			s += "  (" + strings.Join(msg.res.Warnings, "; ") + ")"
		}
		m.say(s)
	}
}

// reloadHint keeps the reload instruction inside 80 columns.
func reloadHint(verb string, res core.Result) string {
	out := verb + "  ·  reload Ghostty ⌘⇧,"
	if verb == "Saved" {
		out += "  ·  u undo"
	}
	if n := len(res.Warnings); n > 0 {
		note := " note"
		if n > 1 {
			note = " notes"
		}
		out += "  ·  " + strconv.Itoa(n) + note
	}
	return out
}

func detailText(res core.Result) string {
	parts := nonEmpty([]string{res.Message})
	if len(res.Warnings) > 0 {
		parts = append(parts, strings.Join(res.Warnings, "; "))
	}
	if res.BackupPath != "" {
		parts = append(parts, "backup "+res.BackupPath)
	}
	return strings.Join(parts, "  ·  ")
}

// startBusy runs an action off the update loop, publishing busy state.
func (m *Model) startBusy(label string, fn func() tea.Msg) tea.Cmd {
	if m.busy {
		return nil
	}
	if m.engine == nil {
		m.fail("core engine unavailable")
		return nil
	}
	m.busy = true
	m.busyFor = label
	return fn
}
