package ui

import (
	"ghostty.studio/internal/core"
)

// lookPane shows the focused appearance preset as a specimen.
func (m *Model) lookPane(w, h int) []string {
	c, _, ok := m.looks.Selected()
	title := "current appearance"
	pal := paletteFromValues(m.mergedValues())
	if ok {
		title = c.Name
		if p := choicePalette(c); p.Background != "" {
			pal = p
		}
	}
	rows := splitLines(renderSpecimen(specimenData{Title: title, Pal: pal}, w, max(h-3, 5), m.styles))
	return append(rows, m.choiceInfoLines(c, ok, w)...)
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
