package core

import (
	"strings"
	"testing"
)

func TestNormalizeHex(t *testing.T) {
	cases := map[string]string{
		"#abc":     "#aabbcc",
		"#AABBCC":  "#aabbcc",
		" #2E3440": "#2e3440",
		"nord":     "",
		"#12345":   "",
		"#gggggg":  "",
		"":         "",
	}
	for input, want := range cases {
		if got := NormalizeHex(input); got != want {
			t.Errorf("NormalizeHex(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestContrastAndLuminance(t *testing.T) {
	if got := ContrastRatio("#000000", "#ffffff"); got < 20.9 || got > 21.1 {
		t.Errorf("black on white contrast = %v, want 21", got)
	}
	if got := ContrastRatio("#ffffff", "#ffffff"); got != 1 {
		t.Errorf("identical colors contrast = %v, want 1", got)
	}
	if got := ContrastRatio("nonsense", "#ffffff"); got != 0 {
		t.Errorf("invalid color contrast = %v, want 0", got)
	}
	if !IsLight("#ffffff") || !IsLight("#eff1f5") {
		t.Error("white and Catppuccin Latte backgrounds must read as light")
	}
	if IsLight("#2e3440") || IsLight("#101010") {
		t.Error("Nord and near-black backgrounds must read as dark")
	}
}

func TestPaletteFromValues(t *testing.T) {
	values := Values{
		"background":           []string{"#101518"},
		"foreground":           []string{"#e5e8df"},
		"cursor-color":         []string{"#dab979"},
		"selection-background": []string{"#a8b9a5"},
		"palette":              []string{"0=#111111", "7=#eeeeee", "not-a-palette-entry", "20=#222222"},
	}
	palette := values.Palette()
	if palette.Background != "#101518" || palette.Foreground != "#e5e8df" {
		t.Fatalf("unexpected palette %+v", palette)
	}
	if palette.Cursor != "#dab979" || palette.Selection != "#a8b9a5" {
		t.Fatalf("unexpected cursor or selection in %+v", palette)
	}
	if palette.ANSI[0] != "#111111" || palette.ANSI[7] != "#eeeeee" {
		t.Fatalf("palette entries not parsed: %+v", palette.ANSI)
	}
	if palette.ANSI[15] != "" {
		t.Fatalf("unset palette entry should stay empty, got %q", palette.ANSI[15])
	}
	// The space separated form Ghostty also accepts.
	if idx, hex, ok := SplitPaletteEntry("3 #aabbcc"); !ok || idx != 3 || hex != "#aabbcc" {
		t.Fatalf("space separated palette entry not parsed: %d %s %v", idx, hex, ok)
	}
}

func TestPatchSemantics(t *testing.T) {
	base := Values{
		"font-size": []string{"12"},
		"palette":   []string{"0=#000000"},
	}
	patch := Values{
		"font-size":   []string{"14"},
		"palette":     []string{"0=#111111", "1=#222222"},
		"background":  []string{"#101010"},
		"unset-me":    nil,
		"font-family": []string{"", "Iosevka"},
	}
	got := Patch(base, patch)
	if got.First("font-size") != "14" {
		t.Errorf("scalar replacement failed: %v", got["font-size"])
	}
	if len(got["palette"]) != 2 || got["palette"][0] != "0=#111111" {
		t.Errorf("list replacement failed: %v", got["palette"])
	}
	if _, ok := got["unset-me"]; ok {
		t.Error("an empty value list must remove the managed key")
	}
	if list := got["font-family"]; len(list) != 2 || list[0] != "" {
		t.Errorf("reset entry must be preserved: %v", list)
	}
	if got.First("background") != "#101010" {
		t.Error("new key must be added")
	}
	// An absent key in the patch leaves the base value alone.
	absent := Patch(base, Values{"font-size": []string{"16"}})
	if absent.First("palette") != "0=#000000" {
		t.Error("absent patch keys must leave base values unchanged")
	}
	// Patch must not alias the patch or the base.
	patch["font-size"][0] = "99"
	if got.First("font-size") != "14" {
		t.Error("Patch must copy values")
	}
}

func TestEqualAndClone(t *testing.T) {
	a := Values{"font-family": []string{"A", "B"}}
	b := a.Clone()
	if !Equal(a, b) {
		t.Fatal("clone must compare equal")
	}
	b["font-family"][0] = "C"
	if Equal(a, b) {
		t.Fatal("mutating a clone must not affect the original")
	}
	if Equal(a, Values{"font-family": []string{"A"}}) {
		t.Fatal("different list lengths must not compare equal")
	}
}

func TestParseEntriesMatchesGhosttyGrammar(t *testing.T) {
	content := []byte(`# a comment
   font-size=14
font-family = JetBrains Mono
font-family = "Iosevka Term"
background = #101518 # not a comment
empty =
	tab-key	=	value
not a directive
`)
	entries := parseEntries(content)
	want := []struct{ key, value string }{
		{"font-size", "14"},
		{"font-family", "JetBrains Mono"},
		{"font-family", "Iosevka Term"},
		{"background", "#101518 # not a comment"},
		{"empty", ""},
		{"tab-key", "value"},
	}
	if len(entries) != len(want) {
		t.Fatalf("parsed %d entries, want %d: %+v", len(entries), len(want), entries)
	}
	for i, expected := range want {
		if entries[i].key != expected.key || entries[i].value != expected.value {
			t.Errorf("entry %d = %q=%q, want %q=%q", i, entries[i].key, entries[i].value, expected.key, expected.value)
		}
	}
}

func TestUnquoteValue(t *testing.T) {
	cases := map[string]string{
		`"Iosevka Term"`: "Iosevka Term",
		`""`:             "",
		`"a\"b"`:         `a"b`,
		`plain`:          "plain",
		`"unterminated`:  `"unterminated`,
		`"back\\slash"`:  `back\slash`,
		`text:\x1b\r`:    `text:\x1b\r`,
	}
	for input, want := range cases {
		if got := unquoteValue(input); got != want {
			t.Errorf("unquoteValue(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestCheckValuesRejectsInjection(t *testing.T) {
	bad := []Values{
		{"font-size": []string{"14\ncommand = rm -rf /"}},
		{"font-size": []string{"14\r\nbackground = #000000"}},
		{"font-size": []string{"14\x00"}},
		{"font size": []string{"14"}},
		{"": []string{"14"}},
		{"font-size": []string{"14"}},
	}
	if err := checkValues(bad[0]); err == nil {
		t.Error("a newline in a value must be rejected")
	}
	if err := checkValues(bad[1]); err == nil {
		t.Error("a CRLF in a value must be rejected")
	}
	if err := checkValues(bad[2]); err == nil {
		t.Error("a NUL byte in a value must be rejected")
	}
	if err := checkValues(bad[3]); err == nil {
		t.Error("a key with a space must be rejected")
	}
	if err := checkValues(bad[4]); err == nil {
		t.Error("an empty key must be rejected")
	}
	if err := checkValues(bad[5]); err != nil {
		t.Errorf("a valid value must be accepted: %v", err)
	}
}

func TestRenderDirective(t *testing.T) {
	cases := []struct{ key, value, want string }{
		{"font-size", "14", "font-size = 14"},
		{"font-family", "", `font-family = ""`},
		{"palette", "0=#3b4252", "palette = 0=#3b4252"},
		{"background", "#101518", "background = #101518"},
		{"config-file", "?/tmp/x", `config-file = "?/tmp/x"`},
	}
	for _, c := range cases {
		if got := renderDirective(c.key, c.value); got != c.want {
			t.Errorf("renderDirective(%q, %q) = %q, want %q", c.key, c.value, got, c.want)
		}
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Catppuccin Mocha":  "catppuccin-mocha",
		"Rose Pine":         "rose-pine",
		"3024 Night":        "3024-night",
		"Gruvbox Material+": "gruvbox-material+",
		"  spaced  out  ":   "spaced-out",
		"Ünïcödé":           "ünïcödé",
	}
	for input, want := range cases {
		if got := slugify(input); got != want {
			t.Errorf("slugify(%q) = %q, want %q", input, got, want)
		}
	}
	if strings.Contains(slugify("a/b"), "/") {
		t.Error("slugs must not contain path separators")
	}
}
