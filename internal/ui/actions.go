package ui

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"ghostty.studio/internal/core"
)

// ----- key dispatch -----

func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if k == "ctrl+c" {
		return m, tea.Quit
	}
	if m.modal.active() {
		return m.handleModalKey(msg)
	}
	if m.searchOn {
		return m.handleSearchKey(msg)
	}
	return m.handleGlobalKey(msg)
}

func (m *Model) handleSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if m.search.Value() != "" {
			m.search.SetValue("")
			m.applyQuery()
			m.say("search cleared")
			return m, nil
		}
		m.searchOn = false
		m.search.Blur()
		return m, nil
	case "enter":
		m.searchOn = false
		m.search.Blur()
		return m, nil
	case "up", "ctrl+p":
		m.activeList().Move(-1)
		return m, nil
	case "down", "ctrl+n":
		m.activeList().Move(1)
		return m, nil
	case "ctrl+d", "pgdown":
		m.activeList().HalfPage(1)
		return m, nil
	case "ctrl+u", "pgup":
		m.activeList().HalfPage(-1)
		return m, nil
	case "tab":
		m.nextSection(1)
		return m, nil
	case "shift+tab":
		m.nextSection(-1)
		return m, nil
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	m.applyQuery()
	return m, cmd
}

func (m *Model) handleGlobalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	list := m.activeList()
	switch msg.String() {
	case "/":
		m.searchOn = true
		m.search.Placeholder = m.searchPlaceholder()
		return m, m.search.Focus()
	case "?":
		m.openHelp()
		return m, nil
	case "esc":
		m.status = ""
		m.statusErr = false
		return m, nil
	case "q":
		if m.dirty() {
			m.openQuitConfirm()
			return m, nil
		}
		return m, tea.Quit
	case "tab":
		m.nextSection(1)
		return m, nil
	case "shift+tab":
		m.nextSection(-1)
		return m, nil
	case "1", "2", "3", "4":
		n := int(msg.String()[0] - '1')
		m.setSection(Section(n))
		return m, nil
	case "j", "down":
		list.Move(1)
		return m, nil
	case "k", "up":
		list.Move(-1)
		return m, nil
	case "g", "home":
		list.MoveTop()
		return m, nil
	case "G", "end":
		list.MoveBottom()
		return m, nil
	case "ctrl+d", "pgdown":
		list.HalfPage(1)
		return m, nil
	case "ctrl+u", "pgup":
		list.HalfPage(-1)
		return m, nil
	case "[":
		if m.sec == SecMore {
			m.switchPane(-1)
		}
		return m, nil
	case "]":
		if m.sec == SecMore {
			m.switchPane(1)
		}
		return m, nil
	case "enter":
		return m, m.stageFocused()
	case "a":
		return m, m.applyStaged()
	case "u":
		return m, m.undo()
	case "p":
		return m, m.preview()
	case "t":
		m.openTune()
		return m, nil
	case "i":
		m.openInspect()
		return m, nil
	case "r":
		return m, m.reloadAll()
	case "f":
		if m.sec == SecThemes || m.sec == SecFonts {
			m.toggleFavorite()
		}
		return m, nil
	case "l":
		m.cycleFilter()
		return m, nil
	case "b":
		m.togglePin()
		return m, nil
	case "d":
		m.toggleDiscover()
		return m, nil
	case "I":
		m.openInstallReview()
		return m, nil
	case "s":
		if m.sec == SecLooks {
			m.openSaveLook()
		}
		return m, nil
	case "e":
		if m.sec == SecLooks {
			m.openExportLook()
		}
		return m, nil
	case "o":
		if m.sec == SecLooks {
			m.openImportLook()
		}
		return m, nil
	case "space":
		if m.sec == SecMore && m.more == PaneEffects {
			m.toggleEffect()
		}
		return m, nil
	case "x":
		if m.sec == SecMore && m.more == PaneEffects {
			m.clearStack()
		}
		return m, nil
	}
	return m, nil
}

// ----- navigation -----

func (m *Model) setSection(s Section) {
	if s == m.sec {
		return
	}
	m.sec = s
	if m.search.Value() != "" {
		m.applyQuery()
	}
	m.layout()
	m.say("section · " + s.String())
}

func (m *Model) nextSection(delta int) {
	if m.sec == SecMore {
		// Left/right within More are handled by [ and ]; tab wraps sections.
	}
	next := (int(m.sec) + delta + len(sectionNames)) % len(sectionNames)
	m.setSection(Section(next))
}

func (m *Model) searchPlaceholder() string {
	switch m.sec {
	case SecThemes:
		return "search themes, ids, tags…"
	case SecFonts:
		if m.fontsDiscover {
			return "search discover shelf, casks, licenses…"
		}
		return "search families and styles…"
	case SecLooks:
		return "search looks…"
	default:
		return "search " + strings.ToLower(m.more.String()) + "…"
	}
}

// applyQuery pushes the current search text into every list so switching
// sections keeps the filter.
func (m *Model) applyQuery() {
	q := m.search.Value()
	m.themes.SetQuery(q)
	m.fonts.SetQuery(q)
	m.discover.SetQuery(q)
	m.looks.SetQuery(q)
	m.effects.SetQuery(q)
	m.workflows.SetQuery(q)
	m.settingsL.SetQuery(q)
	m.health.SetQuery(q)
	m.exports.SetQuery(q)
}

// ----- staging entry -----

// stageFocused stages the focused item without writing anything.
func (m *Model) stageFocused() tea.Cmd {
	if m.busy {
		m.fail("working…")
		return nil
	}
	switch m.sec {
	case SecThemes:
		m.stageFocusedTheme()
	case SecFonts:
		m.stageFocusedFont()
	case SecLooks:
		m.stageFocusedLook()
	default:
		m.stageFocusedMore()
	}
	return nil
}

// ----- actions -----

func (m *Model) applyStaged() tea.Cmd {
	switch {
	case !m.dirty():
		m.fail("nothing staged — press enter on a choice first")
		return nil
	case !m.stateReady:
		m.fail("configuration is still loading")
		return nil
	case m.engine == nil:
		m.fail("core engine unavailable")
		return nil
	}
	rev := m.stagedRev
	if rev == "" {
		rev = m.state.Revision
	}
	ch := core.Change{Values: cloneValues(m.staged), Revision: rev, Label: m.stagedFor}
	eng := m.engine
	return m.startBusy("apply", func() tea.Msg {
		res, err := eng.Apply(ch)
		return actionDoneMsg{kind: "apply", label: "apply", res: res, err: err}
	})
}

func (m *Model) undo() tea.Cmd {
	if m.engine == nil {
		m.fail("core engine unavailable")
		return nil
	}
	eng := m.engine
	return m.startBusy("undo", func() tea.Msg {
		res, err := eng.Undo()
		return actionDoneMsg{kind: "undo", label: "undo", res: res, err: err}
	})
}

func (m *Model) preview() tea.Cmd {
	if m.engine == nil {
		m.fail("core engine unavailable")
		return nil
	}
	values := cloneValues(m.staged)
	label := m.stagedFor
	if len(values) == 0 {
		if c, _, ok := m.activeList().Selected(); ok && len(c.Values) > 0 {
			values = cloneValues(c.Values)
			label = c.Name
		}
	}
	if len(values) == 0 {
		m.fail("nothing to preview")
		return nil
	}
	rev := m.stagedRev
	if rev == "" {
		rev = m.state.Revision
	}
	ch := core.Change{Values: values, Revision: rev, Label: label}
	eng := m.engine
	return m.startBusy("native preview", func() tea.Msg {
		res, err := eng.Preview(ch)
		return actionDoneMsg{kind: "preview", label: "preview", res: res, err: err}
	})
}

// reloadAll re-reads the configuration graph, then every catalog. Core's
// cached state is only replaced when the engine refreshes, so an external edit
// could never be picked up by reads alone.
func (m *Model) reloadAll() tea.Cmd {
	if m.engine == nil {
		m.fail("core engine unavailable")
		return nil
	}
	eng := m.engine
	return m.startBusy("reload", func() tea.Msg {
		return actionDoneMsg{kind: "refresh", label: "reload", err: eng.Refresh()}
	})
}

// reloadCatalogs re-reads state, catalogs, settings, favorites and diagnostics.
func (m *Model) reloadCatalogs() tea.Cmd {
	if m.engine == nil {
		return nil
	}
	cmds := []tea.Cmd{
		loadState(m.engine),
		loadSettings(m.engine),
		loadFavorites(m.engine),
		loadDiagnostics(m.engine),
	}
	for _, section := range catalogSections {
		cmds = append(cmds, loadCatalog(m.engine, section))
	}
	return tea.Batch(cmds...)
}

func cloneValues(v core.Values) core.Values { return v.Clone() }

// ----- modals -----

func (m *Model) closeModal() { m.modal = nil }

func (m *Model) infoModal(title, subtitle string, body []string) {
	m.modal = &modal{
		kind:     modalInfo,
		title:    title,
		subtitle: subtitle,
		body:     body,
		buttons:  []string{"close"},
		action:   "close",
	}
	m.modal.focusButtons()
}

func (m *Model) openHelp() {
	st := m.styles
	body := []string{}
	for _, group := range m.helpGroups() {
		for _, b := range group {
			h := b.Help()
			if h.Key == "" || h.Desc == "" {
				continue
			}
			body = append(body, st.Sage.Render(padRight(h.Key, 16))+st.Text.Render(h.Desc))
		}
		body = append(body, "")
	}
	body = append(body,
		st.Dim.Render("staging never writes. apply is the only write path."),
		st.Dim.Render("undo restores the byte-identical original and rejects external edits."),
	)
	m.infoModal("Keyboard", "every binding listed here is live", body)
}

func (m *Model) helpGroups() [][]key.Binding {
	extra := [][]key.Binding{}
	switch m.sec {
	case SecThemes:
		extra = append(extra, []key.Binding{m.keys.Fav, m.keys.Filter, m.keys.Pin})
	case SecFonts:
		extra = append(extra, []key.Binding{m.keys.Discover, m.keys.Install})
	case SecLooks:
		extra = append(extra, []key.Binding{m.keys.Save, m.keys.Export, m.keys.Import})
	default:
		extra = append(extra, []key.Binding{m.keys.PanePrev, m.keys.Toggle, m.keys.Clear, m.keys.Refresh})
	}
	return m.keys.fullHelp(extra)
}

func (m *Model) openInspect() {
	st := m.styles
	c, _, ok := m.activeList().Selected()
	body := []string{}
	if ok {
		body = append(body, st.Amber.Render(c.Name))
		body = append(body, st.Dim.Render("id      "+c.ID))
		if c.Source != "" {
			body = append(body, st.Dim.Render("source  "+c.Source))
		}
		if c.License != "" {
			body = append(body, st.Dim.Render("license "+c.License))
		}
		if c.URL != "" {
			body = append(body, st.Dim.Render("url     "+c.URL))
		}
		if len(c.Tags) > 0 {
			body = append(body, st.Dim.Render("tags    "+strings.Join(c.Tags, ", ")))
		}
		body = append(body, "")
		if len(c.Values) > 0 {
			body = append(body, st.Sage.Render("values this choice stages"))
			for _, k := range stagedKeys(c.Values) {
				body = append(body, st.Text.Render("  "+k+" = "+clipText(strings.Join(c.Values[k], " "), 60)))
			}
			body = append(body, "")
		}
	}

	if m.dirty() {
		body = append(body, st.Sage.Render("diff of staged changes"))
		ch := core.Change{Values: cloneValues(m.staged), Revision: m.stagedRev, Label: m.stagedFor}
		diff := ""
		if m.engine != nil {
			diff = m.engine.Diff(ch)
		}
		if strings.TrimSpace(diff) == "" {
			body = append(body, st.Dim.Render("  no textual diff"))
		} else {
			for _, l := range splitLines(diff) {
				style := st.Text
				switch {
				case strings.HasPrefix(l, "+"):
					style = st.OK
				case strings.HasPrefix(l, "-"):
					style = st.Err
				case strings.HasPrefix(l, "@"):
					style = st.Sage
				}
				body = append(body, style.Render(l))
			}
		}
		body = append(body, "")
	} else {
		body = append(body, st.Dim.Render("nothing staged yet"), "")
	}

	body = append(body, st.Sage.Render("config sources"))
	if len(m.state.Sources) == 0 {
		body = append(body, st.Dim.Render("  (none reported)"))
	}
	for _, s := range m.state.Sources {
		body = append(body, st.Text.Render("  "+s))
	}
	for _, wmsg := range m.state.Warnings {
		body = append(body, st.Err.Render("  warning: "+wmsg))
	}
	if m.lastDetail != "" {
		body = append(body, "",
			st.Sage.Render("last transaction"),
			st.Dim.Render(clipText(m.lastDetail, 100)))
	}
	m.infoModal("Inspect", "staged diff, provenance and configuration sources", body)
}

func (m *Model) openTune() {
	st := m.styles
	keys := m.tuneKeys()
	items := make([]string, 0, len(keys))
	ids := make([]string, 0, len(keys))
	for _, k := range keys {
		s, _ := m.settingByKey(k)
		items = append(items, st.Text.Render(padRight(k, 24))+st.Sage.Render(clipText(m.currentSettingValue(k), 24))+"  "+st.Dim.Render(s.Kind))
		ids = append(ids, k)
	}
	md := &modal{
		kind:     modalList,
		title:    "Tune · " + m.sec.String(),
		subtitle: "enter opens the value editor · staged only",
		items:    items,
		itemIDs:  ids,
		buttons:  []string{"close"},
		action:   "tune",
	}
	m.modal = md
	md.focusList()
}

// tuneKeys lists the quick controls for the active section.
func (m *Model) tuneKeys() []string {
	switch m.sec {
	case SecThemes:
		return []string{"background-opacity", "background-blur", "window-padding-x", "window-padding-y", "background-image"}
	case SecFonts:
		return []string{"font-size", "font-feature", "font-family", "adjust-cell-height", "font-thicken"}
	case SecLooks:
		return []string{"background-opacity", "window-padding-x", "window-padding-y", "font-size", "cursor-style"}
	default:
		if m.more == PaneEffects {
			return []string{"custom-shader-animation", "custom-shader"}
		}
		return []string{"window-padding-x", "window-padding-y", "background-opacity", "font-size", "scrollback-limit"}
	}
}

// openSettingEditor edits the focused setting, or an explicit key.
func (m *Model) openSettingEditor() {
	c, _, ok := m.settingsL.Selected()
	if !ok {
		m.fail("no setting selected")
		return
	}
	m.openSettingKeyEditor(strings.TrimPrefix(c.ID, "setting:"))
}

// settingEditPayload tells submitModal how to interpret the field.
type settingEditPayload struct {
	key      string
	repeated bool
}

func isRepeatedKey(key string, s core.Setting) bool {
	return core.IsRepeatable(key) || strings.EqualFold(s.Kind, "list")
}

// openSettingKeyEditor edits one configuration key. Single-valued keys keep the
// plain editor; repeated keys get an explicit JSON array so no fallback entry
// is silently folded into another.
func (m *Model) openSettingKeyEditor(key string) {
	s, _ := m.settingByKey(key)
	repeated := isRepeatedKey(key, s)

	var raw []string
	switch {
	case m.staged[key] != nil:
		raw = m.staged[key]
	case m.state.Current[key] != nil:
		raw = m.state.Current[key]
	case s.Default != "":
		raw = []string{s.Default}
	}

	var (
		value    string
		hint     string
		buttons  = []string{"stage", "cancel"}
		validate func(string) error
	)
	if repeated {
		// Reset markers are serialization, not font names or editable entries.
		for i := len(raw) - 1; i >= 0; i-- {
			if raw[i] == "" {
				raw = raw[i+1:]
				break
			}
		}
		value = jsonList(raw)
		hint = `JSON array, e.g. ["JetBrains Mono","Menlo"]`
		buttons = []string{"stage", "reset", "cancel"}
		validate = func(v string) error {
			_, err := parseJSONList(v)
			return err
		}
	} else {
		value = strings.Join(raw, " ")
		hint = "empty resets the managed value"
		validate = func(v string) error { return validateSettingValue(s, v) }
	}

	f := newField("value for "+key, value, hint, 46, validate)
	md := &modal{
		kind:     modalForm,
		title:    key,
		subtitle: describeSetting(s) + "  ·  " + hint,
		fields:   []*modalField{f},
		buttons:  buttons,
		action:   "setting-set",
		payload:  settingEditPayload{key: key, repeated: repeated},
	}
	m.modal = md
	md.focusField(0)
}

// jsonList renders repeated values as the JSON array the editor expects.
func jsonList(vals []string) string {
	if len(vals) == 0 {
		return "[]"
	}
	data, err := json.Marshal(vals)
	if err != nil {
		return "[]"
	}
	return string(data)
}

// parseJSONList validates an edited array.
func parseJSONList(raw string) ([]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "[]" {
		return nil, errors.New(`enter a JSON array such as ["JetBrains Mono","Menlo"], or press reset`)
	}
	var out []string
	if err := json.Unmarshal([]byte(trimmed), &out); err != nil {
		return nil, errors.New("expected a JSON array of strings")
	}
	if len(out) == 0 {
		return nil, errors.New("the array is empty; press reset instead")
	}
	for _, v := range out {
		if strings.TrimSpace(v) == "" {
			return nil, errors.New("entries cannot be empty")
		}
		if strings.ContainsAny(v, "\r\n") {
			return nil, errors.New("newlines are not allowed")
		}
	}
	return out, nil
}

func describeSetting(s core.Setting) string {
	parts := nonEmpty([]string{s.Group, s.Kind, reloadLabel(s.Reload)})
	if s.Name != "" && s.Name != s.Key {
		parts = append([]string{s.Name}, parts...)
	}
	return strings.Join(parts, "  ·  ")
}

// validateSettingValue gives immediate local feedback. Ghostty remains the
// authority: a value that passes here is still validated on apply.
func validateSettingValue(s core.Setting, v string) error {
	if strings.ContainsAny(v, "\r\n") {
		return errors.New("newlines are not allowed")
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	kind := strings.ToLower(s.Kind)
	switch {
	case strings.Contains(kind, "bool"):
		if v != "true" && v != "false" {
			return errors.New("expected true or false")
		}
	case strings.Contains(kind, "int"):
		if _, err := strconv.Atoi(v); err != nil {
			return errors.New("expected a whole number")
		}
	case strings.Contains(kind, "float"), strings.Contains(kind, "opacity"), strings.Contains(kind, "percent"):
		if _, err := strconv.ParseFloat(v, 64); err != nil {
			return errors.New("expected a number")
		}
	case strings.Contains(kind, "enum"), strings.Contains(kind, "option"), len(s.Options) > 0:
		for _, o := range s.Options {
			if o == v {
				return nil
			}
		}
		if len(s.Options) > 0 {
			return errors.New("expected one of: " + strings.Join(s.Options, ", "))
		}
	}
	return nil
}

func (m *Model) openSaveLook() {
	name := &modalField{}
	name = newField("look name", "", "e.g. Evening Desk", 40, func(v string) error {
		if v == "" {
			return errors.New("a name is required")
		}
		if strings.ContainsAny(v, "\r\n/") {
			return errors.New("no slashes or newlines")
		}
		return nil
	})
	md := &modal{
		kind:     modalForm,
		title:    "Save look",
		subtitle: "saves the current + staged appearance through core's allowlist",
		fields:   []*modalField{name},
		buttons:  []string{"save", "cancel"},
		action:   "look-save",
	}
	m.modal = md
	md.focusField(0)
}

func (m *Model) openExportLook() {
	fName := newField("look name", "studio-look", "preset name", 40, nil)
	fPath := newField("export path", defaultLookPath(), "~/Documents/studio-look.json", 44, nil)
	md := &modal{
		kind:     modalForm,
		title:    "Export look",
		subtitle: "allowlisted JSON only · unrelated config is never included",
		fields:   []*modalField{fName, fPath},
		buttons:  []string{"export", "cancel"},
		action:   "look-export",
	}
	m.modal = md
	md.focusField(0)
}

func (m *Model) openImportLook() {
	path := newField("json path", "", "~/Documents/studio-look.json", 44, func(v string) error {
		if v == "" {
			return errors.New("a path is required")
		}
		return nil
	})
	md := &modal{
		kind:     modalForm,
		title:    "Import look",
		subtitle: "imported shader paths and commands are validated before they can apply",
		fields:   []*modalField{path},
		buttons:  []string{"import", "cancel"},
		action:   "look-import",
	}
	m.modal = md
	md.focusField(0)
}

func defaultLookPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "studio-look.json"
	}
	return filepath.Join(home, "Documents", "studio-look.json")
}

func (m *Model) openInstallReview() {
	st := m.styles
	if m.sec != SecFonts || !m.fontsDiscover {
		m.fail("press d for the discover shelf, then I to review")
		return
	}
	c, _, ok := m.discover.Selected()
	if !ok {
		m.fail("no discover item selected")
		return
	}
	body := []string{
		st.Amber.Render(c.Name),
		"",
		st.Sage.Render("source    ") + st.Text.Render(orDash(c.Source)),
		st.Sage.Render("license   ") + st.Text.Render(orDash(c.License)),
		st.Sage.Render("homepage  ") + st.Text.Render(orDash(c.URL)),
		st.Sage.Render("installs  ") + st.Text.Render("Homebrew cask, into your user font directory"),
		"",
		st.Dim.Render("Homebrew keeps ownership and upstream notices."),
		st.Dim.Render("Nothing changes in your config until you apply a font."),
	}
	md := &modal{
		kind:     modalInfo,
		title:    "Install font",
		subtitle: c.ID,
		body:     body,
		buttons:  []string{"install", "cancel"},
		action:   "font-install",
		payload:  c.ID,
	}
	m.modal = md
	md.focusButtons()
	md.btnIdx = 1 // default to cancel: installs are explicit
}

func (m *Model) openExportPrompt() {
	c, _, ok := m.exports.Selected()
	if !ok {
		m.fail("no export format selected")
		return
	}
	format, _ := c.Category, true
	path := newField("write to", defaultExportPath(format), "~/Documents/palette", 46, func(v string) error {
		if v == "" {
			return errors.New("a path is required")
		}
		return nil
	})
	md := &modal{
		kind:     modalForm,
		title:    "Export palette",
		subtitle: c.Name,
		fields:   []*modalField{path},
		buttons:  []string{"write", "cancel"},
		action:   "export-write",
		payload:  format,
	}
	m.modal = md
	md.focusField(0)
}

func defaultExportPath(format string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "palette." + format
	}
	return filepath.Join(home, "Documents", "ghostty-palette."+format)
}

func (m *Model) openQuitConfirm() {
	md := &modal{
		kind:     modalInfo,
		title:    "Discard staged changes?",
		subtitle: m.pendingSummary(),
		body: []string{
			m.styles.Dim.Render("Nothing has been written yet."),
			m.styles.Dim.Render("Discarding drops the staged patch and quits."),
		},
		buttons: []string{"discard and quit", "cancel"},
		action:  "quit-confirm",
	}
	m.modal = md
	md.focusButtons()
	md.btnIdx = 1
}

// ----- modal interaction -----

func (m *Model) handleModalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	md := m.modal
	if m.busy {
		return m, nil
	}
	k := msg.String()
	switch k {
	case "esc":
		m.closeModal()
		return m, nil
	case "tab", "shift+tab":
		m.cycleModalFocus(k == "shift+tab")
		return m, nil
	case "enter":
		return m, m.modalEnter()
	case "pgup", "pgdown":
		// Page keys scroll the body in every modal, including while a field
		// is focused: no text input binds them.
		md.scrollBy(pageSign(k) * modalPage)
		return m, nil
	}

	// A focused field owns every remaining key, arrows included, so editing
	// left/right never jerks the button row.
	if f := md.selectedField(); f != nil {
		var cmd tea.Cmd
		f.input, cmd = f.input.Update(msg)
		return m, cmd
	}

	if md.kind == modalList && md.focusKind == 0 {
		switch k {
		case "j", "down", "ctrl+n":
			md.itemIdx = clampInt(md.itemIdx+1, 0, max(len(md.items)-1, 0))
			return m, nil
		case "k", "up", "ctrl+p":
			md.itemIdx = clampInt(md.itemIdx-1, 0, max(len(md.items)-1, 0))
			return m, nil
		case "g", "home":
			md.itemIdx = 0
			return m, nil
		case "G", "end":
			md.itemIdx = max(len(md.items)-1, 0)
			return m, nil
		}
	}

	step := 0
	switch k {
	case "ctrl+d":
		step = modalPage
	case "ctrl+u":
		step = -modalPage
	case "j", "down":
		step = 1
	case "k", "up":
		step = -1
	}
	if step != 0 && md.kind == modalInfo {
		md.scrollBy(step)
		return m, nil
	}

	if md.focusKind == 1 {
		switch k {
		case "h", "left":
			md.btnIdx = (md.btnIdx - 1 + max(len(md.buttons), 1)) % max(len(md.buttons), 1)
		case "l", "right":
			md.btnIdx = (md.btnIdx + 1) % max(len(md.buttons), 1)
		case "j", "down":
			md.btnIdx = (md.btnIdx + 1) % max(len(md.buttons), 1)
		case "k", "up":
			md.btnIdx = (md.btnIdx - 1 + max(len(md.buttons), 1)) % max(len(md.buttons), 1)
		}
		return m, nil
	}
	return m, nil
}

// modalPage is how far one page key scrolls a long modal body.
const modalPage = 8

func pageSign(k string) int {
	if k == "pgup" {
		return -1
	}
	return 1
}

// scrollBy moves the modal body without scrolling past its bounds.
func (md *modal) scrollBy(delta int) {
	md.scroll += delta
	if md.scroll < 0 {
		md.scroll = 0
	}
	if md.kind == modalInfo && len(md.body) > 0 && md.scroll > len(md.body)-1 {
		md.scroll = len(md.body) - 1
	}
}

func (m *Model) cycleModalFocus(back bool) {
	md := m.modal
	if len(md.fields) == 0 && md.kind != modalList {
		md.focusButtons()
		return
	}
	switch md.focusKind {
	case 0:
		if back {
			md.focusButtons()
			return
		}
		if md.kind == modalList {
			md.focusButtons()
			return
		}
		if md.fieldIdx+1 < len(md.fields) {
			md.focusField(md.fieldIdx + 1)
			return
		}
		md.focusButtons()
	case 1:
		if back {
			if md.kind == modalList {
				md.focusList()
				return
			}
			if len(md.fields) > 0 {
				md.focusField(len(md.fields) - 1)
				return
			}
		}
	}
}

func (md *modal) focusList() {
	md.focusKind = 0
	for _, f := range md.fields {
		f.input.Blur()
	}
}

// modalEnter activates the focused field, list item or button.
func (m *Model) modalEnter() tea.Cmd {
	md := m.modal
	if md.focusKind == 0 {
		switch md.kind {
		case modalList:
			if md.itemIdx >= 0 && md.itemIdx < len(md.itemIDs) {
				key := md.itemIDs[md.itemIdx]
				m.openSettingKeyEditor(key)
			}
			return nil
		default:
			if md.fieldIdx+1 < len(md.fields) {
				md.focusField(md.fieldIdx + 1)
				return nil
			}
			md.focusButtons()
			return nil
		}
	}
	// Buttons: the last button in every modal is cancel.
	if md.btnIdx == len(md.buttons)-1 {
		m.closeModal()
		return nil
	}
	return m.submitModal()
}

// submitModal performs the modal's action. Every mutation goes through core.
func (m *Model) submitModal() tea.Cmd {
	md := m.modal
	vals := md.values()
	edit, editingSetting := md.payload.(settingEditPayload)
	resetList := editingSetting && edit.repeated && md.btnIdx == 1
	if !resetList {
		if label, err := md.fieldError(); err != nil {
			m.fail(label + ": " + err.Error())
			return nil
		}
	}
	switch md.action {
	case "close", "tune":
		m.closeModal()
		return nil
	case "quit-confirm":
		m.closeModal()
		m.clearStaged()
		return tea.Quit
	case "setting-set":
		edit, _ := md.payload.(settingEditPayload)
		key := edit.key
		resetting := edit.repeated && md.btnIdx == 1
		if !edit.repeated && vals[0] == "" {
			resetting = true
		}
		m.closeModal()
		patch := core.Values{key: nil}
		switch {
		case resetting:
			// An empty value list hands the key back to the user's own
			// configuration; that is the only reset path.
		case edit.repeated:
			list, err := parseJSONList(vals[0])
			if err != nil {
				m.fail(key + ": " + err.Error())
				return nil
			}
			// The editor represents the entire desired list. Replace inherited
			// entries instead of appending behind the old primary font.
			patch[key] = append([]string{""}, list...)
		default:
			patch[key] = []string{vals[0]}
		}
		m.stagePatch(patch, "setting · "+key)
		switch {
		case resetting:
			m.say("staged reset of " + key)
		case edit.repeated:
			m.say("staged " + key + " = " + strings.Join(patch[key], ", "))
		default:
			m.say("staged " + key + " = " + vals[0])
		}
		return nil
	case "look-save":
		m.closeModal()
		if m.engine == nil {
			m.fail("core engine unavailable")
			return nil
		}
		name := vals[0]
		values := m.mergedValues()
		eng := m.engine
		return m.startBusy("save look", func() tea.Msg {
			err := eng.SaveLook(name, values)
			return actionDoneMsg{kind: "look-save", label: "save look " + name, err: err}
		})
	case "look-export":
		m.closeModal()
		if m.engine == nil {
			m.fail("core engine unavailable")
			return nil
		}
		name, path := vals[0], expandPath(vals[1])
		values := m.mergedValues()
		eng := m.engine
		return m.startBusy("export look", func() tea.Msg {
			err := eng.ExportLook(name, values, path)
			return actionDoneMsg{kind: "look-export", label: "export look → " + path, err: err}
		})
	case "look-import":
		m.closeModal()
		if m.engine == nil {
			m.fail("core engine unavailable")
			return nil
		}
		path := expandPath(vals[0])
		eng := m.engine
		return m.startBusy("import look", func() tea.Msg {
			choice, err := eng.ImportLook(path)
			if err == nil {
				err = eng.SaveLook(choice.Name, choice.Values)
			}
			return importedLookMsg{choice: choice, err: err}
		})
	case "font-install":
		id, _ := md.payload.(string)
		m.closeModal()
		if m.engine == nil {
			m.fail("core engine unavailable")
			return nil
		}
		eng := m.engine
		return m.startBusy("install font", func() tea.Msg {
			res, err := eng.InstallFont(id)
			return actionDoneMsg{kind: "install", label: "install font", res: res, err: err}
		})
	case "export-write":
		format, _ := md.payload.(string)
		path := expandPath(vals[0])
		text := m.exportPreview(format)
		m.closeModal()
		if strings.TrimSpace(text) == "" {
			m.fail("export produced no output")
			return nil
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			m.fail("export: " + err.Error())
			return nil
		}
		m.say("wrote " + path)
		return nil
	}
	m.closeModal()
	return nil
}

// importedLookMsg carries an imported preset back to the update loop.
type importedLookMsg struct {
	choice core.Choice
	err    error
}

func expandPath(p string) string {
	p = strings.TrimSpace(p)
	if strings.HasPrefix(p, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}
