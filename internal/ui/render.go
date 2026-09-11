package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func trimSpace(s string) string { return strings.TrimSpace(s) }

// box renders s as exactly w cells of one line. The text is ansi-aware clipped
// before styling: Width alone would wrap an over-long string into two lines.
func box(st lipgloss.Style, s string, w int) string {
	if w <= 0 {
		return ""
	}
	if ansi.StringWidth(s) > w {
		s = ansi.Truncate(s, w, "")
	}
	return st.Width(w).MaxWidth(w).Render(s)
}

// fitLine clips and pads a plain or styled line to exactly w cells without
// applying any style. Used where the caller composed styling itself.
func fitLine(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if ansi.StringWidth(s) > w {
		s = ansi.Truncate(s, w, "")
	}
	if pad := w - ansi.StringWidth(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

// fitLines returns exactly h lines, each exactly w cells.
func fitLines(lines []string, w, h int) []string {
	out := make([]string, 0, max(h, 0))
	for i := 0; i < h; i++ {
		if i < len(lines) {
			out = append(out, fitLine(lines[i], w))
		} else {
			out = append(out, strings.Repeat(" ", max(w, 0)))
		}
	}
	return out
}

func splitLines(s string) []string { return strings.Split(s, "\n") }

func blankLines(n int) []string {
	if n <= 0 {
		return nil
	}
	out := make([]string, n)
	return out
}

// rule renders a horizontal separator of exactly w cells.
func rule(st lipgloss.Style, ch string, w int) string {
	if w <= 0 {
		return ""
	}
	unit := ch
	uw := ansi.StringWidth(unit)
	if uw <= 0 {
		unit, uw = "─", 1
	}
	var b strings.Builder
	for written := 0; written < w; written += uw {
		b.WriteString(unit)
	}
	return st.Render(ansi.Truncate(b.String(), w, ""))
}

// placeCenter centers a block inside a w×h area, clipped to fit.
func placeCenter(block string, w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	lines := splitLines(block)
	if len(lines) > h {
		lines = lines[:h]
	}
	block = strings.Join(lines, "\n")
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, block)
}

// clipText truncates a plain string to w cells with an ellipsis when needed.
func clipText(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	return ansi.Truncate(s, w-1, "") + "…"
}

// indent prefixes every non-empty line with n spaces.
func indent(s string, n int) string {
	pad := strings.Repeat(" ", max(n, 0))
	lines := splitLines(s)
	for i, l := range lines {
		if l != "" {
			lines[i] = pad + l
		}
	}
	return strings.Join(lines, "\n")
}

// displayWidth is the ANSI-aware cell width of the widest line.
func displayWidth(s string) int {
	w := 0
	for _, l := range splitLines(s) {
		if lw := ansi.StringWidth(l); lw > w {
			w = lw
		}
	}
	return w
}
