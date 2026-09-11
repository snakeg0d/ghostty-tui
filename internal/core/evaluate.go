package core

import (
	"fmt"
	"strings"
)

// repeatableKeys lists the Ghostty configuration keys whose repeated
// occurrences accumulate instead of replacing the previous value. Everything
// else is scalar: the last occurrence in load order wins.
var repeatableKeys = map[string]bool{
	"font-family":                true,
	"font-family-bold":           true,
	"font-family-italic":         true,
	"font-family-bold-italic":    true,
	"font-variation":             true,
	"font-variation-bold":        true,
	"font-variation-italic":      true,
	"font-variation-bold-italic": true,
	"font-feature":               true,
	"font-codepoint-map":         true,
	"clipboard-codepoint-map":    true,
	"font-style":                 true,
	"keybind":                    true,
	"key-remap":                  true,
	"palette":                    true,
	"custom-shader":              true,
	"config-file":                true,
	"background-image":           true,
	"gtk-custom-css":             true,
	"input":                      true,
}

// IsRepeatable reports whether a Ghostty key accumulates repeated values.
func IsRepeatable(key string) bool { return repeatableKeys[key] }

// themeValues resolves a theme name to the configuration values it defines.
type themeValues func(name string) (Values, bool)

// safeThemeKeys are the theme directives the studio is willing to expand. A
// theme file can contain any directive, including `command`, so only palette
// and appearance keys are ever applied.
var safeThemeKeys = map[string]bool{
	"background":           true,
	"foreground":           true,
	"cursor-color":         true,
	"cursor-text":          true,
	"selection-background": true,
	"selection-foreground": true,
	"palette":              true,
}

// applyDirective folds one directive into out, honouring repeat semantics.
func applyDirective(out, defaults Values, key, value string) {
	repeatable := repeatableKeys[key]
	switch {
	case value == "" && !repeatable:
		// An empty scalar value clears the user's setting: the default applies.
		if def, ok := defaults[key]; ok {
			cp := make([]string, len(def))
			copy(cp, def)
			out[key] = cp
			return
		}
		out[key] = []string{""}
	case value == "" && repeatable:
		// An empty list value resets the accumulated list.
		out[key] = []string{}
	case repeatable:
		out[key] = append(out[key], value)
	default:
		out[key] = []string{value}
	}
}

// evaluate computes the effective configuration of a resolved include graph.
// It mirrors Ghostty's documented behaviour: a file's own directives apply
// first, its includes are folded in afterwards in order, repeated keys
// accumulate and scalar keys are last-wins.
func evaluate(defaults Values, files []*configFile, themes themeValues) Values {
	out := defaults.Clone()
	if out == nil {
		out = make(Values)
	}
	if spec, present := lastThemeDirective(files); present {
		if spec != "" {
			expandTheme(out, defaults, spec, themes)
		}
		// The directive itself is recorded so the effective configuration still
		// names the theme that supplied the colours.
		applyDirective(out, defaults, "theme", spec)
	}
	for _, file := range files {
		for _, e := range file.entries {
			switch e.key {
			case "config-file", "theme":
				// config-file is structure the graph resolver already followed;
				// theme contributions were applied above at lower precedence.
			default:
				applyDirective(out, defaults, e.key, e.value)
			}
		}
	}
	return out
}

// lastThemeDirective returns the theme directive Ghostty resolves last and
// whether any theme directive is present at all. An empty value cancels the
// theme, exactly as Ghostty treats it.
func lastThemeDirective(files []*configFile) (string, bool) {
	spec := ""
	present := false
	for _, file := range files {
		for _, e := range file.entries {
			if e.key == "theme" {
				spec = e.value
				present = true
			}
		}
	}
	return spec, present
}

// expandTheme applies the appearance keys of a theme directive. The argument
// may be a single theme name or an explicit `light:NAME,dark:NAME` pair; the
// dark variant is expanded because the studio renders on a dark canvas.
func expandTheme(out, defaults Values, spec string, themes themeValues) {
	if themes == nil {
		return
	}
	name := selectThemeVariant(spec)
	if name == "" {
		return
	}
	values, ok := themes(name)
	if !ok {
		return
	}
	for _, key := range values.Keys() {
		if !safeThemeKeys[key] {
			// A theme file may contain arbitrary directives. Only appearance
			// keys are ever expanded, so a theme can never smuggle a command
			// into the studio's staged changes.
			continue
		}
		for _, value := range values[key] {
			applyDirective(out, defaults, key, value)
		}
	}
}

// selectThemeVariant picks the variant Ghostty would use for this machine
// appearance preference. Explicit dark wins, then a lone variant.
func selectThemeVariant(spec string) string {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return ""
	}
	var plain, light, dark string
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if idx := strings.IndexByte(part, ':'); idx > 0 {
			kind := strings.ToLower(strings.TrimSpace(part[:idx]))
			value := strings.TrimSpace(part[idx+1:])
			switch kind {
			case "dark":
				dark = value
			case "light":
				light = value
			default:
				plain = value
			}
			continue
		}
		plain = part
	}
	switch {
	case dark != "":
		return dark
	case plain != "":
		return plain
	default:
		return light
	}
}

// overlayChanges merges Ghostty's `--changes-only` output onto its `--default`
// output using the same repeat semantics as the graph evaluator.
func overlayChanges(defaults, changes Values) Values {
	out := defaults.Clone()
	if out == nil {
		out = make(Values)
	}
	for _, key := range changes.Keys() {
		for _, value := range changes[key] {
			applyDirective(out, defaults, key, value)
		}
	}
	return out
}

// diffValues renders the difference between two effective value sets for the
// keys touched by a patch.
func diffValues(before, after Values, keys []string) string {
	var b strings.Builder
	for _, key := range keys {
		bv := before[key]
		av := after[key]
		if equalStrings(bv, av) {
			continue
		}
		fmt.Fprintf(&b, "%s\n", key)
		for _, v := range removedValues(bv, av) {
			fmt.Fprintf(&b, "  - %s\n", displayValue(v))
		}
		for _, v := range addedValues(bv, av) {
			fmt.Fprintf(&b, "  + %s\n", displayValue(v))
		}
	}
	if b.Len() == 0 {
		return "No effective change: the staged values already match the current configuration.\n"
	}
	return b.String()
}

// removedValues lists the values present before but not after.
func removedValues(before, after []string) []string {
	remaining := countValues(after)
	var out []string
	for _, v := range before {
		if remaining[v] > 0 {
			remaining[v]--
			continue
		}
		out = append(out, v)
	}
	return out
}

// addedValues lists the values present after but not before.
func addedValues(before, after []string) []string {
	remaining := countValues(before)
	var out []string
	for _, v := range after {
		if remaining[v] > 0 {
			remaining[v]--
			continue
		}
		out = append(out, v)
	}
	return out
}

func countValues(values []string) map[string]int {
	out := make(map[string]int, len(values))
	for _, v := range values {
		out[v]++
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// displayValue renders an empty value readably in diffs.
func displayValue(value string) string {
	if value == "" {
		return `"" (reset)`
	}
	return value
}
