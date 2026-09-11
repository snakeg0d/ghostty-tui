package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"ghostty.studio/internal/core"
)

// Snapshot renders the exact terminal view for deterministic visual QA.
//
// It builds a real model, fills it synchronously with the engine's state and
// catalogs, applies the requested section and search query, and returns
// Model.View().Content: the same string the live TUI renders for that size.
//
// The CLI entry point is:
//
//	ui.Snapshot(engine, opts, width, height, section, query)
//
// section accepts "themes", "fonts", "looks", "more", the More pane names
// ("effects", "settings", "settings"/"all settings", "workflows", "health",
// "export") and "" for the default (themes). An unknown section leaves the
// default in place.
func Snapshot(engine *core.Engine, opts core.Options, width, height int, section, query string) string {
	m := New(engine, opts)
	m.SetSize(width, height)
	m.Hydrate()
	m.SetSectionByName(section)
	m.SetSearch(query)
	return m.View().Content
}

// Hydrate fills the model synchronously instead of waiting on async commands.
// It exists for snapshot rendering and tests; the interactive app uses Init.
func (m *Model) Hydrate() {
	if m.engine == nil {
		return
	}
	m.Update(stateLoadedMsg{st: m.engine.State()})
	m.Update(settingsLoadedMsg{items: m.engine.Settings()})
	m.Update(favsLoadedMsg{ids: m.engine.Favorites()})
	for _, section := range catalogSections {
		m.Update(catalogLoadedMsg{section: section, items: m.engine.Catalog(section)})
	}
	m.Update(diagLoadedMsg{items: m.engine.Diagnose()})
}

// catalogSections lists every catalog the studio loads.
var catalogSections = []string{"themes", "fonts", "discover", "looks", "effects", "workflows"}

// SetSectionByName selects a section (and More pane) by name. It reports
// whether the name was recognised. Unlike keyboard navigation it stays quiet:
// a QA snapshot must not carry a status line the keyboard path would add.
func (m *Model) SetSectionByName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	switch name {
	case "", "themes", "theme":
		m.selectSection(SecThemes, m.more)
		return true
	case "fonts", "font":
		m.selectSection(SecFonts, m.more)
		return true
	case "looks", "look":
		m.selectSection(SecLooks, m.more)
		return true
	case "more":
		m.selectSection(SecMore, m.more)
		return true
	case "settings", "all settings", "config":
		m.selectSection(SecMore, PaneSettings)
		return true
	case "export", "export palette", "palette":
		m.selectSection(SecMore, PaneExport)
		return true
	}
	for pane, paneName := range morePaneNames {
		if strings.EqualFold(name, paneName) {
			m.selectSection(SecMore, MorePane(pane))
			return true
		}
	}
	return false
}

// selectSection switches section and More pane without touching the status.
func (m *Model) selectSection(s Section, pane MorePane) {
	m.sec = s
	if s == SecMore {
		m.more = pane
	}
	if m.search.Value() != "" {
		m.applyQuery()
	}
	m.layout()
}

// SetSearch applies a query to every list, exactly like typing in the search
// field, and leaves the search field blurred.
func (m *Model) SetSearch(query string) {
	m.search.SetValue(query)
	m.search.CursorEnd()
	m.searchOn = false
	m.search.Blur()
	m.applyQuery()
}

// snapshotView is the View for the current model state. It exists so tests can
// assert against the very same rendering path the CLI uses.
func (m *Model) snapshotView() tea.View { return m.View() }
