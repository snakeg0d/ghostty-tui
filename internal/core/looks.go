package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// lookAllowlist is the set of keys a Look may contain. Looks are appearance and
// typography only: nothing here can run a command, bind a key or relocate the
// configuration.
var lookAllowlist = map[string]bool{
	"theme":                          true,
	"background":                     true,
	"foreground":                     true,
	"cursor-color":                   true,
	"cursor-text":                    true,
	"selection-background":           true,
	"selection-foreground":           true,
	"palette":                        true,
	"font-family":                    true,
	"font-family-bold":               true,
	"font-family-italic":             true,
	"font-family-bold-italic":        true,
	"font-size":                      true,
	"font-feature":                   true,
	"font-variation":                 true,
	"font-style":                     true,
	"font-thicken":                   true,
	"font-thicken-strength":          true,
	"adjust-cell-height":             true,
	"adjust-cell-width":              true,
	"adjust-font-baseline":           true,
	"adjust-underline-position":      true,
	"adjust-underline-thickness":     true,
	"adjust-strikethrough-position":  true,
	"adjust-strikethrough-thickness": true,
	"adjust-overline-position":       true,
	"window-padding-x":               true,
	"window-padding-y":               true,
	"window-padding-balance":         true,
	"window-padding-color":           true,
	"background-opacity":             true,
	"background-blur":                true,
	"background-image-opacity":       true,
	"cursor-style":                   true,
	"cursor-style-blink":             true,
	"cursor-opacity":                 true,
	"custom-shader":                  true,
	"custom-shader-animation":        true,
	"alpha-blending":                 true,
	"minimum-contrast":               true,
	"unfocused-split-opacity":        true,
	"unfocused-split-fill":           true,
	"split-divider-color":            true,
	"search-foreground":              true,
	"search-background":              true,
	"search-selected-foreground":     true,
	"search-selected-background":     true,
	"bold-color":                     true,
	"faint-opacity":                  true,
}

// lookFile is the on-disk Look format.
type lookFile struct {
	Format  string `json:"format"`
	Name    string `json:"name"`
	Created string `json:"created"`
	Values  Values `json:"values"`
}

// lookFormat identifies the Look file schema.
const lookFormat = "ghostty-studio/look@1"

// lookPreset is one editorial appearance preset.
type lookPreset struct {
	slug        string
	name        string
	mood        string
	description string
	themes      []string
	// light selects the required background family when no named theme exists.
	light  *bool
	fonts  []string
	values Values
}

// lookPresets is the editorial Look catalog. Each preset composes a real theme
// from the theme catalog with a real installed font and a small number of typed
// settings.
var lookPresets = []lookPreset{
	{
		slug:        "everyday",
		name:        "Everyday",
		mood:        "calm",
		description: "Nord-family palette, generous padding, no visual noise.",
		themes:      []string{"nord", "catppuccin-mocha", "tokyonight", "flexoki-dark", "gruvbox-dark"},
		fonts:       []string{"JetBrains Mono", "Iosevka", "Menlo"},
		values: Values{
			"font-size":          []string{"14"},
			"window-padding-x":   []string{"14"},
			"window-padding-y":   []string{"10"},
			"background-opacity": []string{"1"},
			"cursor-style":       []string{"block"},
		},
	},
	{
		slug:        "daylight",
		name:        "Daylight",
		mood:        "light",
		description: "A genuinely light palette with roomy padding for daytime work.",
		themes:      []string{"github-light", "catppuccin-latte", "gruvbox-light", "flexoki-light", "rose-pine-dawn"},
		light:       boolPtr(true),
		fonts:       []string{"JetBrains Mono", "Iosevka", "Menlo"},
		values: Values{
			"font-size":          []string{"14"},
			"window-padding-x":   []string{"18"},
			"window-padding-y":   []string{"12"},
			"background-opacity": []string{"1"},
			"cursor-style":       []string{"bar"},
		},
	},
	{
		slug:        "dense",
		name:        "Dense",
		mood:        "small typography",
		description: "Smaller type and tight padding to fit more output on screen.",
		themes:      []string{"everforest-dark", "flexoki-dark", "gruvbox-dark", "tokyonight"},
		fonts:       []string{"Iosevka", "JetBrains Mono", "Menlo"},
		values: Values{
			"font-size":        []string{"12"},
			"window-padding-x": []string{"8"},
			"window-padding-y": []string{"6"},
			"cursor-style":     []string{"block"},
		},
	},
	{
		slug:        "presentation",
		name:        "Presentation",
		mood:        "large and clear",
		description: "Large type, wide padding and a light palette that reads from across a room.",
		themes:      []string{"github-light", "github", "catppuccin-latte", "gruvbox-light", "nord-light"},
		light:       boolPtr(true),
		fonts:       []string{"JetBrains Mono", "Iosevka", "Menlo"},
		values: Values{
			"font-size":          []string{"20"},
			"window-padding-x":   []string{"28"},
			"window-padding-y":   []string{"20"},
			"font-thicken":       []string{"false"},
			"background-opacity": []string{"1"},
			"cursor-style":       []string{"block"},
		},
	},
	{
		slug:        "after-hours",
		name:        "After Hours",
		mood:        "warm",
		description: "Warm dark palette with a little translucency and blur for late sessions.",
		themes:      []string{"gruvbox-dark", "everforest-dark", "rose-pine", "kanagawa", "nord"},
		fonts:       []string{"JetBrains Mono", "Iosevka", "Menlo"},
		values: Values{
			"font-size":          []string{"14"},
			"window-padding-x":   []string{"16"},
			"window-padding-y":   []string{"12"},
			"background-opacity": []string{"0.92"},
			"background-blur":    []string{"24"},
			"cursor-style":       []string{"bar"},
			"cursor-style-blink": []string{"true"},
		},
	},
}

func boolPtr(v bool) *bool { return &v }

// lookChoices builds the Look catalog: the editorial presets in their curated
// order, then the user's saved Looks in the order the library reader produced.
func lookChoices(themes []themeEntry, fonts []installedFont, current Values, saved []Choice) []Choice {
	choices := make([]Choice, 0, len(lookPresets)+len(saved))
	for _, preset := range lookPresets {
		values := Values{}
		var palette Palette
		themeLabel := "no theme in the catalog"
		if theme, ok := pickTheme(themes, preset.themes, preset.light); ok {
			for key, list := range themePatch(theme) {
				values[key] = list
			}
			palette = theme.palette
			themeLabel = theme.name
		}
		fontLabel := "your current font"
		if family, ok := pickFont(fonts, preset.fonts); ok {
			for key, list := range fontPatch(family, current) {
				values[key] = list
			}
			fontLabel = family
		}
		for _, key := range preset.values.Keys() {
			values[key] = append([]string(nil), preset.values[key]...)
		}
		description := fmt.Sprintf("%s · %s + %s · %s", preset.description, themeLabel, fontLabel, preset.mood)
		choices = append(choices, Choice{
			ID:          "look:" + preset.slug,
			Name:        preset.name,
			Description: description,
			Source:      "Ghostty Studio",
			Category:    "look",
			Tags:        []string{"preset", slugify(preset.mood)},
			Palette:     palette,
			Values:      values,
		})
	}
	for _, look := range saved {
		choices = append(choices, copyChoice(look))
	}
	return choices
}

// copyChoice deep-copies one catalog entry.
func copyChoice(choice Choice) Choice {
	clone := choice
	clone.Tags = append([]string(nil), choice.Tags...)
	clone.Values = choice.Values.Clone()
	return clone
}

// pickTheme chooses the first named theme that exists, then falls back to any
// theme with the requested background family.
func pickTheme(themes []themeEntry, wants []string, light *bool) (themeEntry, bool) {
	for _, want := range wants {
		for _, theme := range themes {
			if theme.slug == want && theme.palette.Background != "" {
				return theme, true
			}
		}
		for _, theme := range themes {
			if strings.HasPrefix(theme.slug, want) && theme.palette.Background != "" {
				return theme, true
			}
		}
	}
	if light != nil {
		for _, theme := range themes {
			if theme.palette.Background != "" && theme.light == *light {
				return theme, true
			}
		}
	}
	for _, theme := range themes {
		if theme.palette.Background != "" {
			return theme, true
		}
	}
	return themeEntry{}, false
}

// pickFont chooses the first preferred family that is actually installed.
func pickFont(fonts []installedFont, wants []string) (string, bool) {
	for _, want := range wants {
		for _, font := range fonts {
			if strings.EqualFold(font.name, want) {
				return font.name, true
			}
		}
	}
	return "", false
}

// savedLookSource labels library entries.
const savedLookSource = "Saved Look"

// looksDir is where named Looks are stored.
func (e *Engine) looksDir() string {
	return filepath.Join(e.paths.stateDir, "looks")
}

// readLookLibrary reads the Look library. Files are validated with the same rules as
// an imported Look, so a saved Look can never smuggle a key the studio would
// refuse to import. Unreadable or invalid files warn and are skipped: a broken
// library entry must not stop the studio from starting or hide the others.
func (e *Engine) readLookLibrary() ([]Choice, []string) {
	dir := e.looksDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, []string{fmt.Sprintf("the Look library %s could not be read: %v", dir, err)}
		}
		return nil, nil
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}
		names = append(names, entry.Name())
	}
	// Sorted before reading so the catalog order never depends on the
	// filesystem's iteration order.
	sort.Strings(names)

	choices := make([]Choice, 0, len(names))
	var warnings []string
	for _, name := range names {
		path := filepath.Join(dir, name)
		id := "look:" + strings.TrimSuffix(slugify(strings.TrimSuffix(name, filepath.Ext(name))), "-")
		choice, err := e.importLookFile(path, id, savedLookSource)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("saved Look %s was skipped: %v", path, err))
			continue
		}
		choices = append(choices, choice)
	}
	sort.SliceStable(choices, func(i, j int) bool {
		li, lj := strings.ToLower(choices[i].Name), strings.ToLower(choices[j].Name)
		if li != lj {
			return li < lj
		}
		return choices[i].ID < choices[j].ID
	})
	return choices, warnings
}

// filterLook keeps only allowlisted values and reports how many keys were
// dropped.
func filterLook(values Values) (Values, int) {
	out := Values{}
	dropped := 0
	for _, key := range values.Keys() {
		if !lookAllowlist[key] {
			dropped++
			continue
		}
		out[key] = append([]string(nil), values[key]...)
	}
	return out, dropped
}

// SaveLook writes a named Look into the studio's state directory.
func (e *Engine) SaveLook(name string, values Values) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errors.New("a Look needs a name")
	}
	if err := checkValues(values); err != nil {
		return err
	}
	filtered, _ := filterLook(values)
	if len(filtered) == 0 {
		return errors.New("a Look needs at least one appearance setting")
	}
	if err := checkPathValues(filtered); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := ensureDir(e.looksDir()); err != nil {
		return err
	}
	path := filepath.Join(e.looksDir(), slugify(trimmed)+".json")
	if err := writeLookFile(path, lookFile{
		Format:  lookFormat,
		Name:    trimmed,
		Created: time.Now().UTC().Format(time.RFC3339),
		Values:  filtered,
	}); err != nil {
		return err
	}
	// Re-read the library and rebuild the catalog so the saved Look is
	// selectable from this session without a full refresh.
	e.reloadLookLibrary()
	return nil
}

// ExportLook writes a Look to an explicit path, stripping everything that is
// not an appearance setting.
func (e *Engine) ExportLook(name string, values Values, path string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errors.New("a Look needs a name")
	}
	if err := checkValues(values); err != nil {
		return err
	}
	filtered, _ := filterLook(values)
	if len(filtered) == 0 {
		return errors.New("there is nothing to export: no appearance settings are staged")
	}
	if err := checkPathValues(filtered); err != nil {
		return err
	}
	target := strings.TrimSpace(path)
	if target == "" {
		target = filepath.Join(e.looksDir(), slugify(trimmed)+".json")
	} else if e.paths.home != "" {
		target = expandHome(target, e.paths.home)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := ensureDir(filepath.Dir(target)); err != nil {
		return err
	}
	return writeLookFile(target, lookFile{
		Format:  lookFormat,
		Name:    trimmed,
		Created: time.Now().UTC().Format(time.RFC3339),
		Values:  filtered,
	})
}

// writeLookFile serialises a Look.
func writeLookFile(path string, look lookFile) error {
	data, err := json.MarshalIndent(look, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, append(data, '\n'), 0o600)
}

// ImportLook reads an external Look file and validates every value before it
// can be staged. Untrusted shader paths are checked against the filesystem, and
// any key outside the appearance allowlist is dropped.
func (e *Engine) ImportLook(path string) (Choice, error) {
	target := strings.TrimSpace(path)
	if target == "" {
		return Choice{}, errors.New("no file path was given to import")
	}
	if e.paths.home != "" {
		target = expandHome(target, e.paths.home)
	}
	return e.importLookFile(target, "", "Imported Look")
}

// importLookFile validates one Look document. A non-empty id makes the file a
// library entry: the identifier comes from the file name, so a saved Look keeps
// a stable ID even if its display name is edited.
func (e *Engine) importLookFile(target, id, source string) (Choice, error) {
	data, err := os.ReadFile(target)
	if err != nil {
		return Choice{}, fmt.Errorf("reading %s: %w", target, err)
	}
	var look lookFile
	if err := json.Unmarshal(data, &look); err != nil {
		return Choice{}, fmt.Errorf("%s is not a Ghostty Studio Look file: %w", target, err)
	}
	if look.Format != "" && look.Format != lookFormat {
		return Choice{}, fmt.Errorf("%s uses the unknown Look format %q", target, look.Format)
	}
	if len(look.Values) == 0 {
		return Choice{}, fmt.Errorf("%s contains no appearance settings", target)
	}
	name := strings.TrimSpace(look.Name)
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(target), filepath.Ext(target))
	}
	filtered, dropped := filterLook(look.Values)
	if err := checkValues(filtered); err != nil {
		return Choice{}, fmt.Errorf("%s contains an unusable value: %w", target, err)
	}
	for _, shader := range filtered["custom-shader"] {
		if shader == "" {
			continue
		}
		if !filepath.IsAbs(shader) {
			return Choice{}, fmt.Errorf("%s references the shader %q with a relative path, which is not allowed", target, shader)
		}
		info, err := os.Stat(shader)
		if err != nil || !info.Mode().IsRegular() {
			return Choice{}, fmt.Errorf("%s references the shader %q, which does not exist", target, shader)
		}
		if !shaderExtensions[strings.ToLower(filepath.Ext(shader))] {
			return Choice{}, fmt.Errorf("%s references %q, which is not a GLSL shader", target, shader)
		}
	}
	if len(filtered) == 0 {
		return Choice{}, fmt.Errorf("%s contains no allowed appearance settings (%d unrelated settings were ignored)", target, dropped)
	}
	if id == "" {
		id = "look:" + slugify(name)
	}
	description := fmt.Sprintf("Imported from %s · %d appearance settings", e.displayPath(target), len(filtered))
	tags := []string{"imported"}
	if source == savedLookSource {
		description = fmt.Sprintf("Saved Look · %d appearance settings", len(filtered))
		tags = []string{"saved"}
	}
	choice := Choice{
		ID:          id,
		Name:        name,
		Description: description,
		Source:      source,
		Category:    "look",
		Tags:        tags,
		Values:      filtered,
		Palette:     filtered.Palette(),
	}
	if dropped > 0 {
		choice.Description += fmt.Sprintf(" · %d settings ignored because they are not appearance settings", dropped)
		choice.Tags = append(choice.Tags, "filtered")
	}
	return choice, nil
}
