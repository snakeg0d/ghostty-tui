package ui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"ghostty.studio/internal/core"
)

// fallbackSpecimen fills gaps only when a theme leaves a color undefined, so
// the specimen always renders a complete terminal.
var fallbackSpecimen = core.Palette{
	Background: "#101518",
	Foreground: "#e5e8df",
	Cursor:     "#dab979",
	Selection:  "#1d282e",
	ANSI: [16]string{
		"#101518", "#d98b79", "#a8b9a5", "#dab979",
		"#8fa8b8", "#b39dc4", "#8fb8b0", "#c8ccc2",
		"#39464b", "#e0a191", "#bccdb8", "#e8cf9c",
		"#a6bcc9", "#c8b6d3", "#a9cbc3", "#e5e8df",
	},
}

// paletteFromValues derives the specimen palette from effective values using
// core's semantics: native "index=#hex" repeats and plain hex both work.
func paletteFromValues(v core.Values) core.Palette {
	return core.PaletteFor(v)
}

// choicePalette prefers the palette carried by the choice, falling back to the
// Values patch it would stage.
func choicePalette(c core.Choice) core.Palette {
	if c.Palette.Background != "" {
		return c.Palette
	}
	if len(c.Values) > 0 {
		return core.PaletteFor(c.Values)
	}
	return c.Palette
}

// themeIsLight reports whether a theme reads as a light appearance. This is the
// same classification core uses, so filters and health agree.
func themeIsLight(c core.Choice) bool {
	if bg := choicePalette(c).Background; bg != "" {
		return core.IsLight(bg)
	}
	switch strings.ToLower(strings.TrimSpace(c.Category)) {
	case "light", "day":
		return true
	default:
		return false
	}
}

// palettePair returns usable background and foreground colors.
func palettePair(p core.Palette) (color.Color, color.Color) {
	return validColorOr(p.Background, fallbackSpecimen.Background),
		validColorOr(p.Foreground, fallbackSpecimen.Foreground)
}

func validColorOr(hex, fallback string) color.Color {
	if c, ok := parseHexColor(hex); ok {
		return c
	}
	c, _ := parseHexColor(fallback)
	return c
}

// paletteANSI returns 16 usable ANSI colors for the specimen swatch row.
func paletteANSI(p core.Palette) []color.Color {
	out := make([]color.Color, 0, len(p.ANSI))
	for i, hex := range p.ANSI {
		out = append(out, validColorOr(hex, fallbackSpecimen.ANSI[i]))
	}
	return out
}

// specimenSemantics picks legible sample colors from the palette.
func specimenSemantics(p core.Palette) (comment, str, keyword, plain, red, green, cyan, yellow color.Color) {
	ansi := paletteANSI(p)
	bg, _ := palettePair(p)
	pick := func(i int) color.Color {
		c := ansi[i]
		if core.ContrastRatio(colorHex(c), colorHex(bg)) < 2.2 {
			c = lightenOrDarken(c, bg)
		}
		return c
	}
	return pick(8), pick(2), pick(4), pick(7), pick(1), pick(2), pick(6), pick(3)
}

func lightenOrDarken(c, bg color.Color) color.Color {
	if isLightColor(bg) {
		return lipgloss.Darken(c, 0.35)
	}
	return lipgloss.Lighten(c, 0.35)
}

// contrastLabel formats the foreground/background contrast ratio.
func contrastLabel(p core.Palette) string {
	return core.FormatContrast(core.ContrastRatio(p.Foreground, p.Background))
}
