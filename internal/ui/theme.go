package ui

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"

	"ghostty.studio/internal/core"
)

// Palette: cool ink canvas, lifted slate, clear edges, warm ivory text, muted
// secondary, pale mint accent, warm sand for staged state. Chrome never changes
// when the specimen palette changes.
var (
	colCanvas = lipgloss.Color("#11151e")
	colSlate  = lipgloss.Color("#1b2230")
	colEdge   = lipgloss.Color("#344052")
	colEdgeLo = lipgloss.Color("#232d3b")
	colText   = lipgloss.Color("#edf0e8")
	colMuted  = lipgloss.Color("#9aa8b9")
	colMint   = lipgloss.Color("#b8d6c1")
	colSand   = lipgloss.Color("#e4c899")
	colErr    = lipgloss.Color("#e0907c")
)

// styles holds every chrome style. The specimen carries its own palette-derived
// styles so chrome stays quiet while the hero changes.
type styles struct {
	App        lipgloss.Style
	Identity   lipgloss.Style
	Diamond    lipgloss.Style
	Descriptor lipgloss.Style
	Version    lipgloss.Style
	Tab        lipgloss.Style
	TabActive  lipgloss.Style
	TabIndex   lipgloss.Style
	TabIndexOn lipgloss.Style
	Rule       lipgloss.Style
	RuleSoft   lipgloss.Style
	Text       lipgloss.Style
	Dim        lipgloss.Style
	Mint       lipgloss.Style
	Sand       lipgloss.Style
	Err        lipgloss.Style
	OK         lipgloss.Style
	Row        lipgloss.Style
	RowSel     lipgloss.Style
	Rail       lipgloss.Style
	RailOn     lipgloss.Style
	Marker     lipgloss.Style
	Sub        lipgloss.Style
	SubSel     lipgloss.Style
	Header     lipgloss.Style
	Star       lipgloss.Style
	StarSel    lipgloss.Style
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
		Diamond:    lipgloss.NewStyle().Foreground(colMint),
		Descriptor: lipgloss.NewStyle().Foreground(colMuted),
		Version:    lipgloss.NewStyle().Foreground(colMuted),
		Tab:        lipgloss.NewStyle().Foreground(colMuted),
		TabActive:  lipgloss.NewStyle().Foreground(colCanvas).Background(colMint).Bold(true),
		TabIndex:   lipgloss.NewStyle().Foreground(colMuted).Background(colSlate),
		TabIndexOn: lipgloss.NewStyle().Foreground(colCanvas).Background(colMint).Bold(true),
		Rule:       lipgloss.NewStyle().Foreground(colEdge),
		RuleSoft:   lipgloss.NewStyle().Foreground(colEdgeLo),
		Text:       lipgloss.NewStyle().Foreground(colText),
		Dim:        lipgloss.NewStyle().Foreground(colMuted),
		Mint:       lipgloss.NewStyle().Foreground(colMint),
		Sand:       lipgloss.NewStyle().Foreground(colSand),
		Err:        lipgloss.NewStyle().Foreground(colErr),
		OK:         lipgloss.NewStyle().Foreground(colMint),
		Row:        lipgloss.NewStyle().Foreground(colText).Background(colCanvas),
		RowSel:     lipgloss.NewStyle().Foreground(colText).Background(colSlate),
		Rail:       lipgloss.NewStyle().Foreground(colCanvas).Background(colCanvas),
		RailOn:     lipgloss.NewStyle().Foreground(colMint).Background(colSlate),
		Marker:     lipgloss.NewStyle().Foreground(colMuted),
		Sub:        lipgloss.NewStyle().Foreground(colMuted).Background(colCanvas),
		SubSel:     lipgloss.NewStyle().Foreground(colMuted).Background(colSlate),
		Header:     lipgloss.NewStyle().Foreground(colMint),
		Star:       lipgloss.NewStyle().Foreground(colSand).Background(colCanvas),
		StarSel:    lipgloss.NewStyle().Foreground(colSand).Background(colSlate),
		Modal:      lipgloss.NewStyle().Background(colSlate).Foreground(colText),
		ModalSel:   lipgloss.NewStyle().Background(colEdge).Foreground(colText).Bold(true),
		ModalTitle: lipgloss.NewStyle().Foreground(colMint).Bold(true),
		Button:     lipgloss.NewStyle().Foreground(colMuted).Background(colSlate).Padding(0, 2),
		ButtonSel:  lipgloss.NewStyle().Foreground(colCanvas).Background(colMint).Bold(true).Padding(0, 2),
		Empty:      lipgloss.NewStyle().Foreground(colMuted).Italic(true),
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
