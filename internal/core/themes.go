package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// curatedThemeOrder is the editorial ordering of well-known themes. Only themes
// that exist on this machine appear; everything else follows alphabetically.
var curatedThemeOrder = []string{
	"nord",
	"catppuccin-mocha",
	"tokyonight",
	"rose-pine",
	"kanagawa",
	"flexoki-dark",
	"everforest-dark",
	"gruvbox-dark",
	"github-light",
}

// themeEntry is one theme discovered on disk.
type themeEntry struct {
	name     string
	slug     string
	path     string
	source   string
	values   Values
	palette  Palette
	light    bool
	contrast float64
	curated  bool
	ok       bool
}

// loadThemes reads every theme file from the given directories. Later
// directories provide fallbacks for duplicate names, matching Ghostty's own
// precedence where user themes are found before bundled ones.
func loadThemes(dirs []themeDir) ([]themeEntry, []string) {
	var warnings []string
	var entries []themeEntry
	seen := make(map[string]bool)
	for _, dir := range dirs {
		names, err := os.ReadDir(dir.path)
		if err != nil {
			continue
		}
		for _, item := range names {
			if item.IsDir() {
				continue
			}
			info, err := item.Info()
			if err != nil || !info.Mode().IsRegular() {
				continue
			}
			path := filepath.Join(dir.path, item.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("theme %s could not be read: %v", path, err))
				continue
			}
			values := Values{}
			for _, e := range parseEntries(content) {
				if !safeThemeKeys[e.key] {
					continue
				}
				values[e.key] = append(values[e.key], e.value)
			}
			if len(values) == 0 {
				// Not a theme file (a README, for example).
				continue
			}
			key := strings.ToLower(item.Name())
			if seen[key] {
				continue
			}
			seen[key] = true
			palette := values.Palette()
			contrast := ContrastRatio(palette.Foreground, palette.Background)
			entries = append(entries, themeEntry{
				name:     item.Name(),
				slug:     slugify(item.Name()),
				path:     path,
				source:   dir.source,
				values:   values,
				palette:  palette,
				light:    IsLight(palette.Background),
				contrast: contrast,
				ok:       true,
			})
		}
	}
	if len(entries) == 0 {
		warnings = append(warnings, "no theme files were found in the user or Ghostty theme directories")
	}
	return entries, warnings
}

// sortThemes applies the curated ordering, then alphabetical order.
func sortThemes(entries []themeEntry) {
	rank := make(map[string]int, len(curatedThemeOrder))
	for i, name := range curatedThemeOrder {
		rank[name] = i
	}
	// Resolve curated names to real entries: exact match first, then prefix.
	for i := range entries {
		if _, ok := rank[entries[i].slug]; ok {
			entries[i].curated = true
			continue
		}
		for _, want := range curatedThemeOrder {
			if strings.HasPrefix(entries[i].slug, want) {
				rank[entries[i].slug] = rank[want]
				entries[i].curated = true
				break
			}
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		ri, iok := rank[entries[i].slug]
		rj, jok := rank[entries[j].slug]
		switch {
		case iok && jok:
			if ri != rj {
				return ri < rj
			}
		case iok != jok:
			return iok
		}
		li, lj := strings.ToLower(entries[i].name), strings.ToLower(entries[j].name)
		if li != lj {
			return li < lj
		}
		return entries[i].name < entries[j].name
	})
}

// themeChoices converts theme entries into catalog choices.
func themeChoices(entries []themeEntry) []Choice {
	choices := make([]Choice, 0, len(entries))
	ambiguous := ambiguousSlugs(entries)
	used := make(map[string]bool, len(entries))
	for _, entry := range entries {
		id := entry.slug
		if ambiguous[id] {
			// Two files can share a slug when their names differ only by
			// punctuation, so the source alone is not enough to separate them.
			id = slugify(entry.source) + "/" + id
		}
		if used[id] {
			base := id
			for n := 2; used[id]; n++ {
				id = fmt.Sprintf("%s-%d", base, n)
			}
		}
		used[id] = true
		choices = append(choices, Choice{
			ID:          "theme:" + id,
			Name:        entry.name,
			Description: themeDescription(entry),
			Source:      entry.source,
			Category:    themeCategory(entry),
			Tags:        themeTags(entry),
			Palette:     entry.palette,
			Values:      themePatch(entry),
		})
	}
	return choices
}

// ambiguousSlugs reports slugs shared by more than one theme.
func ambiguousSlugs(entries []themeEntry) map[string]bool {
	counts := make(map[string]int, len(entries))
	for _, entry := range entries {
		counts[entry.slug]++
	}
	out := make(map[string]bool)
	for slug, n := range counts {
		if n > 1 {
			out[slug] = true
		}
	}
	return out
}

func themeCategory(entry themeEntry) string {
	switch {
	case entry.palette.Background == "":
		return "unknown"
	case entry.light:
		return "light"
	default:
		return "dark"
	}
}

func themeDescription(entry themeEntry) string {
	parts := []string{strings.ToUpper(themeCategory(entry)[:1]) + themeCategory(entry)[1:]}
	if entry.contrast > 0 {
		parts = append(parts, "contrast "+FormatContrast(entry.contrast))
	}
	parts = append(parts, entry.source)
	return strings.Join(parts, " · ")
}

func themeTags(entry themeEntry) []string {
	tags := []string{themeCategory(entry)}
	if entry.curated {
		tags = append(tags, "curated")
	}
	if entry.source != "Ghostty bundle" {
		tags = append(tags, "user")
	}
	if entry.contrast > 0 && entry.contrast < 4.5 {
		tags = append(tags, "low-contrast")
	}
	return tags
}

// themePatch builds the staged change for a theme: an explicit, complete palette
// rather than a theme name, so a personal colour override elsewhere in the
// configuration cannot mask the specimen. The theme directive itself is reset
// so auto light/dark switching cannot mix two palettes, and appearance keys the
// theme leaves out are explicitly reset so a stale value cannot survive.
func themePatch(entry themeEntry) Values {
	values := Values{"theme": []string{""}}
	for _, key := range themeAppearanceKeys {
		if vals, ok := entry.values.Get(key); ok && len(vals) > 0 {
			values[key] = vals
			continue
		}
		values[key] = []string{""}
	}
	var palette []string
	palette = append(palette, "")
	for i := range 16 {
		if hex := entry.palette.ANSI[i]; hex != "" {
			palette = append(palette, fmt.Sprintf("%d=%s", i, hex))
		}
	}
	values["palette"] = palette
	return values
}

// themeIndex provides name based lookup for theme directives in the user's own
// configuration file.
func themeIndex(entries []themeEntry) themeValues {
	byName := make(map[string]Values, len(entries))
	byLower := make(map[string]Values, len(entries))
	for _, entry := range entries {
		if len(entry.values) == 0 {
			continue
		}
		byName[entry.name] = entry.values
		if _, ok := byLower[strings.ToLower(entry.name)]; !ok {
			byLower[strings.ToLower(entry.name)] = entry.values
		}
	}
	return func(name string) (Values, bool) {
		name = strings.TrimSpace(name)
		if values, ok := byName[name]; ok {
			return values, true
		}
		if values, ok := byLower[strings.ToLower(name)]; ok {
			return values, true
		}
		return nil, false
	}
}

// slugify renders a display name as a stable identifier fragment.
func slugify(name string) string {
	var b strings.Builder
	lastDash := true

	for _, r := range strings.ToLower(name) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		case r == '+' || r == '.':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// themeAppearanceKeys are the color keys a theme may define. A theme patch
// always writes all of them: the ones the theme defines explicitly, and an
// explicit reset for the ones it leaves out, so a stale value from elsewhere in
// the configuration cannot leak into the specimen or the applied result.
var themeAppearanceKeys = []string{
	"background",
	"foreground",
	"cursor-color",
	"cursor-text",
	"selection-background",
	"selection-foreground",
}
