package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// darkTheme is a theme file body with the keys Ghostty themes carry.
func darkTheme(background, foreground string) string {
	return "background = " + background + "\nforeground = " + foreground + "\n" +
		"cursor-color = #ffffff\npalette = 0=#101010\npalette = 1=#ff0000\n"
}

// sparseTheme leaves cursor, cursor-text and selection colours undefined.
func sparseTheme(background, foreground string) string {
	return "background = " + background + "\nforeground = " + foreground + "\n" +
		"palette = 0=#101010\n"
}

func TestThemeCatalogOrderingCategoriesAndIdentity(t *testing.T) {
	fx := newFixture(t)
	fx.writeBundleTheme("Nord", darkTheme("#2e3440", "#d8dee9"))
	fx.writeBundleTheme("Zebra Dark", darkTheme("#101010", "#eeeeee"))
	fx.writeBundleTheme("GitHub Light", darkTheme("#ffffff", "#111111"))
	fx.writeBundleTheme("Zebra-Dark", darkTheme("#202020", "#dddddd"))
	fx.writeBundleTheme("README.md", "# not a theme\n")
	fx.writeTheme("Onenord", "#2e3440", "#d8dee9")
	e := fx.openEngine()

	themes := e.Catalog(SectionThemes)
	if len(themes) != 5 {
		t.Fatalf("expected 5 themes, got %d: %+v", len(themes), themes)
	}
	byName := map[string]Choice{}
	for _, theme := range themes {
		byName[theme.Name] = theme
	}
	if themes[0].Name != "Nord" {
		t.Errorf("the curated theme must sort first, got %q", themes[0].Name)
	}
	if themes[0].ID != "theme:nord" {
		t.Errorf("theme id = %q, want theme:nord", themes[0].ID)
	}
	for _, theme := range themes {
		if theme.Palette.Background == "" || theme.Palette.Foreground == "" {
			t.Errorf("theme %s must expose a palette", theme.ID)
		}
		if theme.Values.First("theme") != "" || len(theme.Values["theme"]) != 1 {
			t.Errorf("theme %s must reset the theme directive", theme.ID)
		}
	}
	if got := byName["Nord"].Category; got != "dark" {
		t.Errorf("Nord category = %q, want dark", got)
	}
	if got := byName["GitHub Light"].Category; got != "light" {
		t.Errorf("GitHub Light category = %q, want light", got)
	}
	if got := byName["GitHub Light"].Palette.ANSI[1]; got != "#ff0000" {
		t.Errorf("theme palette must be parsed, got %q", got)
	}
	if !strings.Contains(byName["Nord"].Description, "contrast") {
		t.Errorf("description should report contrast: %q", byName["Nord"].Description)
	}
	// Zebra Dark and Zebra-Dark collide as slugs and must stay distinguishable.
	if byName["Zebra Dark"].ID == byName["Zebra-Dark"].ID {
		t.Errorf("colliding theme slugs must be disambiguated: %q", byName["Zebra Dark"].ID)
	}
	for _, theme := range themes {
		if strings.Contains(theme.Name, "README") {
			t.Error("a file without theme keys must not become a theme")
		}
	}
	if _, ok := byName["Onenord"]; !ok {
		t.Error("user themes must be offered")
	}
}

func TestThemePatchResetsUndefinedAppearanceKeys(t *testing.T) {
	fx := newFixture(t)
	fx.writeBundleTheme("Sparse", sparseTheme("#101010", "#eeeeee"))
	fx.writeBundleTheme("Full", darkTheme("#202020", "#abcdef"))
	e := fx.openEngine()

	themes := map[string]Choice{}
	for _, theme := range e.Catalog(SectionThemes) {
		themes[theme.Name] = theme
	}
	sparse := themes["Sparse"].Values
	for _, key := range []string{"cursor-color", "cursor-text", "selection-background", "selection-foreground"} {
		list, ok := sparse.Get(key)
		if !ok || len(list) != 1 || list[0] != "" {
			t.Errorf("theme patch must reset %s, got %v (present %t)", key, list, ok)
		}
	}
	if sparse.First("background") != "#101010" || sparse.First("foreground") != "#eeeeee" {
		t.Errorf("theme patch must carry the defined colours: %v", sparse)
	}
	if list := sparse["palette"]; len(list) != 2 || list[0] != "" || list[1] != "0=#101010" {
		t.Errorf("theme patch must reset then set palette entries, got %v", list)
	}
	full := themes["Full"].Values
	if got := full.First("cursor-color"); got != "#ffffff" {
		t.Errorf("defined colours must be carried verbatim, got %q", got)
	}
}

func TestThemeLookupIsCaseInsensitiveAndFallsBackToBackgroundFamily(t *testing.T) {
	fx := newFixture(t)
	fx.writeBundleTheme("GitHub Light", darkTheme("#ffffff", "#111111"))
	e := fx.openEngine()

	themes := e.Catalog(SectionThemes)
	picked, ok := pickTheme(fx.themeEntries(t, e), []string{"no-such-theme"}, boolPtr(true))
	if !ok || picked.name != "GitHub Light" {
		t.Errorf("a light preset must fall back to a light theme, got %q ok=%t", picked.name, ok)
	}
	if len(themes) == 0 {
		t.Fatal("expected at least one theme")
	}
	index := themeIndex(fx.themeEntries(t, e))
	if _, ok := index("github light"); !ok {
		t.Error("theme lookup must be case-insensitive")
	}
	if _, ok := index("GitHub Light"); !ok {
		t.Error("theme lookup must accept the exact name")
	}
	if _, ok := index("nope"); ok {
		t.Error("unknown themes must not resolve")
	}
}

// themeEntries exposes the loaded theme entries for assertions.
func (fx *testFixture) themeEntries(t *testing.T, e *Engine) []themeEntry {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.themes
}

func TestFontCatalogKeepsFallbacksAndResetsStyles(t *testing.T) {
	fx := newFixture(t)
	fx.writeConfig("font-family = Menlo\nfont-family = Iosevka\n")
	e := fx.openEngine()

	fonts := e.Catalog(SectionFonts)
	if len(fonts) != 3 {
		t.Fatalf("expected the three fake families, got %d", len(fonts))
	}
	var jetbrains Choice
	for _, font := range fonts {
		if font.Name == "JetBrains Mono" {
			jetbrains = font
		}
	}
	if jetbrains.ID != "font:jetbrains-mono" {
		t.Errorf("font id = %q, want font:jetbrains-mono", jetbrains.ID)
	}
	want := []string{"", "JetBrains Mono", "Menlo", "Iosevka"}
	if got := jetbrains.Values["font-family"]; !equalStrings(got, want) {
		t.Errorf("font chain = %v, want %v (fallbacks preserved, primary first)", got, want)
	}
	for _, key := range []string{"font-family-bold", "font-family-italic", "font-family-bold-italic"} {
		if got := jetbrains.Values[key]; !equalStrings(got, []string{""}) {
			t.Errorf("%s must be reset so it follows the regular family, got %v", key, got)
		}
	}
	if !jetbrains.Installed {
		t.Error("families reported by +list-fonts are installed")
	}
}

func TestDiscoverShelfMarksInstalledAndRefusesUninstallableFonts(t *testing.T) {
	fx := newFixture(t)
	e := fx.openEngine()

	shelf := e.Catalog(SectionDiscover)
	byName := make(map[string]Choice, len(shelf))
	for _, choice := range shelf {
		byName[choice.Name] = choice
	}
	commercial, okCommercial := byName["Berkeley Mono"]
	iosevka, okIosevka := byName["Iosevka"]
	maple, okMaple := byName["Maple Mono"]
	if !okCommercial || !okIosevka || !okMaple {
		t.Fatalf("the curated discover shelf is incomplete: %+v", shelf)
	}
	if iosevka.Installed != true {
		t.Error("Iosevka is installed in the fixture and must be marked installed")
	}
	if maple.Installed {
		t.Error("Maple Mono is not installed and must not claim to be")
	}
	if commercial.License == "" || commercial.URL == "" {
		t.Error("the commercial entry must still explain its licence and where to get it")
	}
	if _, err := e.InstallFont(commercial.ID); err == nil {
		t.Error("a commercial typeface must not be installable")
	}
	if fx.runner.brewCommandCount() != 0 {
		t.Error("refusing a commercial font must not invoke Homebrew")
	}
	if _, err := e.InstallFont("discover:font-nope"); err == nil {
		t.Error("an unknown id must be rejected")
	}
}

func TestEffectCatalogOffersNoneFirstAndOrderedStackPatches(t *testing.T) {
	fx := newFixture(t)
	screen := fx.writeShader("screen/soft-scanlines.glsl")
	cursor := fx.writeShader("cursors/ripple.glsl")
	e := fx.openEngine()

	effects := e.Catalog(SectionEffects)
	if len(effects) != 3 {
		t.Fatalf("expected None plus two shaders, got %d", len(effects))
	}
	if effects[0].ID != "effect:none" {
		t.Errorf("None must come first, got %q", effects[0].ID)
	}
	if got := effects[0].Values["custom-shader"]; !equalStrings(got, []string{""}) {
		t.Errorf("None must reset the shader stack, got %v", got)
	}
	var found bool
	for _, effect := range effects[1:] {
		stack := effect.Values["custom-shader"]
		if len(stack) != 2 || stack[0] != "" {
			t.Errorf("an effect patch must reset then set, got %v", stack)
			continue
		}
		if stack[1] == screen || stack[1] == cursor {
			found = true
		}
		if !filepath.IsAbs(stack[1]) {
			t.Errorf("effect paths must be absolute, got %q", stack[1])
		}
		if got := effect.Values["custom-shader-animation"]; !equalStrings(got, []string{"false"}) {
			t.Errorf("animation must default to off, got %v", got)
		}
		if !strings.Contains(effect.Description, "renderer") {
			t.Errorf("the renderer-only compilation caveat must be stated: %q", effect.Description)
		}
	}
	if !found {
		t.Error("discovered shaders must appear in the catalog")
	}

	// The effect stack is staged, not written on selection.
	if _, err := os.Stat(filepath.Join(filepath.Dir(fx.config), managedFileName)); !os.IsNotExist(err) {
		t.Error("browsing effects must not write anything")
	}
}

func TestWorkflowCatalogReportsBindingConflicts(t *testing.T) {
	fx := newFixture(t)
	fx.writeConfig("keybind = super+f=toggle_fullscreen\n")
	e := fx.openEngine()

	var search Choice
	for _, workflow := range e.Catalog(SectionWorkflows) {
		if workflow.ID == "workflow:scrollback-search" {
			search = workflow
		}
	}
	if search.ID == "" {
		t.Fatal("the scrollback search workflow is missing")
	}
	if !strings.Contains(search.Description, "Conflict") || !strings.Contains(search.Description, "toggle_fullscreen") {
		t.Errorf("an existing binding must be reported in the description: %q", search.Description)
	}
	var conflicted bool
	for _, tag := range search.Tags {
		if tag == "conflict" {
			conflicted = true
		}
	}
	if !conflicted {
		t.Error("a conflicting workflow must be tagged")
	}
	// The staged bindings are additions only: a reset here would erase the
	// user's bindings and Ghostty's defaults, because the managed include is the
	// last file Ghostty loads.
	if got := search.Values["keybind"]; !equalStrings(got, []string{"super+f=start_search"}) {
		t.Errorf("a workflow must add its binding without a reset, got %v", got)
	}
	if got := search.Values.First("scrollback-limit"); got != "10000000" {
		t.Errorf("scrollback-limit patch = %q", got)
	}
}

func TestLookCatalogComposesRealThemesAndFonts(t *testing.T) {
	fx := newFixture(t)
	fx.writeBundleTheme("Nord", darkTheme("#2e3440", "#d8dee9"))
	fx.writeBundleTheme("GitHub Light", darkTheme("#ffffff", "#111111"))
	fx.writeConfig("font-family = Menlo\n")
	e := fx.openEngine()

	looks := e.Catalog(SectionLooks)
	if len(looks) != 5 {
		t.Fatalf("expected five presets, got %d", len(looks))
	}
	byID := map[string]Choice{}
	for _, look := range looks {
		byID[look.ID] = look
	}
	everyday := byID["look:everyday"]
	if everyday.ID == "" {
		t.Fatal("look:everyday missing")
	}
	if got := everyday.Values.First("background"); got != "#2e3440" {
		t.Errorf("Everyday must use the real Nord palette, got %q", got)
	}
	want := []string{"", "JetBrains Mono", "Menlo"}
	if got := everyday.Values["font-family"]; !equalStrings(got, want) {
		t.Errorf("Everyday font chain = %v, want %v", got, want)
	}
	if got := everyday.Values.First("window-padding-x"); got != "14" {
		t.Errorf("Everyday padding = %q", got)
	}
	daylight := byID["look:daylight"]
	if got := daylight.Values.First("background"); got != "#ffffff" {
		t.Errorf("Daylight must use the light theme, got %q", got)
	}
	if !strings.Contains(everyday.Description, "Nord") || !strings.Contains(everyday.Description, "JetBrains Mono") {
		t.Errorf("the description must name the real artifacts: %q", everyday.Description)
	}
	for _, look := range looks {
		for key := range look.Values {
			if !lookAllowlist[key] {
				t.Errorf("look %s stages %q, which is outside the appearance allowlist", look.ID, key)
			}
		}
	}

	// Without any installed font the preset must not invent one.
	fx.runner.ghostty.fonts = ""
	e2 := fx.openEngine()
	for _, look := range e2.Catalog(SectionLooks) {
		if _, ok := look.Values["font-family"]; ok {
			t.Errorf("look %s must not set a font when none is installed", look.ID)
		}
		if !strings.Contains(look.Description, "current font") {
			t.Errorf("look %s must say the font is untouched: %q", look.ID, look.Description)
		}
	}
}

func TestSettingsReferenceIsCompleteDeduplicatedAndTyped(t *testing.T) {
	fx := newFixture(t)
	fx.runner.ghostty.options = map[string][]string{
		"cursor-style": {"block", "bar", "underline", "block_hollow"},
	}
	fx.runner.ghostty.restartKeys = map[string]bool{"macos-titlebar-style": true}
	fx.runner.ghostty.defaults["keybind"] = []string{"super+shift+,=reload_config", "super+r=reload_config", "super+i=inspector:toggle"}
	fx.runner.known = knownKeys(fx.runner.ghostty.defaults)
	e := fx.openEngine()

	settings := e.Settings()
	seen := map[string]bool{}
	for _, setting := range settings {
		if seen[setting.Key] {
			t.Fatalf("key %q appears twice in the reference", setting.Key)
		}
		seen[setting.Key] = true
		if setting.Name == "" || setting.Description == "" {
			t.Errorf("setting %q needs a name and documentation", setting.Key)
		}
	}
	if len(settings) != len(fx.runner.ghostty.defaults) {
		t.Errorf("reference has %d entries for %d keys", len(settings), len(fx.runner.ghostty.defaults))
	}

	byKey := map[string]Setting{}
	for _, setting := range settings {
		byKey[setting.Key] = setting
	}
	cursor, ok := byKey["cursor-style"]
	if !ok {
		t.Fatal("cursor-style missing from the reference")
	}
	if cursor.Kind != "enum" || !equalStrings(cursor.Options, []string{"block", "bar", "underline", "block_hollow"}) {
		t.Errorf("enum hints must come from the installed docs: kind=%q options=%v", cursor.Kind, cursor.Options)
	}
	if cursor.Group != GroupCursor {
		t.Errorf("cursor-style group = %q, want %q", cursor.Group, GroupCursor)
	}
	if cursor.Default != "block" {
		t.Errorf("cursor-style default = %q, want block", cursor.Default)
	}
	if got := byKey["font-size"].Kind; got != "float" {
		t.Errorf("curated font-size kind = %q, want float", got)
	}
	if got := byKey["macos-titlebar-style"].Reload; got != "restart" {
		t.Errorf("reload semantics must come from the docs, got %q", got)
	}
	if got := byKey["keybind"]; got.Default != "super+shift+,=reload_config" {
		t.Errorf("repeated key default = %q", got.Default)
	} else if !strings.Contains(got.Description, "3 default values") {
		t.Errorf("a repeated key must explain its several defaults: %q", got.Description)
	}
	reference, ok := byKey["window-show-tab-bar"]
	if !ok {
		t.Fatal("every documented key must be searchable")
	}
	if reference.Kind != "reference" || reference.Group != GroupReference {
		t.Errorf("advanced keys must be reference only, got kind=%q group=%q", reference.Kind, reference.Group)
	}
}

func TestSettingsFallBackToTheCuratedSchemaWithoutGhostty(t *testing.T) {
	fx := newFixture(t)
	hideInstalledBundle(t)
	e, err := open(Options{ConfigPath: fx.config, StateDir: fx.state}, fx.runner,
		func(string) string { return "" }, fx.home, func(string) (string, error) { return "", os.ErrNotExist })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	settings := e.Settings()
	if len(settings) != len(curatedSettings) {
		t.Fatalf("expected only the curated schema, got %d entries", len(settings))
	}
	for _, setting := range settings {
		if !strings.Contains(setting.Description, "documentation unavailable") {
			t.Errorf("fallback descriptions must be honest: %q", setting.Description)
		}
		if setting.Group == GroupReference {
			t.Errorf("%s is curated, not a reference key", setting.Key)
		}
	}
}

func TestParseDocDumpHandlesProseBulletsAndRepeatedKeys(t *testing.T) {
	docs := []byte(`# The colour of a search match.
# 
# Valid values:
# 
#   - Hex (` + "`#RRGGBB`" + ` or ` + "`RRGGBB`" + `)
#   - Named X11 color
#   - "cell-foreground" to match the cell foreground color
# 
# The default value is black text on a golden yellow background.
search-foreground = 
# The cursor style.
# 
# Valid values are:
# 
#   * ` + "`block`" + `
#   * ` + "`bar`" + `
# 
# Warning: This setting cannot be reloaded at runtime.
cursor-style = block
# Bindings.
keybind = super+a=reload_config
keybind = super+b=new_tab
`)
	docsParsed := parseDocDump(docs)
	if len(docsParsed) != 3 {
		t.Fatalf("expected three keys, got %d: %+v", len(docsParsed), docsParsed)
	}
	search := docsParsed[0]
	if len(search.options) != 0 {
		t.Errorf("prose value forms must not become enum hints, got %v", search.options)
	}
	cursor := docsParsed[1]
	if !equalStrings(cursor.options, []string{"block", "bar"}) {
		t.Errorf("backticked bullets must become options, got %v", cursor.options)
	}
	if cursor.reload != "restart" {
		t.Errorf("reload semantics = %q, want restart", cursor.reload)
	}
	bindings := docsParsed[2]
	if len(bindings.defaults) != 2 {
		t.Errorf("repeated keys must be folded, got %v", bindings.defaults)
	}
	if len(parseOptions([]string{"Valid values are:", "   * `", "   * `plain`"})) != 1 {
		t.Error("an empty code span must be skipped, not panic")
	}
}

func TestDiffExplainsTheStagedChangeAndConflicts(t *testing.T) {
	fx := newFixture(t)
	// Ghostty documents `language` as requiring a full restart.
	fx.runner.ghostty.restartKeys = map[string]bool{"language": true}
	fx.writeConfig("font-size = 13\nkeybind = super+f=toggle_fullscreen\n")
	e := fx.openEngine()

	diff := e.Diff(Change{Revision: e.State().Revision, Label: "Tune", Values: Values{"font-size": {"16"}}})
	if !strings.Contains(diff, "font-size") || !strings.Contains(diff, "13") || !strings.Contains(diff, "16") {
		t.Errorf("diff must show the change:\n%s", diff)
	}
	unchanged := e.Diff(Change{Revision: e.State().Revision, Label: "Noop", Values: Values{"font-size": {"13"}}})
	if !strings.Contains(unchanged, "No effective change") {
		t.Errorf("an unchanged patch must say so:\n%s", unchanged)
	}
	conflict := e.Diff(Change{
		Revision: e.State().Revision,
		Label:    "Bind",
		Values:   Values{"keybind": {"", "super+f=start_search"}},
	})
	if !strings.Contains(conflict, "rebinds a key") || !strings.Contains(conflict, "toggle_fullscreen") {
		t.Errorf("a keybind conflict must be surfaced:\n%s", conflict)
	}
	stale := e.Diff(Change{Revision: "deadbeef", Label: "Stale", Values: Values{"font-size": {"15"}}})
	if !strings.Contains(stale, "older revision") {
		t.Errorf("a stale revision must be flagged:\n%s", stale)
	}
	restart := e.Diff(Change{Revision: e.State().Revision, Label: "Restart", Values: Values{"language": {"de"}}})
	if !strings.Contains(restart, "restart") {
		t.Errorf("settings that need a restart must be flagged:\n%s", restart)
	}
	injection := e.Diff(Change{Revision: e.State().Revision, Label: "Bad", Values: Values{"font-size": {"13\ncommand = x"}}})
	if !strings.Contains(injection, "Cannot stage") {
		t.Errorf("an unusable patch must be refused in the diff:\n%s", injection)
	}
}

func TestCatalogReturnsCopies(t *testing.T) {
	fx := newFixture(t)
	fx.writeBundleTheme("Nord", darkTheme("#2e3440", "#d8dee9"))
	e := fx.openEngine()

	first := e.Catalog(SectionThemes)
	if len(first) == 0 {
		t.Fatal("expected themes")
	}
	first[0].Name = "mutated"
	first[0].Tags = append(first[0].Tags, "mutated")
	first[0].Values["background"] = []string{"#000000"}

	second := e.Catalog(SectionThemes)
	if second[0].Name == "mutated" {
		t.Error("catalog names must be copies")
	}
	for _, tag := range second[0].Tags {
		if tag == "mutated" {
			t.Error("catalog tags must be copies")
		}
	}
	if got := second[0].Values.First("background"); got == "#000000" {
		t.Error("catalog values must be copies")
	}
	favorites := e.Favorites()
	favorites = append(favorites, "x")
	if len(e.Favorites()) != 0 {
		t.Error("favorites must be copies")
	}
	settings := e.Settings()
	if len(settings) > 0 {
		settings[0].Options = append(settings[0].Options, "mutated")
		if len(e.Settings()[0].Options) == len(settings[0].Options) {
			t.Error("setting options must be copies")
		}
	}
}

func TestFavoritesPersistOnlyWhenToggled(t *testing.T) {
	fx := newFixture(t)
	fx.writeConfig("font-size = 13\n")
	e := fx.openEngine()

	if _, err := os.Stat(e.favoritesPath()); !os.IsNotExist(err) {
		t.Error("reading favorites must not create state")
	}
	if err := e.ToggleFavorite("theme:nord"); err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if err := e.ToggleFavorite("font:menlo"); err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if got := strings.Join(e.Favorites(), ","); got != "font:menlo,theme:nord" {
		t.Errorf("favorites = %q", got)
	}
	data, err := os.ReadFile(e.favoritesPath())
	if err != nil {
		t.Fatalf("read favorites: %v", err)
	}
	if !strings.Contains(string(data), "theme:nord") || !strings.Contains(string(data), lookFormat) {
		t.Errorf("favorites document = %s", data)
	}
	if err := e.ToggleFavorite("theme:nord"); err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if got := strings.Join(e.Favorites(), ","); got != "font:menlo" {
		t.Errorf("removing a favorite failed: %q", got)
	}
	if err := e.ToggleFavorite("   "); err == nil {
		t.Error("an empty identifier must be rejected")
	}
	// A fresh engine must see the persisted list.
	e2 := fx.openEngine()
	if got := strings.Join(e2.Favorites(), ","); got != "font:menlo" {
		t.Errorf("favorites must persist across engines: %q", got)
	}
}

func TestNativeModeUsesGhosttyEffectiveConfiguration(t *testing.T) {
	fx := newFixture(t)
	// No explicit override: the engine reads the platform configuration files
	// and must prefer Ghostty's own effective output.
	native := filepath.Join(fx.home, "Library", "Application Support", "com.mitchellh.ghostty")
	if err := os.MkdirAll(native, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(native, "config"), []byte("font-size = 14\n"), 0o644); err != nil {
		t.Fatalf("write native config: %v", err)
	}
	fx.runner.ghostty.changesOnly = "font-size = 19\nbackground = #123456\n"
	e, err := fx.openEngineWithEnv(Options{StateDir: fx.state, GhosttyPath: fx.bin}, nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	state := e.State()
	if len(state.Sources) != 1 || state.Sources[0] != filepath.Join(native, "config") {
		t.Errorf("sources = %v", state.Sources)
	}
	if got := state.Current.Last("font-size"); got != "19" {
		t.Errorf("the engine must report Ghostty's own effective value, got %q", got)
	}
	if got := state.Current.Last("background"); got != "#123456" {
		t.Errorf("effective colours must come from Ghostty, got %q", got)
	}
}

func TestConfigGhosttyShadowsTheLegacyName(t *testing.T) {
	fx := newFixture(t)
	dir := filepath.Join(fx.home, ".config", "ghostty")
	if err := os.WriteFile(filepath.Join(dir, "config"), []byte("font-size = 12\n"), 0o644); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.ghostty"), []byte("font-size = 18\n"), 0o644); err != nil {
		t.Fatalf("write modern config: %v", err)
	}
	e, err := fx.openEngineWithEnv(Options{StateDir: fx.state, GhosttyPath: fx.bin}, nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	state := e.State()
	if len(state.Sources) != 1 || filepath.Base(state.Sources[0]) != "config.ghostty" {
		t.Fatalf("config.ghostty must win in the same directory: %v", state.Sources)
	}
	if state.ConfigPath != filepath.Join(dir, "config.ghostty") {
		t.Errorf("managed config path = %q", state.ConfigPath)
	}
	var shadowed bool
	for _, warning := range state.Warnings {
		if strings.Contains(warning, "has no effect") {
			shadowed = true
		}
	}
	if !shadowed {
		t.Errorf("a shadowed file must be reported: %v", state.Warnings)
	}
}
