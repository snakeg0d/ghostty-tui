package ui

import (
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"ghostty.studio/internal/core"
)

// ----- helpers -----

func testModel() *Model {
	m := New(nil, core.Options{})
	m.SetSize(120, 36)
	return m
}

func press(m *Model, k tea.KeyPressMsg) tea.Cmd {
	_, cmd := m.Update(k)
	return cmd
}

func pressKey(text string) tea.KeyPressMsg {
	switch text {
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "home":
		return tea.KeyPressMsg{Code: tea.KeyHome}
	case "end":
		return tea.KeyPressMsg{Code: tea.KeyEnd}
	case "pgdown":
		return tea.KeyPressMsg{Code: tea.KeyPgDown}
	case "pgup":
		return tea.KeyPressMsg{Code: tea.KeyPgUp}
	case "ctrl+d":
		return tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl}
	case "ctrl+u":
		return tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	}
	r := []rune(text)[0]
	msg := tea.KeyPressMsg{Code: r, Text: text}
	if r >= 'A' && r <= 'Z' {
		msg = tea.KeyPressMsg{Code: r, Text: text, Mod: tea.ModShift}
	}
	return msg
}

func injectCatalog(m *Model, section string, items []core.Choice) {
	m.Update(catalogLoadedMsg{section: section, items: items})
}

func sampleTheme(id, name string, light bool) core.Choice {
	pal := core.Palette{
		Background: "#1e1e2e",
		Foreground: "#cdd6f4",
		Cursor:     "#f5e0dc",
		Selection:  "#585b70",
		ANSI: [16]string{
			"#45475a", "#f38ba8", "#a6e3a1", "#f9e2af",
			"#89b4fa", "#f5c2e7", "#94e2d5", "#bac2de",
			"#585b70", "#f38ba8", "#a6e3a1", "#f9e2af",
			"#89b4fa", "#f5c2e7", "#94e2d5", "#a6b0c4",
		},
	}
	if light {
		pal = core.Palette{Background: "#ffffff", Foreground: "#24292e"}
	}
	return core.Choice{
		ID: "theme:" + id, Name: name, Source: "fixture themes", Category: "dark",
		Palette: pal,
		Values: core.Values{
			"background": {pal.Background},
			"foreground": {pal.Foreground},
		},
	}
}

// loadedModel returns a model with catalogs, settings and state injected.
func loadedModel() *Model {
	m := testModel()
	injectCatalog(m, "themes", []core.Choice{
		sampleTheme("nord", "Nord", false),
		sampleTheme("daylight", "Daylight", true),
		sampleTheme("catppuccin-mocha", "Catppuccin Mocha", false),
	})
	injectCatalog(m, "fonts", []core.Choice{
		{ID: "font:menlo", Name: "Menlo", Source: "system", Installed: true, Values: core.Values{"font-family": {"Menlo"}}},
	})
	injectCatalog(m, "discover", []core.Choice{
		{ID: "cask:font-monaspace", Name: "Monaspace", Source: "Homebrew cask", URL: "https://formulae.brew.sh/cask/font-monaspace", License: "OFL-1.1"},
	})
	injectCatalog(m, "looks", []core.Choice{
		{ID: "look:everyday", Name: "Everyday", Source: "studio", Values: core.Values{"background-opacity": {"0.95"}}},
	})
	injectCatalog(m, "effects", []core.Choice{
		{ID: "effect:none", Name: "None", Category: "none", Values: core.Values{"custom-shader": {""}}},
		{ID: "effect:grain", Name: "Grain", Source: "Bundled effect", Category: "effect",
			Values: core.Values{"custom-shader": {"", "/tmp/grain.glsl"}, "custom-shader-animation": {"false"}}},
		{ID: "effect:scan", Name: "Scanlines", Source: "user shaders", Category: "effect",
			Values: core.Values{"custom-shader": {"", "/tmp/scan.glsl"}, "custom-shader-animation": {"false"}}},
	})
	injectCatalog(m, "workflows", []core.Choice{
		{ID: "workflow:quick", Name: "Quick terminal", Source: "studio",
			Values: core.Values{"keybind": {"global:ctrl+grave_accent=toggle_quick_terminal"}}},
	})
	m.Update(settingsLoadedMsg{items: []core.Setting{
		{Key: "font-size", Name: "Font size", Description: "Point size.", Group: "Typography", Default: "13", Kind: "int", Reload: "reload"},
		{Key: "background-opacity", Name: "Background opacity", Description: "Opacity.", Group: "Window", Default: "1.0", Kind: "float", Reload: "reload"},
		{Key: "cursor-style", Name: "Cursor style", Description: "Cursor shape.", Group: "Cursor", Default: "block", Kind: "enum", Reload: "reload", Options: []string{"block", "bar", "underline"}},
		{Key: "font-family", Name: "Font family", Description: "Font family fallback chain.", Group: "Typography", Kind: "string", Reload: "reload"},
	}})
	m.Update(stateLoadedMsg{st: core.State{
		Current:  core.Values{"font-size": {"14"}},
		Version:  "1.3.1",
		Revision: "rev-1",
		Sources:  []string{"/tmp/studio-test/config"},
	}})
	return m
}

// ----- geometry -----

func assertGeometry(t *testing.T, name, view string, w, h int) {
	t.Helper()
	lines := strings.Split(view, "\n")
	if len(lines) != h {
		t.Fatalf("%s: got %d lines, want %d", name, len(lines), h)
	}
	for i, l := range lines {
		if got := lipgloss.Width(l); got != w {
			t.Fatalf("%s: line %d has width %d, want %d", name, i, got, w)
		}
	}
}

func TestRenderExactGeometry(t *testing.T) {
	for _, size := range []struct{ w, h int }{{80, 24}, {120, 36}, {60, 18}, {200, 50}, {76, 20}} {
		m := loadedModel()
		m.SetSize(size.w, size.h)
		assertGeometry(t, "themes", m.render(), size.w, size.h)
	}
}

func TestRenderGeometryEverySection(t *testing.T) {
	for _, section := range []string{"themes", "fonts", "looks", "effects", "settings", "workflows", "health", "export"} {
		for _, size := range []struct{ w, h int }{{80, 24}, {120, 36}} {
			m := loadedModel()
			m.SetSize(size.w, size.h)
			if !m.SetSectionByName(section) {
				t.Fatalf("section %q not recognised", section)
			}
			assertGeometry(t, section, m.render(), size.w, size.h)
		}
	}
}

func TestRenderGeometryWithModalOpen(t *testing.T) {
	for _, size := range []struct{ w, h int }{{80, 24}, {120, 36}} {
		m := loadedModel()
		m.SetSize(size.w, size.h)
		m.openHelp()
		assertGeometry(t, "help-modal", m.render(), size.w, size.h)
		m.openInspect()
		assertGeometry(t, "inspect-modal", m.render(), size.w, size.h)
		m.openTune()
		assertGeometry(t, "tune-modal", m.render(), size.w, size.h)
	}
}

func TestTooSmallMessage(t *testing.T) {
	m := loadedModel()
	m.SetSize(59, 17)
	view := m.render()
	assertGeometry(t, "too-small", view, 59, 17)
	if !strings.Contains(view, "needs at least 60") {
		t.Fatalf("expected a truthful resize message, got:\n%s", view)
	}
	// Exactly the documented minimum renders the real studio.
	m.SetSize(minWidth, minHeight)
	if strings.Contains(m.render(), "needs at least 60") {
		t.Fatal("minimum size should render the studio, not the resize message")
	}
}

func TestLongNamesAndUnicodeStayInBounds(t *testing.T) {
	m := loadedModel()
	long := strings.Repeat("メナージュ・ドゥ・", 6) + "Très Long Théâtre Name"
	injectCatalog(m, "themes", []core.Choice{
		sampleTheme("long", long, false),
		sampleTheme("unicode", "日本語テーマ・ダーク", false),
	})
	for _, size := range []struct{ w, h int }{{80, 24}, {120, 36}, {60, 18}} {
		m.SetSize(size.w, size.h)
		assertGeometry(t, "long-names", m.render(), size.w, size.h)
	}
}

// ----- search -----

func TestSearchTypesTextInsteadOfRunningShortcuts(t *testing.T) {
	m := loadedModel()
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	press(m, pressKey("/"))
	if !m.searchOn {
		t.Fatal("/ should focus the search field")
	}
	before := m.themes.Cursor()
	// j/k/q/a must be typed, not interpreted as navigation or quit.
	for _, k := range []string{"j", "k", "q", "a"} {
		cmd := press(m, pressKey(k))
		if cmd != nil {
			if _, quit := cmd().(tea.QuitMsg); quit {
				t.Fatalf("typing %q while searching must not quit", k)
			}
		}
	}
	if got := m.search.Value(); got != "j kqa" && got != "jkqa" {
		t.Fatalf("search should hold typed text, got %q", got)
	}
	if m.themes.Cursor() != before {
		t.Fatal("typing in search must not move the list cursor")
	}
	if m.sec != SecThemes {
		t.Fatal("typing in search must not switch sections")
	}
}

func TestSearchFiltersAndClears(t *testing.T) {
	m := loadedModel()
	press(m, pressKey("/"))
	for _, k := range []string{"c", "a", "t"} {
		press(m, pressKey(k))
	}
	if m.themes.Len() == 0 || m.themes.Len() >= m.themes.Total() {
		t.Fatalf("query should narrow the list, got %d of %d", m.themes.Len(), m.themes.Total())
	}
	if got, _, ok := m.themes.Selected(); !ok || !strings.Contains(got.Name, "Catppuccin") {
		t.Fatalf("expected Catppuccin Mocha, got %+v", got)
	}
	press(m, pressKey("esc"))
	if m.search.Value() != "" {
		t.Fatalf("esc should clear the query, got %q", m.search.Value())
	}
	if m.themes.Len() != m.themes.Total() {
		t.Fatal("clearing the query should restore the full list")
	}
	press(m, pressKey("esc"))
	if m.searchOn {
		t.Fatal("a second esc should leave the search field")
	}
}

func TestNoResultsEmptyState(t *testing.T) {
	m := loadedModel()
	m.SetSearch("zzzzz")
	view := m.render()
	if !strings.Contains(view, "No matches for") {
		t.Fatalf("expected a designed empty state, got:\n%s", view)
	}
	assertGeometry(t, "empty-state", view, 120, 36)
}

func TestFavoritesEmptyState(t *testing.T) {
	m := loadedModel()
	m.themes.SetFilter(FilterFavorites)
	view := m.render()
	if !strings.Contains(view, "No favorites yet") {
		t.Fatalf("expected favorites empty state, got:\n%s", view)
	}
}

// ----- shortcuts -----

// TestShortcutTable exercises every displayed binding.
func TestShortcutTable(t *testing.T) {
	cases := []struct {
		name  string
		key   string
		real  bool
		setup func(*Model)
		check func(*testing.T, *Model)
	}{
		{"stage", "enter", false, nil, func(t *testing.T, m *Model) {
			if len(m.staged) == 0 {
				t.Fatal("enter should stage the focused theme")
			}
		}},
		{"apply", "a", true, nil, func(t *testing.T, m *Model) {
			if !m.statusErr || !strings.Contains(m.status, "nothing staged") {
				t.Fatalf("apply without a patch must surface an error, got %q", m.status)
			}
		}},
		{"undo", "u", true, nil, func(t *testing.T, m *Model) {
			if m.busy && m.busyFor != "undo" {
				t.Fatalf("undo should start its own action, got %q", m.busyFor)
			}
		}},
		{"preview", "p", true, func(m *Model) { m.staged = core.Values{"background": {"#000000"}} },
			func(t *testing.T, m *Model) {
				if !m.busy || m.busyFor != "native preview" {
					t.Fatalf("preview should start the native route, got %q", m.busyFor)
				}
			}},
		{"search", "/", false, nil, func(t *testing.T, m *Model) {
			if !m.searchOn {
				t.Fatal("/ should focus search")
			}
		}},
		{"help", "?", false, nil, func(t *testing.T, m *Model) {
			if !m.modal.active() {
				t.Fatal("? should open help")
			}
		}},
		{"quit-clean", "q", false, nil, func(t *testing.T, m *Model) {
			cmd := press(m, pressKey("q"))
			if cmd == nil {
				t.Fatal("clean quit should return a command")
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatal("clean quit should quit")
			}
		}},
		{"tune", "t", false, nil, func(t *testing.T, m *Model) {
			if !m.modal.active() {
				t.Fatal("t should open the tune modal")
			}
		}},
		{"inspect", "i", false, nil, func(t *testing.T, m *Model) {
			if !m.modal.active() {
				t.Fatal("i should open inspect")
			}
		}},
		{"next-section", "tab", false, nil, func(t *testing.T, m *Model) {
			if m.sec != SecFonts {
				t.Fatalf("tab should advance to Fonts, got %v", m.sec)
			}
		}},
		{"prev-section", "shift+tab", false, nil, func(t *testing.T, m *Model) {
			if m.sec != SecMore {
				t.Fatalf("shift+tab should wrap to More, got %v", m.sec)
			}
		}},
		{"sections-1-4", "3", false, nil, func(t *testing.T, m *Model) {
			if m.sec != SecLooks {
				t.Fatalf("3 should select Looks, got %v", m.sec)
			}
		}},
		{"down", "j", false, nil, func(t *testing.T, m *Model) {
			if m.themes.Cursor() != 1 {
				t.Fatalf("j should move down, cursor=%d", m.themes.Cursor())
			}
		}},
		{"up", "k", false, func(m *Model) { m.themes.Move(2) }, func(t *testing.T, m *Model) {
			if m.themes.Cursor() != 1 {
				t.Fatalf("k should move up, cursor=%d", m.themes.Cursor())
			}
		}},
		{"top", "g", false, func(m *Model) { m.themes.Move(2) }, func(t *testing.T, m *Model) {
			if m.themes.Cursor() != 0 {
				t.Fatal("g should jump to the top")
			}
		}},
		{"bottom", "G", false, nil, func(t *testing.T, m *Model) {
			if m.themes.Cursor() != m.themes.Len()-1 {
				t.Fatal("G should jump to the bottom")
			}
		}},
		{"page-down", "ctrl+d", false, nil, func(t *testing.T, m *Model) {
			if m.themes.Cursor() == 0 {
				t.Fatal("ctrl+d should page down")
			}
		}},
		{"page-up", "ctrl+u", false, func(m *Model) { m.themes.MoveBottom() }, func(t *testing.T, m *Model) {
			if m.themes.Cursor() == m.themes.Len()-1 {
				t.Fatal("ctrl+u should page up")
			}
		}},
		{"favorite", "f", true, nil, func(t *testing.T, m *Model) {
			c, _, ok := m.themes.Selected()
			if !ok {
				t.Fatal("no theme to favorite")
			}
			if !m.favs[c.ID] {
				t.Fatalf("f should favorite %s, got %v", c.ID, m.favs)
			}
			// Put it back so the shared favorites file stays stable.
			if err := m.engine.ToggleFavorite(c.ID); err != nil {
				t.Fatalf("restoring favorite: %v", err)
			}
		}},
		{"filter", "l", false, nil, func(t *testing.T, m *Model) {
			if m.themes.Filter() != FilterDark {
				t.Fatalf("l should cycle the filter, got %v", m.themes.Filter())
			}
		}},
		{"pin", "b", false, nil, func(t *testing.T, m *Model) {
			if m.pinned == "" {
				t.Fatal("b should pin the focused theme")
			}
		}},
		{"discover", "d", false, func(m *Model) { m.setSection(SecFonts) }, func(t *testing.T, m *Model) {
			if !m.fontsDiscover {
				t.Fatal("d should switch to the discover shelf")
			}
		}},
		{"install", "I", false, func(m *Model) { m.setSection(SecFonts); m.fontsDiscover = true }, func(t *testing.T, m *Model) {
			if !m.modal.active() {
				t.Fatal("I should open the install review sheet")
			}
		}},
		{"save-look", "s", false, func(m *Model) { m.setSection(SecLooks) }, func(t *testing.T, m *Model) {
			if !m.modal.active() {
				t.Fatal("s should open the save-look sheet")
			}
		}},
		{"export-look", "e", false, func(m *Model) { m.setSection(SecLooks) }, func(t *testing.T, m *Model) {
			if !m.modal.active() {
				t.Fatal("e should open the export-look sheet")
			}
		}},
		{"import-look", "o", false, func(m *Model) { m.setSection(SecLooks) }, func(t *testing.T, m *Model) {
			if !m.modal.active() {
				t.Fatal("o should open the import-look sheet")
			}
		}},
		{"toggle-effect", "space", false, func(m *Model) { m.SetSectionByName("effects"); m.effects.Move(1) }, func(t *testing.T, m *Model) {
			if len(m.stack) != 1 {
				t.Fatalf("space should add the focused effect, stack=%v", m.stack)
			}
		}},
		{"clear-stack", "x", false, func(m *Model) { m.SetSectionByName("effects") }, func(t *testing.T, m *Model) {
			if _, ok := m.staged["custom-shader"]; !ok {
				t.Fatal("x should stage a shader reset")
			}
		}},
		{"reload", "r", true, nil, func(t *testing.T, m *Model) {
			if !m.busy || m.busyFor != "reload" {
				t.Fatalf("r should start a real refresh, busy=%v for=%q", m.busy, m.busyFor)
			}
		}},
		{"more-panes", "]", false, func(m *Model) { m.setSection(SecMore) }, func(t *testing.T, m *Model) {
			if m.more != PaneSettings {
				t.Fatalf("] should advance the More pane, got %v", m.more)
			}
		}},
		{"more-panes-back", "[", false, func(m *Model) { m.setSection(SecMore); m.more = PaneSettings }, func(t *testing.T, m *Model) {
			if m.more != PaneEffects {
				t.Fatalf("[ should step back, got %v", m.more)
			}
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := loadedModel()
			if tc.real {
				m = engineModel(t)
			}
			if tc.setup != nil {
				tc.setup(m)
			}
			if tc.key != "q" {
				press(m, pressKey(tc.key))
			}
			tc.check(t, m)
		})
	}
}

// TestEveryDisplayedBindingIsCovered guarantees no binding is shown that the
// update loop cannot handle.
func TestEveryDisplayedBindingIsCovered(t *testing.T) {
	covered := map[string]bool{
		"enter": true, "a": true, "u": true, "p": true, "/": true, "?": true, "q": true,
		"t": true, "i": true, "tab": true, "shift+tab": true, "1": true, "2": true,
		"3": true, "4": true, "up": true, "k": true, "down": true, "j": true, "g": true,
		"G": true, "ctrl+d": true, "ctrl+u": true, "f": true, "l": true, "b": true,
		"d": true, "I": true, "s": true, "e": true, "o": true, "space": true, "x": true,
		"r": true, "[": true, "]": true,
	}
	for _, section := range []Section{SecThemes, SecFonts, SecLooks, SecMore} {
		m := loadedModel()
		m.sec = section
		for _, group := range m.helpGroups() {
			for _, b := range group {
				keys := b.Keys()
				if len(keys) == 0 {
					continue
				}
				if !covered[keys[0]] {
					t.Fatalf("binding %q (%v) is displayed but not handled in the shortcut table",
						keys[0], b.Help().Desc)
				}
			}
		}
		for _, b := range m.shortBindings() {
			keys := b.Keys()
			if len(keys) == 0 {
				continue
			}
			if !covered[keys[0]] {
				t.Fatalf("footer binding %q is displayed but not handled", keys[0])
			}
		}
	}
}

// ----- dirty quit -----

func TestDirtyQuitOffersDiscardOrCancel(t *testing.T) {
	m := loadedModel()
	press(m, pressKey("enter"))
	if !m.dirty() {
		t.Fatal("staging should make the model dirty")
	}
	cmd := press(m, pressKey("q"))
	if cmd != nil {
		t.Fatal("dirty quit must not quit directly")
	}
	if !m.modal.active() || m.modal.action != "quit-confirm" {
		t.Fatal("dirty quit should open a confirmation")
	}
	// Cancel is the default button and must not quit or drop the patch.
	m.modal.btnIdx = len(m.modal.buttons) - 1
	if got := m.modalEnter(); got != nil {
		if _, ok := got().(tea.QuitMsg); ok {
			t.Fatal("cancel must not quit")
		}
	}
	if !m.dirty() {
		t.Fatal("cancel must keep the staged patch")
	}
	// Discard quits.
	press(m, pressKey("q"))
	m.modal.btnIdx = 0
	got := m.modalEnter()
	if got == nil {
		t.Fatal("discard should return a command")
	}
	if _, ok := got().(tea.QuitMsg); !ok {
		t.Fatal("discard should quit")
	}
}

func TestEscapeAlwaysExitsWithoutQuitting(t *testing.T) {
	m := loadedModel()
	m.openHelp()
	cmd := press(m, pressKey("esc"))
	if m.modal.active() {
		t.Fatal("esc should close a modal")
	}
	if cmd != nil {
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Fatal("esc must never quit")
		}
	}
	press(m, pressKey("/"))
	press(m, pressKey("esc"))
	press(m, pressKey("esc"))
	if m.searchOn {
		t.Fatal("esc should leave search")
	}
}

// ----- effects contract -----

func TestEffectPathSkipsLeadingReset(t *testing.T) {
	c := core.Choice{Values: core.Values{"custom-shader": {"", "/tmp/grain.glsl"}}}
	if got := effectPath(c); got != "/tmp/grain.glsl" {
		t.Fatalf("effectPath = %q, want the non-empty path", got)
	}
	if got := effectPath(core.Choice{Values: core.Values{"custom-shader": {""}}}); got != "" {
		t.Fatalf("reset-only effect should expose no path, got %q", got)
	}
}

func TestEffectStackPreservesLeadingReset(t *testing.T) {
	m := loadedModel()
	m.SetSectionByName("effects")
	if c, _, ok := m.effects.Selected(); !ok || c.ID != "effect:none" {
		t.Fatalf("None should lead the effects catalog, got %+v", c)
	}
	// Stage the first real effect.
	m.effects.Select("effect:grain")
	press(m, pressKey("space"))
	got := m.staged["custom-shader"]
	if len(got) != 2 || got[0] != "" || got[1] != "/tmp/grain.glsl" {
		t.Fatalf("stack must keep one leading reset, got %#v", got)
	}
	if anim := strings.Join(m.staged["custom-shader-animation"], " "); anim != "false" {
		t.Fatalf("animation should stage false with an active stack, got %q", anim)
	}
	// Add a second effect: order is preserved after the single reset.
	m.effects.Select("effect:scan")
	press(m, pressKey("space"))
	got = m.staged["custom-shader"]
	if len(got) != 3 || got[0] != "" || got[1] != "/tmp/grain.glsl" || got[2] != "/tmp/scan.glsl" {
		t.Fatalf("ordered stack expected, got %#v", got)
	}
	// Toggle the first one off: only the reset and the remaining shader stay.
	m.effects.Select("effect:grain")
	press(m, pressKey("space"))
	got = m.staged["custom-shader"]
	if len(got) != 2 || got[0] != "" || got[1] != "/tmp/scan.glsl" {
		t.Fatalf("removing an effect should keep the reset, got %#v", got)
	}
	// Clear stages a lone reset, never an empty slice (which would revive
	// inherited shaders the catalog never saw).
	press(m, pressKey("x"))
	got = m.staged["custom-shader"]
	if len(got) != 1 || got[0] != "" {
		t.Fatalf("clear must stage exactly one reset, got %#v", got)
	}
	if anim := strings.Join(m.staged["custom-shader-animation"], " "); anim != "false" {
		t.Fatalf("animation stays explicitly false, got %q", anim)
	}
	// None stages the same reset.
	m.effects.Select("effect:none")
	m.staged = core.Values{}
	press(m, pressKey("space"))
	got = m.staged["custom-shader"]
	if len(got) != 1 || got[0] != "" {
		t.Fatalf("None must stage exactly one reset, got %#v", got)
	}
}

func TestClearStackWorksWithoutCatalogEntries(t *testing.T) {
	m := loadedModel()
	m.SetSectionByName("effects")
	// An inherited stack from the user's own config that the catalog cannot name.
	injectCatalog(m, "effects", []core.Choice{{ID: "effect:none", Name: "None", Category: "none", Values: core.Values{"custom-shader": {""}}}})
	m.stack = nil
	press(m, pressKey("x"))
	got := m.staged["custom-shader"]
	if len(got) != 1 || got[0] != "" {
		t.Fatalf("clear must reset an unknown inherited stack, got %#v", got)
	}
}

// ----- settings -----

func TestStagedSettingValueTakesPrecedence(t *testing.T) {
	m := loadedModel()
	if got := m.currentSettingValue("font-size"); got != "14" {
		t.Fatalf("live value expected, got %q", got)
	}
	m.stagePatch(core.Values{"font-size": {"18"}}, "setting · font-size")
	got := m.currentSettingValue("font-size")
	if !strings.HasPrefix(got, "18") || !strings.Contains(got, "(staged)") {
		t.Fatalf("staged value must win and be labelled, got %q", got)
	}
	m.stagePatch(core.Values{"font-size": nil}, "setting · font-size")
	if got := m.currentSettingValue("font-size"); !strings.Contains(got, "(reset staged)") {
		t.Fatalf("a staged reset must be visible, got %q", got)
	}
}

func TestSettingEditorValidatesAndStagesByKey(t *testing.T) {
	m := loadedModel()
	m.SetSectionByName("settings")
	m.settingsL.Select("setting:font-size")
	press(m, pressKey("enter"))
	edit, ok := m.modal.payload.(settingEditPayload)
	if !m.modal.active() || !ok || edit.key != "font-size" || edit.repeated {
		t.Fatalf("enter should edit the focused setting key, got %+v", m.modal.payload)
	}
	m.modal.fields[0].input.SetValue("not-a-number")
	press(m, pressKey("enter")) // move to buttons
	m.modal.btnIdx = 0
	m.submitModal()
	if !m.statusErr || !strings.Contains(m.status, "expected a whole number") {
		t.Fatalf("invalid typed value must be rejected before staging, got %q", m.status)
	}
	press(m, pressKey("enter"))
	m.modal.fields[0].input.SetValue("17")
	m.modal.focusButtons()
	m.modal.btnIdx = 0
	press(m, pressKey("enter"))
	if got := strings.Join(m.staged["font-size"], " "); got != "17" {
		t.Fatalf("valid value should stage under the setting key, got %q", got)
	}
}

// ----- staging and apply failure surfaces -----

func TestWorkflowConflictIsSurfaced(t *testing.T) {
	m := loadedModel()
	m.Update(stateLoadedMsg{st: core.State{
		Current:  core.Values{"keybind": {"global:ctrl+grave_accent=nope"}},
		Revision: "rev-1",
	}})
	m.SetSectionByName("workflows")
	m.workflows.Select("workflow:quick")
	press(m, pressKey("enter"))
	if !m.statusErr || !strings.Contains(m.status, "overrides") {
		t.Fatalf("an overriding keybind must be disclosed, got %q", m.status)
	}
	if !strings.Contains(m.render(), "keybind") {
		t.Fatal("the staged patch should be visible in the pending summary")
	}
}

// TestSearchMatchesNameFuzzyButProseContiguously pins the two match strengths:
// a subsequence finds a theme by name or ID, while descriptions, tags, sources
// and categories only match when the query appears intact.
func TestSearchMatchesNameFuzzyButProseContiguously(t *testing.T) {
	m := testModel()
	m.SetSize(120, 36)
	tokyo := sampleTheme("tokyo-night", "TokyoNight", false)
	tokyo.Tags = []string{"curated"}
	bathory := sampleTheme("black-metal-bathory", "Black Metal Bathory", false)
	bathory.Source = "Ghostty bundle"
	bathory.Category = "dark, curated"
	bathory.Description = "Black and white metal palette."
	kitty := sampleTheme("kitty-low-contrast", "Kitty Low Contrast", false)
	kitty.Source = "Ghostty bundle"
	kitty.Category = "dark, curated"
	prose := sampleTheme("flexoki", "Flexoki Dark", false)
	prose.Tags = []string{"tokyo drift"}
	injectCatalog(m, "themes", []core.Choice{tokyo, bathory, kitty, prose})

	m.SetSearch("tokyo")
	names := map[string]bool{}
	for _, c := range m.themes.Filtered() {
		names[c.Name] = true
	}
	if !names["TokyoNight"] {
		t.Error("a name subsequence (tokyo in TokyoNight) should match")
	}
	if !names["Flexoki Dark"] {
		t.Error("a contiguous tag match should match")
	}
	if names["Black Metal Bathory"] {
		t.Error("scattered letters across combined fields must not match")
	}
	if names["Kitty Low Contrast"] {
		t.Error("scattered letters across combined fields must not match")
	}

	// The subsequence still works for the ID, not just the display name.
	m.SetSearch("tn")
	if !names["TokyoNight"] {
		t.Error("sanity: the fuzzy path should find TokyoNight")
	}
	m.SetSearch("tn")
	if m.themes.Find(tokyo.ID) < 0 {
		t.Error("an ID subsequence should match")
	}
}

// TestInfoModalScrolling proves long bodies are reachable with page and motion
// keys, and that arrow keys stay with a focused text field.
func TestInfoModalScrolling(t *testing.T) {
	m := loadedModel()
	m.SetSize(80, 24)
	body := make([]string, 0, 60)
	for i := range 60 {
		body = append(body, "line "+strconv.Itoa(i))
	}
	m.infoModal("Long", "overflow", body)
	if m.modal.scroll != 0 {
		t.Fatal("a fresh modal starts at the top")
	}
	press(m, pressKey("pgdown"))
	if m.modal.scroll == 0 {
		t.Fatal("pgdown should scroll the body")
	}
	after := m.modal.scroll
	press(m, pressKey("ctrl+d"))
	if m.modal.scroll <= after {
		t.Fatal("ctrl+d should page further down")
	}
	before := m.modal.scroll
	press(m, pressKey("pgup"))
	if m.modal.scroll >= before {
		t.Fatal("pgup should scroll back up")
	}
	press(m, pressKey("ctrl+u"))
	press(m, pressKey("ctrl+u"))
	if m.modal.scroll < 0 {
		t.Fatal("scrolling must never go negative")
	}
	// The rendered view stays inside the terminal while scrolled.
	assertGeometry(t, "scrolled-modal", m.render(), 80, 24)

	// A focused field keeps the arrow keys: they must not jump to buttons.
	m.closeModal()
	m.openSettingKeyEditor("font-size")
	if m.modal.focusKind != 0 {
		t.Fatal("the value field should have focus")
	}
	press(m, pressKey("left"))
	if m.modal.focusKind != 0 {
		t.Fatal("left/right in a focused field must not move focus to the buttons")
	}
}

// TestStatusStaysConcise pins the compact apply/undo feedback, including the
// reload instruction, inside an 80-column footer.
func TestStatusStaysConcise(t *testing.T) {
	m := engineModel(t)
	press(m, pressKey("enter"))
	cmd := m.applyStaged()
	if cmd == nil {
		t.Fatal("apply should run")
	}
	msg, ok := cmd().(actionDoneMsg)
	if !ok {
		t.Fatal("apply should report through actionDoneMsg")
	}
	if msg.err != nil {
		t.Fatalf("apply failed: %v", msg.err)
	}
	m.Update(msg)
	plain := m.status
	if !strings.HasPrefix(plain, "Saved") {
		t.Fatalf("apply status should lead with Saved, got %q", plain)
	}
	for _, want := range []string{"reload Ghostty", "u undo"} {
		if !strings.Contains(plain, want) {
			t.Errorf("apply status should keep %q, got %q", want, plain)
		}
	}
	if len([]rune(plain)) > 60 {
		t.Errorf("status is too long for a compact footer: %q", plain)
	}
	if !strings.Contains(plain, "reload") {
		t.Error("the reload instruction must survive compaction")
	}
	if m.lastDetail == "" {
		t.Error("the detailed core message should be retained for Inspect")
	}
	if view := m.render(); !strings.Contains(view, "Saved") {
		t.Error("the concise status should be visible in the footer")
	}
	assertGeometry(t, "after-apply", m.render(), 120, 36)
}

// TestRepeatedFontFallbackRoundTrip keeps a font fallback chain intact through
// the advanced editor instead of collapsing it into one space-joined string.
func TestRepeatedFontFallbackRoundTrip(t *testing.T) {
	m := loadedModel()
	m.stagePatch(core.Values{"font-family": {"JetBrains Mono", "Menlo"}}, "font · JetBrains Mono")

	m.openSettingKeyEditor("font-family")
	edit, ok := m.modal.payload.(settingEditPayload)
	if !ok || !edit.repeated {
		t.Fatalf("font-family should use the repeated editor, got %+v", m.modal.payload)
	}
	if got := m.modal.fields[0].input.Value(); got != `["JetBrains Mono","Menlo"]` {
		t.Fatalf("the editor should prepopulate JSON, got %q", got)
	}

	m.modal.fields[0].input.SetValue(`["Berkeley Mono", "Menlo"]`)
	m.modal.focusButtons()
	m.modal.btnIdx = 0
	press(m, pressKey("enter"))
	if got := m.staged["font-family"]; len(got) != 3 || got[0] != "" || got[1] != "Berkeley Mono" || got[2] != "Menlo" {
		t.Fatalf("the fallback chain must stage as separate values, got %#v", got)
	}
	merged := m.mergedValues()["font-family"]
	if len(merged) != 3 || merged[2] != "Menlo" {
		t.Fatalf("merged values should keep the whole chain, got %#v", merged)
	}

	// Invalid JSON is rejected before anything is staged.
	m.openSettingKeyEditor("font-family")
	m.modal.fields[0].input.SetValue("JetBrains Mono, Menlo")
	m.modal.focusButtons()
	m.modal.btnIdx = 0
	press(m, pressKey("enter"))
	if !m.statusErr || !strings.Contains(m.status, "JSON array") {
		t.Fatalf("non-JSON input must be rejected, got %q", m.status)
	}
	if got := m.staged["font-family"]; len(got) != 3 || got[1] != "Berkeley Mono" {
		t.Fatalf("a rejected edit must not change the staged chain, got %#v", got)
	}

	// The reset button stages the nil-reset, which is how a managed value is
	// handed back to the user's own configuration.
	m.openSettingKeyEditor("font-family")
	m.modal.focusButtons()
	m.modal.btnIdx = 1
	press(m, pressKey("enter"))
	got, present := m.staged["font-family"]
	if !present || len(got) != 0 {
		t.Fatalf("reset should stage an empty value list, got %#v (present=%v)", got, present)
	}
}

// specimenText renders a specimen and strips styling so the sample's content
// can be asserted.
func specimenText(p core.Palette, title string, w, h int) string {
	return stripANSI(renderSpecimen(specimenData{Title: title, Pal: p}, w, h, newStyles()))
}

// TestSpecimenAdaptiveComposition proves the sample drops whole blocks instead
// of clipping them, keeps the palette ribbon, and always travels with the
// SAMPLE tag.
func TestSpecimenAdaptiveComposition(t *testing.T) {
	pal := sampleTheme("nord", "Nord", false).Palette
	for _, h := range []int{6, 8, 10, 14, 18, 24, 30} {
		got := specimenText(pal, "Nord", 78, h)
		assertGeometry(t, "specimen", got, 78, h)
		if !strings.Contains(got, "SAMPLE") {
			t.Errorf("h=%d: the SAMPLE tag must always be present", h)
		}
		if !strings.Contains(got, "bg #1e1e2e") {
			t.Errorf("h=%d: the palette ribbon metadata must survive every height", h)
		}
		// The git diff is one block: both rows or neither.
		hasMinus := strings.Contains(got, "- background = #101518")
		hasPlus := strings.Contains(got, "+ background = #1e1e2e")
		if hasMinus != hasPlus {
			t.Errorf("h=%d: a half-drawn diff block leaked through:\n%s", h, got)
		}
		// No invented work is ever reported.
		for _, banned := range []string{"compiled", "packages in", "files changed", "reviewed"} {
			if strings.Contains(strings.ToLower(got), banned) {
				t.Errorf("h=%d: specimen reports invented work: %q", h, banned)
			}
		}
	}
	// A tall specimen shows more blocks than a short one, never fewer.
	short := specimenText(pal, "Nord", 78, 10)
	tall := specimenText(pal, "Nord", 78, 30)
	if strings.Count(tall, "\n") <= strings.Count(short, "\n") {
		t.Fatal("adaptive composition should reveal more at greater height")
	}
	if !strings.Contains(tall, "func Look(n int) []string {") {
		t.Error("a roomy specimen should show the code sample")
	}
}

// TestSpecimenRibbonRhythm rejects the old continuous barcode: the ribbon is a
// known number of separated cells.
func TestSpecimenRibbonRhythm(t *testing.T) {
	for _, tc := range []struct{ listW, count, cell, sep int }{
		{34, 6, 1, 1},
		{32, 5, 1, 1},
		{29, 4, 1, 1},
		{24, 3, 1, 1},
	} {
		count, cell, sep := swatchPlan(tc.listW)
		if count != tc.count || cell != tc.cell || sep != tc.sep {
			t.Errorf("swatchPlan(%d) = (%d,%d,%d), want (%d,%d,%d)",
				tc.listW, count, cell, sep, tc.count, tc.cell, tc.sep)
		}
		if sep < 1 {
			t.Errorf("swatchPlan(%d): swatches must be separated", tc.listW)
		}
	}
}

// TestTwoLineRowsKeepWholeNames proves two-line entries give the name the whole
// first line, and that compact terminals fall back to one line per entry.
func TestTwoLineRowsKeepWholeNames(t *testing.T) {
	m := loadedModel()
	long := "Catppuccin Mocha Frappé Extra Long Name"
	m.themes.SetItems([]core.Choice{sampleTheme("catppuccin-mocha", long, false)})

	m.SetSize(120, 36)
	if m.rowLines() != 2 {
		t.Fatal("a roomy terminal should use two-line entries")
	}
	m.layout()
	body := m.themes.View(m.styles, m.rowOpts())
	if len(body) != m.themes.height {
		t.Fatalf("two-line list should fill its extent, got %d of %d", len(body), m.themes.height)
	}
	first := stripANSI(body[0])
	if !strings.Contains(first, "Catppuccin Mocha Frappé") {
		t.Fatalf("the first line should carry the name, got %q", first)
	}
	second := stripANSI(body[1])
	if !strings.Contains(second, "dark") {
		t.Fatalf("the second line should carry the appearance class, got %q", second)
	}
	if strings.Contains(second, "fixture themes") {
		t.Fatalf("the second line must not repeat the source, got %q", second)
	}
	if strings.Contains(first, "▪") {
		t.Fatalf("the first line must stay purely typographic, got %q", first)
	}
	if !strings.Contains(second, "▪") {
		t.Fatalf("the second line should carry the swatches, got %q", second)
	}
	// The list honours the two-line grid, so the second entry starts at row 2.
	if m.themes.cap() != m.themes.height/2 {
		t.Fatalf("two-line capacity should halve: cap=%d height=%d", m.themes.cap(), m.themes.height)
	}
	assertGeometry(t, "two-line", m.render(), 120, 36)
	m.themes.SetFavorites(map[string]bool{"theme:catppuccin-mocha": true})
	if got := strings.Count(strings.Join(m.themes.View(m.styles, m.rowOpts()), "\n"), "★"); got != 1 {
		t.Fatalf("a roomy favorite should have one marker, got %d", got)
	}

	// Compact terminals use one line per entry and still show a useful name.
	m.SetSize(80, 24)
	if m.rowLines() != 1 {
		t.Fatal("a compact terminal should use single-line entries")
	}
	m.layout()
	view := stripANSI(m.render())
	if !strings.Contains(view, "Catppuccin Mocha") {
		t.Errorf("a compact list should still show a useful name:\n%s", view)
	}
	assertGeometry(t, "one-line", m.render(), 80, 24)
}

// TestTypographySpecimenIsTruthful pins the font sample: real letterforms, the
// ambiguity pairs, three weights, and an explicit statement that the inline
// sample is not the selected family.
func TestTypographySpecimenIsTruthful(t *testing.T) {
	m := loadedModel()
	m.SetSectionByName("fonts")
	m.SetSize(120, 36)
	view := stripANSI(m.render())
	for _, want := range []string{
		"Inline uses your terminal font",
		"Preview selected font in Ghostty",
		"0 O o",
		"1 l I |",
		"0123456789",
		"regular   The quick brown fox",
		"bold      Sphinx of black quartz",
		"italic    Pack my box with five dozen",
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ",
		"abcdefghijklmnopqrstuvwxyz",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("typography specimen should contain %q:\n%s", want, view)
		}
	}
	for _, banned := range []string{"rendered in Menlo", "your font is", "Menlo selected"} {
		if strings.Contains(view, banned) {
			t.Errorf("the inline sample must not claim to render the selected font: %q", banned)
		}
	}
	// The theme code sample must not be reused for fonts.
	if strings.Contains(view, "func Look(n int)") {
		t.Error("the fonts pane should use the typography specimen, not the code sample")
	}
	// It stays truthful and complete at the compact size too.
	m.SetSize(80, 24)
	small := stripANSI(m.render())
	assertGeometry(t, "fonts-compact", m.render(), 80, 24)
	if !strings.Contains(small, "Inline uses your terminal font") {
		t.Error("the compact fonts pane must keep the truthful note")
	}
	if !strings.Contains(small, "0 O o") {
		t.Error("the compact fonts pane must keep the ambiguity row")
	}
}

// TestFooterKeepsStatusVisibleWithLongStagedSummary protects the behaviour that
// Saved/reload/error text is reserved before the staged summary.
func TestFooterKeepsStatusVisibleWithLongStagedSummary(t *testing.T) {
	m := loadedModel()
	m.SetSize(80, 24)
	long := core.Values{}
	for i := range 12 {
		long["very-long-setting-key-number-"+strconv.Itoa(i)] = []string{strings.Repeat("v", 20)}
	}
	m.stagePatch(long, "long patch")
	m.say("Saved  ·  reload Ghostty ⌘⇧,  ·  u undo")
	view := m.render()
	assertGeometry(t, "long-summary", view, 80, 24)
	plain := stripANSI(view)
	if !strings.Contains(plain, "reload Ghostty") {
		t.Fatalf("the status must survive a long staged summary:\n%s", plain)
	}
	if !strings.Contains(plain, "u undo") {
		t.Fatal("the undo hint must survive a long staged summary")
	}
}

// TestSpecimenHeadingIsReserved proves the heading owns its own row and never
// overwrites a sample row, at every panel height.
func TestSpecimenHeadingIsReserved(t *testing.T) {
	pal := sampleTheme("nord", "Nord", false).Palette
	for _, h := range []int{6, 7, 9, 12, 16, 22, 30} {
		got := specimenText(pal, "Nord", 78, h)
		assertGeometry(t, "specimen-heading", got, 78, h)
		lines := strings.Split(got, "\n")
		if len(lines) < 3 {
			continue
		}
		if !strings.Contains(lines[1], "Nord") || !strings.Contains(lines[1], "SAMPLE") {
			t.Errorf("h=%d: the first interior row must be the heading, got %q", h, lines[1])
		}
		// Compare the interior only: the frame's own border sits on the row.
		if interior := strings.TrimSpace(strings.Trim(lines[2], "│ ")); interior != "" {
			t.Errorf("h=%d: the heading must be followed by a blank row, got %q", h, interior)
		}
		// Semantic groups are whole: the code block and the diff are all-or-nothing.
		if strings.Contains(got, "package studio") {
			for _, want := range []string{"func Look(n int) []string {", "return theme.ANSI[:n]", "}"} {
				if !strings.Contains(got, want) {
					t.Errorf("h=%d: the code group is incomplete, missing %q", h, want)
				}
			}
		}
		if strings.Contains(got, "- background = #101518") != strings.Contains(got, "+ background = #1e1e2e") {
			t.Errorf("h=%d: the diff group is incomplete:\n%s", h, got)
		}
		// Every interior row stays inside the frame: no wrapped samples.
		for i, l := range lines {
			if lipgloss.Width(l) != 78 {
				t.Fatalf("h=%d: row %d is %d cells wide", h, i, lipgloss.Width(l))
			}
		}
	}
}

// TestFullAlphabetSurvivesAtRoomierSizes pins the whole alphabet, not a prefix.
func TestFullAlphabetSurvivesAtRoomierSizes(t *testing.T) {
	m := loadedModel()
	m.SetSectionByName("fonts")
	m.SetSize(120, 36)
	view := stripANSI(m.render())
	if !strings.Contains(view, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		t.Error("the uppercase alphabet must be complete")
	}
	if !strings.Contains(view, "abcdefghijklmnopqrstuvwxyz") {
		t.Error("the lowercase alphabet must be complete")
	}
	// The compact pane keeps the heading, one alphabet line, the ambiguity row
	// and one weight sample, and still states the caveat.
	m.SetSize(80, 24)
	small := stripANSI(m.render())
	assertGeometry(t, "fonts-compact", m.render(), 80, 24)
	for _, want := range []string{"abcdefghijklmnopqrstuvwxyz", "0 O o", "regular   The quick brown fox", "SAMPLE"} {
		if !strings.Contains(small, want) {
			t.Errorf("the compact type specimen should keep %q", want)
		}
	}
	if !strings.Contains(small, "Inline uses your terminal font") {
		t.Error("the compact pane must keep the truthful caveat")
	}
	if !strings.Contains(small, "Preview selected font in Ghostty") {
		t.Error("the compact pane must keep the native preview action visible")
	}
}

// TestThemePaneStatesSourceOnce protects the single source line beneath the
// specimen, and the footer's sole ownership of contrast.
func TestThemePaneStatesSourceOnce(t *testing.T) {
	m := engineModel(t)
	m.SetSectionByName("themes")
	m.SetSize(120, 36)
	view := stripANSI(m.render())
	if n := strings.Count(view, "contrast"); n != 1 {
		t.Errorf("contrast belongs to the footer only, found %d mentions", n)
	}
	if !strings.Contains(view, "i inspect palette") {
		t.Error("the second info line should point at Inspect")
	}
	if m.pinned != "" {
		return
	}
	// The pinned hint must describe the real, stacked comparison.
	m.themes.Select(m.themes.Filtered()[0].ID)
	m.togglePin()
	m.themes.Move(2)
	// A compact terminal cannot stack two specimens, so it states the real
	// comparison instead of the old "side by side" claim.
	m.SetSize(80, 24)
	pinned := stripANSI(m.render())
	if strings.Contains(pinned, "side by side") {
		t.Error("the comparison is stacked, not side by side")
	}
	if !strings.Contains(pinned, "pinned ·") {
		t.Error("the pinned specimen should be labelled")
	}
	if !strings.Contains(pinned, "stacks both specimens") {
		t.Error("the compact hint should describe the stacked comparison")
	}
	// A roomy terminal stacks the two specimens for real.
	m.SetSize(120, 36)
	stacked := stripANSI(m.render())
	if !strings.Contains(stacked, m.pinnedName()) {
		t.Error("the pinned specimen should be visible when stacked")
	}
}
