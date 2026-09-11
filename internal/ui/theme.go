package ui

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"

	"ghostty.studio/internal/core"
)

// Brand palette: dark mineral canvas, ivory text, muted sage, amber accent.
// Chrome colors never change when the specimen palette changes.
var (
	colCanvas    = lipgloss.Color("#101518")
	colCanvasAlt = lipgloss.Color("#161e23")
	colText      = lipgloss.Color("#e5e8df")
	colIvoryDim  = lipgloss.Color("#b6bcb1")
	colSage      = lipgloss.Color("#a8b9a5")
	colAmber     = lipgloss.Color("#dab979")
	colLine      = lipgloss.Color("#253136")
	colLineSoft  = lipgloss.Color("#1a2328")
	colSelect    = lipgloss.Color("#1d282e")
	colErr       = lipgloss.Color("#d98b79")
)

// styles holds every chrome style. Built once; the specimen uses its own
// palette-derived styles so chrome stays stable.
type styles struct {
	App        lipgloss.Style
	Identity   lipgloss.Style
	Descriptor lipgloss.Style
	Version    lipgloss.Style
	Tab        lipgloss.Style
	TabActive  lipgloss.Style
	TabIndex   lipgloss.Style
	Rule       lipgloss.Style
	RuleSoft   lipgloss.Style
	Text       lipgloss.Style
	Dim        lipgloss.Style
	Sage       lipgloss.Style
	Amber      lipgloss.Style
	Err        lipgloss.Style
	OK         lipgloss.Style
	Row        lipgloss.Style
	RowSel     lipgloss.Style
	Marker     lipgloss.Style
	MarkerSel  lipgloss.Style
	Header     lipgloss.Style
	Modal      lipgloss.Style
	ModalSel   lipgloss.Style
	ModalTitle lipgloss.Style
	Button     lipgloss.Style
	ButtonSel  lipgloss.Style
	Empty      lipgloss.Style
}

func newStyles() *styles {
	return &styles{
		App:        lipgloss.NewStyle().Background(colCanvas).Foreground(colText),
		Identity:   lipgloss.NewStyle().Foreground(colText).Bold(true),
		Descriptor: lipgloss.NewStyle().Foreground(colSage),
		Version:    lipgloss.NewStyle().Foreground(colIvoryDim),
		Tab:        lipgloss.NewStyle().Foreground(colIvoryDim),
		TabActive:  lipgloss.NewStyle().Foreground(colAmber).Bold(true),
		TabIndex:   lipgloss.NewStyle().Foreground(colSage),
		Rule:       lipgloss.NewStyle().Foreground(colLine),
		RuleSoft:   lipgloss.NewStyle().Foreground(colLineSoft),
		Text:       lipgloss.NewStyle().Foreground(colText),
		Dim:        lipgloss.NewStyle().Foreground(colIvoryDim),
		Sage:       lipgloss.NewStyle().Foreground(colSage),
		Amber:      lipgloss.NewStyle().Foreground(colAmber),
		Err:        lipgloss.NewStyle().Foreground(colErr),
		OK:         lipgloss.NewStyle().Foreground(colSage),
		Row:        lipgloss.NewStyle().Foreground(colText).Background(colCanvas),
		RowSel:     lipgloss.NewStyle().Foreground(colText).Background(colSelect),
		Marker:     lipgloss.NewStyle().Foreground(colLine),
		MarkerSel:  lipgloss.NewStyle().Foreground(colAmber).Background(colSelect),
		Header:     lipgloss.NewStyle().Foreground(colSage),
		Modal:      lipgloss.NewStyle().Background(colCanvasAlt).Foreground(colText),
		ModalSel:   lipgloss.NewStyle().Background(colSelect).Foreground(colAmber).Bold(true),
		ModalTitle: lipgloss.NewStyle().Foreground(colAmber).Bold(true),
		Button:     lipgloss.NewStyle().Foreground(colIvoryDim).Background(colCanvasAlt).Padding(0, 2),
		ButtonSel:  lipgloss.NewStyle().Foreground(colCanvas).Background(colAmber).Bold(true).Padding(0, 2),
		Empty:      lipgloss.NewStyle().Foreground(colIvoryDim).Italic(true),
	}
}

// colorHex renders a color as #rrggbb, or "" when the color carries no
// information (NoColor / zero alpha).
func colorHex(c color.Color) string {
	if c == nil {
		return ""
	}
	r, g, b, a := c.RGBA()
	if a == 0 {
		return ""
	}
	return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}

// parseHexColor converts "#rrggbb" (or "#rgb") into a color, reporting whether
// the input was a usable hex color.
func parseHexColor(s string) (color.Color, bool) {
	hex, ok := core.NormalizeHexOK(trimSpace(s))
	if !ok {
		return nil, false
	}
	return lipgloss.Color(hex), true
}

// Color math is core's: the engine and the studio classify light/dark and
// measure contrast with one implementation.
func relLuminance(c color.Color) float64 { return core.Luminance(colorHex(c)) }

func contrastRatio(a, b color.Color) float64 {
	return core.ContrastRatio(colorHex(a), colorHex(b))
}

func isLightColor(c color.Color) bool { return core.IsLight(colorHex(c)) }
