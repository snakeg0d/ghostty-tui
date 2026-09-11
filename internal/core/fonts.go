package core

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// installedFont is one family reported by the installed Ghostty build.
type installedFont struct {
	name   string
	styles []string
}

// fontPatch builds the staged change for choosing a font family.
//
// The regular chain is reset and rebuilt with the chosen family first, keeping
// every existing fallback so non-Latin coverage is not silently lost. The
// bold, italic and bold-italic chains are reset to empty so they follow the
// selected family through Ghostty's own stylistic search instead of pointing at
// a stale family.
func fontPatch(family string, current Values) Values {
	fallbacks := fontFallbacks(current, family)
	chain := append([]string{""}, family)
	chain = append(chain, fallbacks...)
	patch := Values{
		"font-family":             chain,
		"font-family-bold":        []string{""},
		"font-family-italic":      []string{""},
		"font-family-bold-italic": []string{""},
	}
	return patch
}

// fontFallbacks returns the currently configured families except the one being
// selected, preserving their order.
func fontFallbacks(current Values, family string) []string {
	var out []string
	seen := map[string]bool{strings.ToLower(family): true}
	for _, name := range current["font-family"] {
		name = strings.TrimSpace(name)
		if name == "" || seen[strings.ToLower(name)] {
			continue
		}
		seen[strings.ToLower(name)] = true
		out = append(out, name)
	}
	return out
}

// fontChoices converts installed families into catalog choices.
func fontChoices(fonts []installedFont, current Values) []Choice {
	choices := make([]Choice, 0, len(fonts))
	for _, font := range fonts {
		styleCount := len(font.styles)
		description := fmt.Sprintf("Installed · %d %s available", styleCount, plural(styleCount, "style", "styles"))
		if styleCount == 0 {
			description = "Installed · styles not reported by +list-fonts"
		}
		choices = append(choices, Choice{
			ID:          "font:" + slugify(font.name),
			Name:        font.name,
			Description: description,
			Source:      "Installed font",
			Category:    "font",
			Tags:        fontTags(font),
			Values:      fontPatch(font.name, current),
			Installed:   true,
		})
	}
	return choices
}

func fontTags(font installedFont) []string {
	tags := []string{"installed"}
	lower := strings.ToLower(font.name)
	if strings.Contains(lower, "nerd font") {
		tags = append(tags, "nerd-font")
	}
	if strings.Contains(lower, "mono") {
		tags = append(tags, "monospace")
	}
	return tags
}

// discoverFont is one curated Homebrew font cask offered for installation.
type discoverFont struct {
	name   string
	cask   string
	prefix string
	// probes lists the native family names a targeted
	// `ghostty +list-fonts --family=NAME` probe must try. Two real conditions
	// make this necessary: the unfiltered listing omits installed families
	// (Commit Mono was installed by hand and never appeared), and one bundle can
	// register several families, so a bare "Monaspace" probe finds nothing.
	probes  []string
	url     string
	license string
	// note explains a deliberate omission from the install shelf.
	note string
}

// discoverFonts is the curated discover shelf. Every cask identifier was
// verified against the Homebrew API and every probe name was verified with
// `ghostty +list-fonts --family=NAME` on the installed build.
var discoverFonts = []discoverFont{
	{
		name: "Monaspace", cask: "font-monaspace", prefix: "Monaspace",
		probes: []string{
			"Monaspace Argon",
			"Monaspace Neon",
			"Monaspace Krypton",
			"Monaspace Radon",
			"Monaspace Xenon",
		},
		url:     "https://formulae.brew.sh/cask/font-monaspace",
		license: "SIL OFL 1.1",
	},
	{
		name: "Maple Mono", cask: "font-maple-mono", prefix: "Maple Mono",
		probes:  []string{"Maple Mono"},
		url:     "https://formulae.brew.sh/cask/font-maple-mono",
		license: "SIL OFL 1.1",
	},
	{
		name: "Commit Mono", cask: "font-commit-mono", prefix: "Commit Mono",
		probes:  []string{"CommitMono"},
		url:     "https://formulae.brew.sh/cask/font-commit-mono",
		license: "SIL OFL 1.1",
	},
	{
		name: "Iosevka", cask: "font-iosevka", prefix: "Iosevka",
		probes:  []string{"Iosevka"},
		url:     "https://formulae.brew.sh/cask/font-iosevka",
		license: "SIL OFL 1.1",
	},
	{
		name: "JetBrains Mono", cask: "font-jetbrains-mono", prefix: "JetBrains Mono",
		probes:  []string{"JetBrains Mono"},
		url:     "https://formulae.brew.sh/cask/font-jetbrains-mono",
		license: "SIL OFL 1.1",
	},
	{
		name: "Berkeley Mono", cask: "",
		probes:  []string{"Berkeley Mono"},
		url:     "https://berkeleygraphics.com/typefaces/berkeley-mono/",
		license: "Commercial license required",
		note:    "Commercial typeface: Ghostty Studio will not install it. Purchase and install it yourself.",
	},
}

// discoverChoices builds the install shelf, marking fonts that are already
// present on this machine.
func discoverChoices(fonts []installedFont) []Choice {
	choices := make([]Choice, 0, len(discoverFonts))
	for _, font := range discoverFonts {
		installed := fontInstalled(font, fonts)
		description := fmt.Sprintf("%s · %s · installs to ~/Library/Fonts", font.name, font.license)
		source := "Homebrew cask"
		if font.cask == "" {
			description = font.note
			source = "Commercial typeface"
		}
		if installed {
			description = "Installed locally · " + font.license
		}
		id := font.cask
		if id == "" {
			id = slugify(font.name)
		}
		choices = append(choices, Choice{
			ID:          "discover:" + id,
			Name:        font.name,
			Description: description,
			Source:      source,
			Category:    "font",
			Tags:        discoverTags(font),
			Installed:   installed,
			URL:         font.url,
			License:     font.license,
		})
	}
	return choices
}

func discoverTags(font discoverFont) []string {
	if font.cask == "" {
		return []string{"commercial", "no-install"}
	}
	return []string{"cask", "installable"}
}

// normalizeFamily folds a family name for comparison. Vendors and Homebrew
// spell the same family differently (+list-fonts reports "CommitMono" where the
// cask says "Commit Mono"), so spacing, case and punctuation are removed.
func normalizeFamily(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	return b.String()
}

// familyMatches reports whether the installed families contain want. A prefix
// match covers bundles such as Monaspace that register several families.
func familyMatches(fonts []installedFont, want string) bool {
	normalized := normalizeFamily(want)
	if normalized == "" {
		return false
	}
	for _, font := range fonts {
		installed := normalizeFamily(font.name)
		if installed == normalized || strings.HasPrefix(installed, normalized) {
			return true
		}
	}
	return false
}

// familyMatchesAny reports whether any of the names is installed.
func familyMatchesAny(fonts []installedFont, names []string) bool {
	for _, name := range names {
		if familyMatches(fonts, name) {
			return true
		}
	}
	return false
}

// fontInstalled reports whether any installed family matches the cask family.
func fontInstalled(font discoverFont, fonts []installedFont) bool {
	return familyMatches(fonts, font.prefix) ||
		familyMatches(fonts, font.name) ||
		familyMatchesAny(fonts, font.probes)
}

// fontProbeNames lists the families worth a targeted native probe: the curated
// shelf names and every family the user's configuration asks for. Names that the
// discovered list already reports are dropped by the caller.
func fontProbeNames(fonts []installedFont, current Values) []string {
	var names []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || familyMatches(fonts, name) {
			return
		}
		for _, existing := range names {
			if normalizeFamily(existing) == normalizeFamily(name) {
				return
			}
		}
		names = append(names, name)
	}
	for _, font := range discoverFonts {
		for _, probe := range font.probes {
			add(probe)
		}
	}
	for _, key := range []string{"font-family", "font-family-bold", "font-family-italic", "font-family-bold-italic"} {
		for _, name := range current[key] {
			add(name)
		}
	}
	return names
}

// findDiscover resolves a discover choice by catalog ID or cask name.
func findDiscover(id string) (discoverFont, bool) {
	trimmed := strings.TrimPrefix(id, "discover:")
	for _, font := range discoverFonts {
		if trimmed == font.cask || strings.EqualFold(trimmed, font.name) || trimmed == slugify(font.name) {
			return font, true
		}
	}
	return discoverFont{}, false
}

// fontAvailable reports whether the discovered font list already contains the
// curated family.
func (e *Engine) fontAvailable(font discoverFont) bool {
	return familyMatches(e.fonts, font.prefix) ||
		familyMatches(e.fonts, font.name) ||
		familyMatchesAny(e.fonts, font.probes)
}

// familyAvailable reports whether Ghostty can already use the curated family. It
// checks the discovered families first and then asks the native build directly,
// because the unfiltered listing does not report every installed family.
func (e *Engine) familyAvailable(ctx context.Context, font discoverFont) bool {
	if e.fontAvailable(font) {
		return true
	}
	if len(font.probes) == 0 || !e.native.available() {
		return false
	}
	found, _ := e.native.probeFamilies(ctx, font.probes)
	return len(found) > 0
}

// InstallFont installs one curated font cask with Homebrew, or reports that the
// family is already available.
//
// The work runs in three phases so the engine never holds its state lock across
// an external process: a cask install can take minutes, and every State and
// Catalog call would otherwise block behind it (and a nested Refresh would
// deadlock outright).
func (e *Engine) InstallFont(id string) (Result, error) {
	ctx := context.Background()
	font, err := e.prepareInstall(id)
	if err != nil {
		return Result{}, err
	}
	if font.cask == "" {
		return Result{}, fmt.Errorf("%s is not installable by Ghostty Studio: %s (%s)", font.name, font.note, font.url)
	}
	if e.familyAvailableLocked(ctx, font) {
		return Result{
			Message: fmt.Sprintf("%s is already available to Ghostty, so nothing was installed and no font files were changed. It can be selected from the font list.", font.name),
		}, nil
	}
	if e.brewPath() == "" {
		return Result{}, errors.New("homebrew was not found, so Ghostty Studio cannot install fonts; install Homebrew or add the font manually")
	}

	// Phase two runs without the engine lock.
	stdout, stderr, runErr := e.runBrewInstall(ctx, font)

	// Phase three re-reads and verifies under the lock again.
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.finishInstall(ctx, font, stdout, stderr, runErr)
}

// prepareInstall resolves the requested item and rejects unknown identifiers
// before any work is attempted.
func (e *Engine) prepareInstall(id string) (discoverFont, error) {
	font, ok := findDiscover(id)
	if !ok {
		return discoverFont{}, fmt.Errorf("unknown discover item %q", id)
	}
	return font, nil
}

// familyAvailableLocked is familyAvailable guarded by the engine lock.
func (e *Engine) familyAvailableLocked(ctx context.Context, font discoverFont) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.familyAvailable(ctx, font)
}

// brewPath reports the Homebrew executable under the engine lock.
func (e *Engine) brewPath() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.brew
}

// runBrewInstall runs `brew install --cask` with an argument array, never a
// shell, and never removes or overwrites an existing font.
func (e *Engine) runBrewInstall(ctx context.Context, font discoverFont) (stdout, stderr []byte, err error) {
	ctx, cancel := context.WithTimeout(ctx, installTimeout)
	defer cancel()
	return e.runner.Run(ctx, Command{
		Path: e.brewPath(),
		Args: []string{"install", "--cask", font.cask},
		Env:  []string{"HOMEBREW_NO_ENV_HINTS=1"},
	})
}

// finishInstall refreshes the font list and reports what is actually available.
func (e *Engine) finishInstall(ctx context.Context, font discoverFont, stdout, stderr []byte, runErr error) (Result, error) {
	if runErr != nil {
		detail := strings.TrimSpace(string(stderr))
		if detail == "" {
			detail = strings.TrimSpace(string(stdout))
		}
		return Result{}, fmt.Errorf("brew install --cask %s failed (exit %d): %s", font.cask, exitCode(runErr), tail(detail))
	}
	result := Result{}
	// InstallFont already holds the engine mutex, so the locked refresh is used.
	if refreshErr := e.refresh(ctx); refreshErr != nil {
		result.Warnings = append(result.Warnings, "the font list could not be re-read after the install: "+refreshErr.Error())
	}
	if e.fontAvailable(font) || e.familyProbed(ctx, font) {
		result.Message = fmt.Sprintf("Installed %s with Homebrew, and Ghostty now reports the family, so it can be selected. Nothing was written to your Ghostty configuration.", font.name)
		return result, nil
	}
	result.Message = fmt.Sprintf("Installed %s with Homebrew, but Ghostty does not report the family yet, so Ghostty Studio cannot claim it is selectable. Restart Ghostty and refresh.", font.name)
	result.Warnings = append(result.Warnings, fmt.Sprintf("the family installed by %s is not reported by Ghostty yet", font.cask))
	return result, nil
}

// familyProbed asks the native build directly, which covers fonts a
// single-family probe can see even when the unfiltered listing omits them.
func (e *Engine) familyProbed(ctx context.Context, font discoverFont) bool {
	if len(font.probes) == 0 || !e.native.available() {
		return false
	}
	found, _ := e.native.probeFamilies(ctx, font.probes)
	return len(found) > 0
}

// tail keeps the last lines of long command output.
func tail(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > 12 {
		lines = lines[len(lines)-12:]
	}
	return strings.Join(lines, "\n")
}

// sortInstalledFonts orders families alphabetically, case-insensitively.
func sortInstalledFonts(fonts []installedFont) {
	sort.SliceStable(fonts, func(i, j int) bool {
		return strings.ToLower(fonts[i].name) < strings.ToLower(fonts[j].name)
	})
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
