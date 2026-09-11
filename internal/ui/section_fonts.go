package ui

import (
	"strings"

	"ghostty.studio/internal/core"
)

// fontPane renders the dedicated typography specimen. It never claims to
// render the selected family inline: only Ghostty itself can do that.
func (m *Model) fontPane(w, h int) []string {
	st := m.styles
	list := m.fontsOrDiscover()
	c, _, ok := list.Selected()
	title := "current typography"
	if ok {
		title = c.Name
	}
	pal := paletteFromValues(m.mergedValues())
	rows := splitLines(renderTypeSpecimen(specimenData{Title: title, Pal: pal}, w, max(h-4, 5), st))
	rows = append(rows, "",
		st.Dim.Render(clipText("Inline uses your terminal font", w)),
		st.Mint.Render("p")+st.Dim.Render(" Preview selected font in Ghostty"),
	)
	if ok {
		rows = append(rows, st.Dim.Render(clipText(fontProvenance(c), w)))
	} else {
		rows = append(rows, st.Dim.Render("no family selected"))
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

// fontProvenance states what the catalog actually knows about an entry. Core's
// description already carries availability and style counts, so it is used as
// the headline and never duplicated with another "installed" synonym.
func fontProvenance(c core.Choice) string {
	head := strings.TrimSpace(c.Description)
	if head == "" {
		if c.Installed {
			head = "installed"
		} else {
			head = "not installed"
		}
	}
	return strings.Join(nonEmpty([]string{head, c.License}), "  ·  ")
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
