package ui

import "slices"

// Section identifies a top-level studio area.
type Section int

const (
	SecThemes Section = iota
	SecFonts
	SecLooks
	SecMore
)

var sectionNames = [...]string{"Themes", "Fonts", "Looks", "More"}

func (s Section) String() string {
	if int(s) < 0 || int(s) >= len(sectionNames) {
		return "Themes"
	}
	return sectionNames[s]
}

// CatalogSection maps a studio section to the core catalog name.
func (s Section) CatalogSection() string {
	switch s {
	case SecThemes:
		return "themes"
	case SecFonts:
		return "fonts"
	case SecLooks:
		return "looks"
	default:
		return ""
	}
}

// MorePane identifies a pane inside the More section.
type MorePane int

const (
	PaneEffects MorePane = iota
	PaneSettings
	PaneWorkflows
	PaneHealth
	PaneExport
)

var morePaneNames = [...]string{"Effects", "All settings", "Workflows", "Health", "Export palette"}

func (p MorePane) String() string {
	if int(p) < 0 || int(p) >= len(morePaneNames) {
		return "Effects"
	}
	return morePaneNames[p]
}

// CatalogSection maps a pane to the core catalog name, or "" when the pane is
// backed by a non-catalog source.
func (p MorePane) CatalogSection() string {
	switch p {
	case PaneEffects:
		return "effects"
	case PaneWorkflows:
		return "workflows"
	default:
		return ""
	}
}

// FilterMode cycles the theme list filter.
type FilterMode int

const (
	FilterAll FilterMode = iota
	FilterDark
	FilterLight
	FilterFavorites
)

var filterNames = [...]string{"all", "dark", "light", "favorites"}

func (f FilterMode) String() string {
	if int(f) < 0 || int(f) >= len(filterNames) {
		return "all"
	}
	return filterNames[f]
}

func (f FilterMode) next() FilterMode { return FilterMode((int(f) + 1) % len(filterNames)) }

// stagedKeys returns the sorted keys present in a patch.
func stagedKeys(v map[string][]string) []string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// shortPaneName keeps the More strip readable inside the narrow left column.
func shortPaneName(p MorePane) string {
	switch p {
	case PaneSettings:
		return "Settings"
	case PaneExport:
		return "Export"
	default:
		return p.String()
	}
}
