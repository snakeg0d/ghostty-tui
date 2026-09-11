package ui

import (
	"strings"

	"ghostty.studio/internal/core"
)

// fontPane renders the typography specimen plus provenance for the focused
// family. The inline specimen cannot change the terminal's own font, which the
// pane states honestly.
func (m *Model) fontPane(w, h int) []string {
	c, _, ok := m.fontsOrDiscover().Selected()
	title := "current typography"
	if ok {
		title = c.Name
	}
	pal := paletteFromValues(m.mergedValues())
	rows := splitLines(renderSpecimen(specimenData{Title: title, Pal: pal}, w, max(h-6, 5), m.styles))

	st := m.styles
	note := "Sample uses your terminal font · p previews this family in Ghostty"
	rows = append(rows, "", st.Dim.Render(clipText(note, max(w, 8))))

	if ok {
		line := c.Source
		if c.License != "" {
			line = strings.TrimSpace(line + "  ·  " + c.License)
		}
		if line == "" {
			line = c.Category
		}
		rows = append(rows, st.Sage.Render(clipText(line+"  ·  "+installHint(c), max(w, 8))))
	} else {
		rows = append(rows, st.Dim.Render("no font family selected"))
	}
	for len(rows) < h {
		rows = append(rows, "")
	}
	return rows[:h]
}

func (m *Model) fontsOrDiscover() *choiceList {
	if m.fontsDiscover {
		return m.discover
	}
	return m.fonts
}

func installHint(c core.Choice) string {
	if c.Installed {
		return "installed"
	}
	if c.URL != "" {
		return "not installed"
	}
	return "available"
}

func (m *Model) toggleDiscover() {
	if m.sec != SecFonts {
		m.fail("discover applies to fonts")
		return
	}
	m.fontsDiscover = !m.fontsDiscover
	m.search.Placeholder = m.searchPlaceholder()
	if m.fontsDiscover {
		m.say("discover shelf")
	} else {
		m.say("installed families")
	}
}

// stageFocusedFont stages an installed family, preserving the fallback chain
// that core computed into the choice's Values.
func (m *Model) stageFocusedFont() {
	list := m.fontsOrDiscover()
	c, _, ok := list.Selected()
	if !ok {
		m.fail("no font selected")
		return
	}
	if m.fontsDiscover {
		m.fail("press I to review and install " + c.Name)
		return
	}
	if len(c.Values) == 0 {
		m.fail("font " + c.Name + " carries no family values")
		return
	}
	m.stagePatch(c.Values, "font · "+c.Name)
	list.MarkRecent(c.ID)
	list.Select(c.ID)
	m.say("staged font " + c.Name)
}
