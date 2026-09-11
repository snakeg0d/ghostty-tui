package ui

import (
	"slices"
	"strconv"
	"strings"

	"ghostty.studio/internal/core"
)

// ----- adapters -----

// settingsAsChoices adapts the settings reference to the shared list. Every key
// from the native reference is searchable by key, human name and options.
func settingsAsChoices(items []core.Setting) []core.Choice {
	out := make([]core.Choice, 0, len(items))
	for _, s := range items {
		tags := append([]string{s.Name, s.Kind, s.Group}, s.Options...)
		out = append(out, core.Choice{
			ID:          "setting:" + s.Key,
			Name:        s.Key,
			Description: s.Description,
			Category:    s.Group,
			Source:      s.Kind,
			Tags:        tags,
			Values:      core.Values{},
			License:     s.Reload,
		})
	}
	return out
}

// exportFormats are the three palette export dialects core supports.
func exportFormats() []core.Choice {
	return []core.Choice{
		{ID: "export:ghostty", Name: "Ghostty config", Category: "ghostty", Description: "Native config snippet with the full palette."},
		{ID: "export:fzf", Name: "fzf shell snippet", Category: "fzf", Description: "Shell variable assignments from validated hex colors. Never executed, never written to rc files."},
		{ID: "export:json", Name: "Terminal palette JSON", Category: "json", Description: "Portable JSON palette for other terminal tools."},
	}
}

// ----- panes -----

func (m *Model) morePane(w, h int) []string {
	switch m.more {
	case PaneEffects:
		return m.effectsPane(w, h)
	case PaneSettings:
		return m.settingsPane(w, h)
	case PaneWorkflows:
		return m.workflowsPane(w, h)
	case PaneHealth:
		return m.healthPane(w, h)
	default:
		return m.exportPane(w, h)
	}
}

func (m *Model) switchPane(delta int) {
	next := (int(m.more) + delta + len(morePaneNames)) % len(morePaneNames)
	m.more = MorePane(next)
	m.layout()
	m.say("more · " + m.more.String())
}

// ----- effects -----

func (m *Model) effectsPane(w, h int) []string {
	st := m.styles
	title := "effects · none"
	if len(m.stack) > 0 {
		names := make([]string, 0, len(m.stack))
		for _, id := range m.stack {
			names = append(names, m.effectName(id))
		}
		title = "effects · " + strings.Join(names, " → ")
	}
	rows := splitLines(renderSpecimen(specimenData{Title: title, Pal: paletteFromValues(m.mergedValues())}, w, max(h-4, 5), m.styles))
	rows = append(rows, "")
	if c, _, ok := m.effects.Selected(); ok {
		rows = append(rows, st.Sage.Render(clipText(c.Name+" · "+c.Source, max(w, 8))))
		rows = append(rows, st.Dim.Render(clipText(c.Description, max(w, 8))))
	} else {
		rows = append(rows, st.Dim.Render("no shaders discovered"))
	}
	rows = append(rows, st.Dim.Render(clipText("Effects render in native preview · p opens Ghostty", max(w, 8))))
	for len(rows) < h {
		rows = append(rows, "")
	}
	return rows[:h]
}

func (m *Model) effectByID(id string) core.Choice {
	for _, c := range m.effects.Items() {
		if c.ID == id {
			return c
		}
	}
	return core.Choice{}
}

func (m *Model) effectName(id string) string {
	if c := m.effectByID(id); c.ID != "" {
		return c.Name
	}
	return id
}

// effectPath returns the shader path a catalog entry points at. Core models an
// effect as {"", "/abs/path.glsl"}: the leading empty string resets the
// inherited stack, and the path is the entry itself.
func effectPath(c core.Choice) string {
	path := ""
	for _, v := range c.Values["custom-shader"] {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			path = trimmed
		}
	}
	return path
}

func (m *Model) toggleEffect() {
	c, _, ok := m.effects.Selected()
	if !ok {
		m.fail("no effect selected")
		return
	}
	if c.Category == "none" || c.ID == "effect:none" {
		m.stack = nil
		m.stageStack()
		m.say("effects reset")
		return
	}
	if effectPath(c) == "" {
		m.fail("effect " + c.Name + " exposes no shader path")
		return
	}
	if idx := indexOf(m.stack, c.ID); idx >= 0 {
		m.stack = append(m.stack[:idx], m.stack[idx+1:]...)
		m.say("removed " + c.Name)
	} else {
		m.stack = append(m.stack, c.ID)
		m.say("added " + c.Name)
	}
	m.stageStack()
	m.effects.Select(c.ID)
}

// clearStack stages an explicit reset even when the inherited stack comes from
// the user's own configuration and is not represented in the catalog.
func (m *Model) clearStack() {
	if len(m.stack) == 0 && !stagedStackIsReset(m.staged) {
		m.say("staging shader reset")
	} else {
		m.say("effect stack cleared")
	}
	m.stack = nil
	m.stageStack()
}

func stagedStackIsReset(staged core.Values) bool {
	vals, ok := staged["custom-shader"]
	return ok && len(vals) == 1 && strings.TrimSpace(vals[0]) == ""
}

// stageStack re-derives the whole managed shader value from the ordered stack.
// It always begins with exactly one empty entry: that resets every inherited
// shader, including ones this catalog does not know about, and an empty slice
// would instead remove the managed override and revive them.
func (m *Model) stageStack() {
	paths := []string{""}
	names := make([]string, 0, len(m.stack))
	for _, id := range m.stack {
		c := m.effectByID(id)
		p := effectPath(c)
		if p == "" {
			continue
		}
		paths = append(paths, p)
		names = append(names, c.Name)
	}
	patch := core.Values{
		"custom-shader":           paths,
		"custom-shader-animation": []string{"false"},
	}
	label := "effects · reset"
	if len(names) > 0 {
		label = "effects · " + strings.Join(names, " → ")
	}
	m.stagePatch(patch, label)
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

// ----- settings -----

func (m *Model) settingsPane(w, h int) []string {
	st := m.styles
	c, idx, ok := m.settingsL.Selected()
	if !ok {
		return append(blankLines(1), st.Dim.Render("no settings reference loaded"))
	}
	s, have := m.settingByKey(settingKeyOf(c))

	rows := []string{
		st.Text.Render(clipText(c.Name, max(w, 8))),
		st.Sage.Render(clipText(strings.Join(nonEmpty([]string{s.Name, s.Group, s.Kind, reloadLabel(s.Reload)}), "  ·  "), max(w, 8))),
		"",
	}
	rows = append(rows, wrapLines(c.Description, max(w, 8), 4)...)
	if !have {
		rows = append(rows, "", st.Dim.Render("reference only · cataloged for search, not typed editing"))
	}
	rows = append(rows,
		"",
		kv(st, "current", m.currentSettingValue(s.Key), w),
		kv(st, "default", s.Default, w),
	)
	if len(s.Options) > 0 {
		rows = append(rows, kv(st, "options", strings.Join(s.Options, ", "), w))
	}
	if idx >= 0 {
		rows = append(rows, kv(st, "index", strconv.Itoa(idx+1)+" of "+strconv.Itoa(m.settingsL.Len()), w))
	}
	rows = append(rows, "", st.Dim.Render(clipText("enter edits the value · staged only until you apply · Ghostty validates on apply", max(w, 8))))

	for len(rows) < h {
		rows = append(rows, "")
	}
	return rows[:h]
}

// settingKeyOf extracts the configuration key from a settings choice ID.
func settingKeyOf(c core.Choice) string { return strings.TrimPrefix(c.ID, "setting:") }

// currentSettingValue reports the staged value first: a staged patch is what
// the user is about to apply, so it must be the visible value.
func (m *Model) currentSettingValue(key string) string {
	if vals, ok := m.staged[key]; ok {
		if len(vals) == 0 {
			return "(reset staged)"
		}
		return strings.Join(vals, " ") + "  (staged)"
	}
	if vals, ok := m.state.Current[key]; ok && len(vals) > 0 {
		return strings.Join(vals, " ")
	}
	return "(unset)"
}

func (m *Model) settingByKey(key string) (core.Setting, bool) {
	for _, s := range m.settingItems {
		if s.Key == key {
			return s, true
		}
	}
	return core.Setting{Key: key}, false
}

func reloadLabel(r string) string {
	if strings.TrimSpace(r) == "" {
		return "reload: unknown"
	}
	return "reload: " + r
}

func kv(st *styles, label, value string, w int) string {
	if strings.TrimSpace(value) == "" {
		value = "—"
	}
	return st.Dim.Render(padRight(label, 10)) + st.Text.Render(clipText(value, max(w-11, 6)))
}

func nonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

// wrapLines soft-wraps text to the pane width, capped at maxLines.
func wrapLines(text string, w, maxLines int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	cur := ""
	for _, word := range words {
		switch {
		case cur == "":
			cur = word
		case len(cur)+1+len(word) <= w:
			cur += " " + word
		default:
			lines = append(lines, cur)
			cur = word
			if len(lines) == maxLines {
				return lines
			}
		}
	}
	if cur != "" && len(lines) < maxLines {
		lines = append(lines, cur)
	}
	return lines
}

// ----- workflows -----

func (m *Model) workflowsPane(w, h int) []string {
	st := m.styles
	c, _, ok := m.workflows.Selected()
	if !ok {
		return append(blankLines(1), st.Dim.Render("no workflows catalog loaded"))
	}
	rows := []string{
		st.Text.Render(clipText(c.Name, max(w, 8))),
		st.Sage.Render(clipText(strings.TrimSpace(c.Source+"  ·  "+c.Category), max(w, 8))),
		"",
	}
	rows = append(rows, wrapLines(c.Description, max(w, 8), 4)...)
	rows = append(rows, "")
	if len(c.Values) == 0 {
		rows = append(rows, st.Dim.Render("this workflow carries no patch"))
	}
	for _, k := range stagedKeys(c.Values) {
		cur, curOK := m.state.Current[k]
		mark := st.Dim.Render("·")
		note := "unset today"
		switch {
		case !curOK:
			note = "unset today"
		case slices.Equal(cur, c.Values[k]):
			note = "already matches"
		default:
			mark = st.Amber.Render("!")
			note = "overrides " + clipText(strings.Join(cur, " "), 18)
		}
		rows = append(rows, mark+" "+st.Text.Render(k)+st.Dim.Render("  "+note))
		if next := strings.Join(c.Values[k], " "); next != "" {
			rows = append(rows, "    "+st.Sage.Render(clipText(next, max(w-6, 6))))
		}
	}
	rows = append(rows, "", st.Dim.Render(clipText("enter stages this patch · i inspects the full diff", max(w, 8))))
	for len(rows) < h {
		rows = append(rows, "")
	}
	return rows[:h]
}

func (m *Model) stageFocusedWorkflow() {
	c, _, ok := m.workflows.Selected()
	if !ok {
		m.fail("no workflow selected")
		return
	}
	if len(c.Values) == 0 {
		m.fail("workflow " + c.Name + " carries no patch")
		return
	}
	patch := c.Values.Clone()
	if bindings := patch["keybind"]; len(bindings) > 0 {
		patch["keybind"] = append(append([]string(nil), m.staged["keybind"]...), bindings...)
	}
	m.stagePatch(patch, "workflow · "+c.Name)
	if len(m.overrides) > 0 {
		m.fail("staged " + c.Name + " · overrides " + strings.Join(m.overrides, ", "))
		return
	}
	m.say("staged workflow " + c.Name)
}

// ----- health -----

// rebuildHealth composes version, config sources and diagnostics into one list.
func (m *Model) rebuildHealth() {
	items := []core.Choice{}
	if m.state.Version != "" {
		items = append(items, core.Choice{
			ID:          "health:version",
			Name:        "Ghostty " + m.state.Version,
			Description: m.state.GhosttyPath,
			Category:    "version",
			Source:      "binary",
			Installed:   true,
		})
	}
	for i, src := range m.state.Sources {
		items = append(items, core.Choice{
			ID:          "health:source:" + strconv.Itoa(i),
			Name:        shortPath(src),
			Description: src,
			Category:    "config source",
			Source:      src,
			Installed:   true,
		})
	}
	for i, d := range m.diagItems {
		items = append(items, core.Choice{
			ID:          "health:diag:" + strconv.Itoa(i),
			Name:        d.Title,
			Description: d.Detail,
			Category:    d.Level,
			Source:      "diagnostic",
		})
	}
	m.health.SetItems(items)
}

func shortPath(p string) string {
	if p == "" {
		return "(unknown)"
	}
	parts := strings.Split(strings.TrimRight(p, "/"), "/")
	if len(parts) <= 3 {
		return p
	}
	return ".../" + strings.Join(parts[len(parts)-3:], "/")
}

func (m *Model) healthPane(w, h int) []string {
	st := m.styles
	c, _, ok := m.health.Selected()
	rows := []string{
		kv(st, "ghostty", orDash(m.state.Version), w),
		kv(st, "binary", orDash(shortPath(m.state.GhosttyPath)), w),
		kv(st, "config", orDash(shortPath(m.state.ConfigPath)), w),
		kv(st, "revision", orDash(clipText(m.state.Revision, 16)), w),
		kv(st, "warnings", strconv.Itoa(len(m.state.Warnings)), w),
		"",
	}
	if len(m.state.Warnings) > 0 {
		rows = append(rows, st.Err.Render("configuration warnings"))
		for _, wrn := range m.state.Warnings {
			rows = append(rows, st.Dim.Render("  "+clipText(wrn, max(w-3, 6))))
		}
		rows = append(rows, "")
	}
	if ok {
		rows = append(rows, st.Text.Render(clipText(c.Name, max(w, 8))))
		rows = append(rows, wrapLines(c.Description, max(w, 8), 3)...)
	} else {
		rows = append(rows, st.Dim.Render("no diagnostics reported"))
	}
	rows = append(rows, "",
		st.Dim.Render(clipText("writes are always backed up before a same-directory atomic rename", max(w, 8))),
		st.Dim.Render(clipText("undo rejects external edits and restores the original bytes", max(w, 8))))
	for len(rows) < h {
		rows = append(rows, "")
	}
	return rows[:h]
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

// ----- export -----

func (m *Model) exportPane(w, h int) []string {
	st := m.styles
	c, _, ok := m.exports.Selected()
	format := "ghostty"
	rows := []string{}
	if ok {
		format = c.Category
		rows = append(rows, st.Text.Render(clipText(c.Name, max(w, 8))))
		rows = append(rows, wrapLines(c.Description, max(w, 8), 3)...)
		rows = append(rows, "")
	}
	preview := m.exportPreview(format)
	if preview == "" {
		rows = append(rows, st.Dim.Render("no palette available to export"))
	} else {
		lines := splitLines(preview)
		if len(lines) > h-len(rows)-3 {
			lines = lines[:max(h-len(rows)-3, 1)]
		}
		for _, l := range lines {
			rows = append(rows, st.Sage.Render(clipText(l, max(w, 8))))
		}
	}
	rows = append(rows, "", st.Dim.Render(clipText("enter writes this to a file you choose · the CLI can print it to stdout", max(w, 8))))
	for len(rows) < h {
		rows = append(rows, "")
	}
	return rows[:h]
}

func (m *Model) exportPreview(format string) string {
	if m.engine == nil {
		return ""
	}
	text, err := m.engine.ExportTheme(m.mergedValues(), format)
	if err != nil {
		return ""
	}
	return text
}

// stageFocusedMore routes enter for the More section.
func (m *Model) stageFocusedMore() {
	switch m.more {
	case PaneEffects:
		m.toggleEffect()
	case PaneSettings:
		m.openSettingEditor()
	case PaneWorkflows:
		m.stageFocusedWorkflow()
	case PaneExport:
		m.openExportPrompt()
	}
}
