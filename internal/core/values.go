package core

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Values is an ordered, repeatable configuration map. It models Ghostty's own
// configuration semantics: a key may appear several times (font-family,
// keybind, palette, custom-shader) and the order of those repeats matters.
//
// Patch semantics, used by Change.Values and by every staged Choice:
//
//   - the key is absent from the map: the managed value is left unchanged
//   - the key maps to an empty list: the managed value is unset, so the user's
//     own configuration and the Ghostty defaults apply again
//   - an empty string inside a non-empty list: the accumulated list is reset at
//     that position, exactly like `key = ""` in a Ghostty configuration file
//
// Values never contains newlines; configuration writes reject any value that
// would inject a new directive.
type Values map[string][]string

// Clone returns a deep copy.
func (v Values) Clone() Values {
	if v == nil {
		return nil
	}
	out := make(Values, len(v))
	for k, vals := range v {
		cp := make([]string, len(vals))
		copy(cp, vals)
		out[k] = cp
	}
	return out
}

// Keys returns the keys in deterministic order.
func (v Values) Keys() []string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Get returns a copy of the values for key, and whether the key is present.
func (v Values) Get(key string) ([]string, bool) {
	vals, ok := v[key]
	if !ok {
		return nil, false
	}
	cp := make([]string, len(vals))
	copy(cp, vals)
	return cp, true
}

// First returns the first value for key, or "" when absent or empty.
func (v Values) First(key string) string {
	if vals := v[key]; len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// Last returns the last value for key, or "" when absent or empty.
func (v Values) Last(key string) string {
	if vals := v[key]; len(vals) > 0 {
		return vals[len(vals)-1]
	}
	return ""
}

// Set replaces the values for key with a copy of vals.
func (v Values) Set(key string, vals ...string) {
	cp := make([]string, len(vals))
	copy(cp, vals)
	v[key] = cp
}

// Patch merges patch into base and returns a new Values. It is the operation
// that turns a staged Change into the next managed value set.
func Patch(base, patch Values) Values {
	out := base.Clone()
	if out == nil {
		out = make(Values, len(patch))
	}
	for _, key := range patch.Keys() {
		vals := patch[key]
		if len(vals) == 0 {
			delete(out, key)
			continue
		}
		cp := make([]string, len(vals))
		copy(cp, vals)
		out[key] = cp
	}
	return out
}

// Equal reports whether two value sets are identical, ignoring key order.
func Equal(a, b Values) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if av[i] != bv[i] {
				return false
			}
		}
	}
	return true
}

// PaletteFor extracts the specimen palette from effective values. Palette
// entries accept both the native `index=#rrggbb` form and a plain hex color.
func PaletteFor(values Values) Palette {
	return values.Palette()
}

// Palette extracts the specimen palette from these values.
func (v Values) Palette() Palette {
	var p Palette
	p.Background = NormalizeHex(v.Last("background"))
	p.Foreground = NormalizeHex(v.Last("foreground"))
	p.Cursor = NormalizeHex(v.Last("cursor-color"))
	p.Selection = NormalizeHex(v.Last("selection-background"))
	for _, entry := range v["palette"] {
		idx, hex, ok := SplitPaletteEntry(entry)
		if !ok || idx < 0 || idx > 15 {
			continue
		}
		p.ANSI[idx] = hex
	}
	return p
}

// SplitPaletteEntry parses one palette repeat: "7=#e5e9f0" or "7 #e5e9f0".
func SplitPaletteEntry(entry string) (int, string, bool) {
	trimmed := strings.TrimSpace(entry)
	idxPart, hexPart := "", ""
	if i := strings.IndexByte(trimmed, '='); i >= 0 {
		idxPart, hexPart = trimmed[:i], trimmed[i+1:]
	} else if fields := strings.Fields(trimmed); len(fields) == 2 {
		idxPart, hexPart = fields[0], fields[1]
	} else {
		return 0, "", false
	}
	idx, err := strconv.Atoi(strings.TrimSpace(idxPart))
	if err != nil {
		return 0, "", false
	}
	hex, ok := NormalizeHexOK(hexPart)
	if !ok {
		return 0, "", false
	}
	return idx, hex, true
}

// NormalizeHex lower-cases and expands a hex color, returning "" for anything
// that is not a color. #abc becomes #aabbcc.
func NormalizeHex(s string) string {
	hex, _ := NormalizeHexOK(s)
	return hex
}

// NormalizeHexOK is NormalizeHex with an explicit validity result.
func NormalizeHexOK(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "#") {
		return "", false
	}
	body := strings.ToLower(s[1:])
	switch len(body) {
	case 3:
		for _, r := range body {
			if !isHexDigit(r) {
				return "", false
			}
		}
		return "#" + string([]byte{body[0], body[0], body[1], body[1], body[2], body[2]}), true
	case 6:
		for _, r := range body {
			if !isHexDigit(r) {
				return "", false
			}
		}
		return "#" + body, true
	default:
		return "", false
	}
}

func isHexDigit(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

// Luminance returns the WCAG relative luminance of a hex color. Invalid colors
// return 0.
func Luminance(hex string) float64 {
	r, g, b, ok := rgb(hex)
	if !ok {
		return 0
	}
	return 0.2126*linear(r) + 0.7152*linear(g) + 0.0722*linear(b)
}

// ContrastRatio returns the WCAG contrast ratio between two hex colors, from
// 1.0 (identical) to 21.0 (black on white). Invalid colors yield 0.
func ContrastRatio(a, b string) float64 {
	la, lb := Luminance(a), Luminance(b)
	_, _, _, okA := rgb(a)
	_, _, _, okB := rgb(b)
	if !okA || !okB {
		return 0
	}
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// IsLight reports whether a background color reads as light. It compares the
// background against pure black and pure white: the closer endpoint wins.
func IsLight(hex string) bool {
	if _, _, _, ok := rgb(hex); !ok {
		return false
	}
	return ContrastRatio(hex, "#ffffff") <= ContrastRatio(hex, "#000000")
}

func rgb(hex string) (float64, float64, float64, bool) {
	norm, ok := NormalizeHexOK(hex)
	if !ok {
		return 0, 0, 0, false
	}
	var v [3]float64
	for i := range 3 {
		n, err := strconv.ParseUint(norm[1+i*2:3+i*2], 16, 8)
		if err != nil {
			return 0, 0, 0, false
		}
		v[i] = float64(n) / 255.0
	}
	return v[0], v[1], v[2], true
}

func linear(c float64) float64 {
	if c <= 0.03928 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

// FormatContrast renders a contrast ratio for display.
func FormatContrast(ratio float64) string {
	if ratio <= 0 {
		return "n/a"
	}
	return strconv.FormatFloat(math.Round(ratio*10)/10, 'f', 1, 64)
}

// containsUnsafe reports whether a value would break out of its line when
// written to a configuration file.
func containsUnsafe(value string) bool {
	return strings.ContainsAny(value, "\r\n\x00")
}

// checkValues rejects values that cannot be written safely.
func checkValues(values Values) error {
	for _, key := range values.Keys() {
		if err := checkKey(key); err != nil {
			return err
		}
		for _, v := range values[key] {
			if containsUnsafe(v) {
				return fmt.Errorf("value for %q contains a line break or NUL byte", key)
			}
		}
	}
	return nil
}

// checkKey rejects keys that are not plain Ghostty configuration keys.
func checkKey(key string) error {
	if key == "" {
		return fmt.Errorf("empty configuration key")
	}
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_':
		default:
			return fmt.Errorf("invalid configuration key %q", key)
		}
	}
	if strings.HasPrefix(key, "-") {
		return fmt.Errorf("invalid configuration key %q", key)
	}
	return nil
}

// PaletteEqual reports whether two palettes carry exactly the same colors.
//
// It is the reliable way to identify the theme that is currently active: a
// staged theme writes explicit colours and resets the theme directive, so the
// name is no longer in the configuration, while the colours still identify it.
func PaletteEqual(a, b Palette) bool {
	if a.Background != b.Background || a.Foreground != b.Foreground {
		return false
	}
	if a.Cursor != b.Cursor || a.Selection != b.Selection {
		return false
	}
	return a.ANSI == b.ANSI
}
