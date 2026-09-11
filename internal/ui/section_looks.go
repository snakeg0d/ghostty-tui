package ui

import (
	"strings"

	"ghostty.studio/internal/core"
)

// lookPane shows the focused preset as a specimen, then a concise summary of
// what the preset actually composes.
func (m *Model) lookPane(w, h int) []string {
	st := m.styles
	c, _, ok := m.looks.Selected()
	title := "current appearance"
	pal := paletteFromValues(m.mergedValues())
	if ok {
		title = c.Name
		if p := choicePalette(c); p.Background != "" {
			pal = p
		}
	}
	rows := splitLines(renderSpecimen(specimenData{Title: title, Pal: pal}, w, max(h-6, 5), st))
	rows = append(rows, "")
	rows = append(rows, m.lookComposition(c, ok, w)...)
	for len(rows) < h {
		rows = append(rows, "")
	}
	return rows[:h]
}

// lookComposition reports the real values a preset composes, from the catalog
// choice itself rather than from invented prose.
func (m *Model) lookComposition(c core.Choice, ok bool, w int) []string {
	st := m.styles
	out := []string{st.Dim.Render("composition")}
	if !ok || len(c.Values) == 0 {
		out = append(out, st.Dim.Render("  nothing staged by this preset"))
		return out
	}
	if c.Description != "" {
		out = append(out, st.Dim.Render("  "+clipText(c.Description, max(w-2, 6))))
	}
	// Lead with the choices people recognize. Alphabetical config keys buried
	// the font and size below the fold; Inspect already exposes the full patch.
	font := ""
	for _, family := range c.Values["font-family"] {
		if strings.TrimSpace(family) != "" {
			font = family
			break
		}
	}
	if size := c.Values.Last("font-size"); size != "" {
		font = strings.TrimSpace(font + "  ·  " + size + " pt")
	}
	for _, item := range [][2]string{{"type", font}, {"padding", strings.Trim(strings.Join([]string{c.Values.Last("window-padding-x"), c.Values.Last("window-padding-y")}, " × "), " ×")}, {"opacity", c.Values.Last("background-opacity")}} {
		if item[1] != "" {
			out = append(out, "  "+st.Dim.Render(padRight(item[0], 10))+st.Text.Render(clipText(item[1], max(w-12, 6))))
		}
	}
	return out
}

// stageFocusedLook stages a preset patch, composing with anything already
// staged so presets stack predictably.
func (m *Model) stageFocusedLook() {
	c, _, ok := m.looks.Selected()
	if !ok {
		m.fail("no look selected")
		return
	}
	if len(c.Values) == 0 {
		m.fail("look " + c.Name + " carries no values")
		return
	}
	m.stagePatch(c.Values, "look · "+c.Name)
	m.looks.MarkRecent(c.ID)
	m.looks.Select(c.ID)
	m.say("staged look " + c.Name)
}

// stageImportedLook merges an imported preset; core owns the allowlist, so only
// filtered keys arrive here.
func (m *Model) stageImportedLook(c core.Choice) {
	if len(c.Values) == 0 {
		m.fail("imported look carries no allowlisted values")
		return
	}
	m.stagePatch(c.Values, "import · "+c.Name)
	m.say("staged imported look " + c.Name)
}
