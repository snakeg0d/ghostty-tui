package ui

import (
	"strings"

	"ghostty.studio/internal/core"
)

// themePane renders the live specimen for the focused theme, with an optional
// pinned second specimen for A/B comparison when the terminal is tall enough.
func (m *Model) themePane(w, h int) []string {
	c, _, ok := m.themes.Selected()
	title := "current appearance"
	pal := paletteFromValues(m.mergedValues())
	if ok {
		title = c.Name
		if p := choicePalette(c); p.Background != "" {
			pal = p
		}
	}

	if m.pinned != "" && h >= 24 && m.pinnedChoice().ID != "" {
		pin := m.pinnedChoice()
		topH := h - 4 - 2
		topH -= topH / 2
		bottomH := h - 4 - topH
		rows := splitLines(renderSpecimen(specimenData{Title: title, Pal: pal}, w, max(topH, 5), m.styles))
		rows = append(rows, "")
		rows = append(rows, splitLines(renderSpecimen(specimenData{Title: pin.Name, Pal: choicePalette(pin)}, w, max(bottomH, 5), m.styles))...)
		return append(rows, m.choiceInfoLines(core.Choice{}, false, w)...)
	}

	rows := splitLines(renderSpecimen(specimenData{Title: title, Pal: pal}, w, max(h-3, 5), m.styles))
	if m.pinned != "" {
		rows = append(rows, "", m.styles.Dim.Render("pinned · "+clipText(m.pinnedName(), max(w-10, 8))+"  (taller terminal compares side by side)"))
		return rows
	}
	return append(rows, m.choiceInfoLines(c, ok, w)...)
}

func (m *Model) pinnedChoice() core.Choice {
	for _, c := range m.themes.Filtered() {
		if c.ID == m.pinned {
			return c
		}
	}
	for _, c := range m.themes.items {
		if c.ID == m.pinned {
			return c
		}
	}
	return core.Choice{}
}

func (m *Model) pinnedName() string {
	if c := m.pinnedChoice(); c.ID != "" {
		return c.Name
	}
	return m.pinned
}

// choiceInfoLines renders provenance for the focused choice.
func (m *Model) choiceInfoLines(c core.Choice, ok bool, w int) []string {
	st := m.styles
	if !ok || c.ID == "" {
		return []string{"", st.Dim.Render("no selection")}
	}
	first := c.Source
	if first == "" {
		first = c.Category
	}
	if c.License != "" {
		first = strings.TrimSpace(first + "  ·  " + c.License)
	}
	second := c.Description
	if c.URL != "" {
		if second != "" {
			second = strings.TrimSpace(second) + "  " + c.URL
		} else {
			second = c.URL
		}
	}
	if len(c.Tags) > 0 {
		second = strings.TrimSpace(second + "  ·  " + strings.Join(c.Tags, ", "))
	}
	return []string{
		"",
		st.Sage.Render(clipText(first, max(w, 8))),
		st.Dim.Render(clipText(second, max(w, 8))),
	}
}

// stageTheme stages a complete theme patch: the explicit palette plus its
// managed color keys, never arbitrary commands from theme files.
func (m *Model) stageTheme(c core.Choice) {
	if len(c.Values) == 0 {
		m.fail("theme " + c.Name + " carries no palette values")
		return
	}
	m.stagePatch(c.Values, "theme · "+c.Name)
	m.themes.MarkRecent(c.ID)
	m.themes.Select(c.ID)
	m.say("staged theme " + c.Name)
}

// stageFocusedTheme stages whatever theme row is focused.
func (m *Model) stageFocusedTheme() {
	if c, _, ok := m.themes.Selected(); ok {
		m.stageTheme(c)
		return
	}
	m.fail("no theme selected")
}

func (m *Model) toggleFavorite() {
	c, _, ok := m.activeList().Selected()
	if !ok {
		m.fail("nothing to favorite")
		return
	}
	if m.sec != SecThemes && m.sec != SecFonts {
		m.fail("favorites apply to themes and fonts")
		return
	}
	if m.engine == nil {
		m.fail("core engine unavailable")
		return
	}
	id := c.ID
	err := m.engine.ToggleFavorite(id)
	if err != nil {
		m.fail("favorite: " + err.Error())
		return
	}
	if m.favs == nil {
		m.favs = map[string]bool{}
	}
	m.favs[id] = !m.favs[id]
	if !m.favs[id] {
		delete(m.favs, id)
	}
	m.applyFavorites()
	m.activeList().Select(id)
	if m.favs[id] {
		m.say("favorited " + c.Name)
		return
	}
	m.say("unfavorited " + c.Name)
}

func (m *Model) cycleFilter() {
	if m.sec != SecThemes {
		m.fail("filter applies to themes")
		return
	}
	m.themes.SetFilter(m.themes.Filter().next())
	m.say("filter: " + m.themes.Filter().String())
}

func (m *Model) togglePin() {
	c, _, ok := m.activeList().Selected()
	if !ok || m.sec != SecThemes {
		m.fail("pin a theme to compare")
		return
	}
	if m.pinned == c.ID {
		m.pinned = ""
		m.say("unpinned " + c.Name)
		return
	}
	m.pinned = c.ID
	m.say("pinned " + c.Name + " for A/B")
}
