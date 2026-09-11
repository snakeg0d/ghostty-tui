// Package core implements the Ghostty Studio engine: it discovers the
// installed Ghostty, resolves the effective configuration graph, catalogs
// themes, fonts, looks, effects and workflows, and stages, validates, applies
// and undoes configuration changes without ever clobbering unrelated user
// configuration.
//
// The engine is UI agnostic. The Bubble Tea front end in internal/ui consumes
// exactly the API declared in this package.
package core

// Section names accepted by Engine.Catalog.
const (
	SectionThemes    = "themes"
	SectionFonts     = "fonts"
	SectionDiscover  = "discover"
	SectionLooks     = "looks"
	SectionEffects   = "effects"
	SectionWorkflows = "workflows"
)

// Palette is the color set a terminal specimen needs to render itself.
//
// Every field is a lower-case #rrggbb string, or empty when the source did not
// define it. ANSI holds the 16 base palette entries; missing entries are empty.
type Palette struct {
	Background string
	Foreground string
	Cursor     string
	Selection  string
	ANSI       [16]string
}

// Choice is one selectable catalog entry. ID is stable and includes section and
// source identity (for example "theme:nord", "font:menlo", "effect:none",
// "look:everyday"). Consumers must stage the ID returned by the engine rather
// than rebuilding it.
type Choice struct {
	ID          string
	Name        string
	Description string
	Source      string
	Category    string
	Tags        []string
	Palette     Palette
	// Values is a staged configuration patch, see Values for patch semantics.
	Values Values
	// Installed reports whether the underlying artifact is already present
	// locally. It is only ever set from a real observation.
	Installed bool
	URL       string
	License   string
}

// Setting describes one Ghostty configuration key.
//
// Kind is one of: bool, int, float, string, enum, list, color, path, duration,
// size, keybind, reference. Options holds the enumerated values for enum kinds.
// Reload is "reload" when Ghostty can pick the value up with reload_config and
// "restart" when a full restart is required.
type Setting struct {
	Key         string
	Name        string
	Description string
	Group       string
	Default     string
	Kind        string
	Reload      string
	Options     []string
}

// Setting groups. GroupReference marks keys that are cataloged for search and
// inspection but are not part of the curated typed editing schema.
const (
	GroupTypography = "Typography"
	GroupWindow     = "Window"
	GroupCursor     = "Cursor"
	GroupBehavior   = "Behavior"
	GroupShortcuts  = "Shortcuts"
	GroupReference  = "Reference"
)

// Options configures an Engine. Zero values fall back to environment overrides
// and then to platform defaults.
type Options struct {
	// ConfigPath is the Ghostty configuration file to manage. When empty the
	// GHOSTTY_CONFIG_PATH environment variable is consulted and then the
	// platform default locations.
	ConfigPath string
	// StateDir holds favorites, looks, backups and the transaction journal.
	// When empty GHOSTTY_TUI_STATE_DIR, then $XDG_CONFIG_HOME/ghostty-studio,
	// then ~/.config/ghostty-studio are used.
	StateDir string
	// GhosttyPath is the ghostty executable. When empty GHOSTTY_BIN,
	// GHOSTTY_TUI_GHOSTTY_PATH, PATH and the macOS application bundle are
	// consulted in that order.
	GhosttyPath string
	// AssetDir holds bundled shader assets. When empty GHOSTTY_TUI_ASSETS and
	// the assets directory next to the executable are consulted.
	AssetDir string
}

// State is an immutable snapshot of the discovered engine state. Slices and
// maps are copies; callers may retain them.
type State struct {
	// Current holds the effective configuration values, including defaults.
	Current     Values
	ConfigPath  string
	GhosttyPath string
	Version     string
	// Sources lists the configuration files that contribute to Current, in
	// load order. Includes and the managed include file appear here.
	Sources []string
	// Warnings lists non-fatal conditions: degraded native listings, an
	// explicit configuration override, conflicting files, missing assets.
	Warnings []string
	// Revision hashes the whole source graph. It changes whenever any
	// contributing file changes.
	Revision string
}

// Change is a staged patch. Values follows the patch semantics documented on
// Values: absent keys are left alone and an empty value list unsets the managed
// value. Revision is State.Revision captured when the patch was staged; Apply
// and Preview reject a stale revision.
type Change struct {
	Values   Values
	Revision string
	Label    string
}

// Result reports the outcome of a mutating operation. Message always describes
// what actually happened; a reload is never claimed unless it happened.
type Result struct {
	Message    string
	BackupPath string
	Warnings   []string
}

// Diagnostic is one health finding, Level is "info", "warn" or "error".
type Diagnostic struct {
	Level  string
	Title  string
	Detail string
}
