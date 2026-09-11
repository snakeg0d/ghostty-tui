package ui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"ghostty.studio/internal/core"
)

// specimenData describes the live terminal specimen.
type specimenData struct {
	Title string
	Pal   core.Palette
}

// specimenStyles are every color of the specimen. Each style carries the themed
// background so nested styled segments stay on the themed canvas.
type specimenStyles struct {
	bg                lipgloss.Style
	cursor, selection lipgloss.Style
	base, dim         lipgloss.Style
	comment, str      lipgloss.Style
	keyword           lipgloss.Style
	red, green        lipgloss.Style
	cyan, yellow      lipgloss.Style
	plain             lipgloss.Style
}

func newSpecimenStyles(p core.Palette) specimenStyles {
	bgCol, fgCol := palettePair(p)
	comment, str, keyword, plain, red, green, cyan, yellow := specimenSemantics(p)
	with := func(c color.Color) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(c).Background(bgCol)
	}
	return specimenStyles{
		bg:        lipgloss.NewStyle().Background(bgCol).Foreground(fgCol),
		cursor:    lipgloss.NewStyle().Background(validColorOr(p.Cursor, colorHex(fgCol))).Foreground(bgCol),
		selection: lipgloss.NewStyle().Background(validColorOr(p.Selection, colorHex(bgCol))).Foreground(fgCol),
		base:      with(fgCol),
		dim:       with(comment),
		comment:   with(comment),
		str:       with(str),
		keyword:   with(keyword),
		red:       with(red),
		green:     with(green),
		cyan:      with(cyan),
		yellow:    with(yellow),
		plain:     with(plain),
	}
}

// renderSpecimen draws the framed terminal specimen: exactly w×h cells, thin
// chrome border, themed interior.
func renderSpecimen(d specimenData, w, h int, st *styles) string {
	if w < 10 || h < 4 {
		return ""
	}
	inner := w - 2
	textW := max(inner-2, 1)
	s := newSpecimenStyles(d.Pal)

	title := "preview / " + d.Title
	if strings.TrimSpace(d.Title) == "" {
		title = "preview"
	}
	head := "─ " + clipText(title, max(inner-4, 1)) + " "
	// The frame stays a quiet Rule; the title itself is Sage so it reads.
	top := st.Rule.Render("╭") + st.Sage.Render(head) +
		st.Rule.Render(strings.Repeat("─", max(inner-lipgloss.Width(head), 0))) + st.Rule.Render("╮")

	bar := st.Rule.Render("│")
	bottom := st.Rule.Render("╰") + st.Rule.Render(strings.Repeat("─", inner)) + st.Rule.Render("╯")

	body := specimenBody(d.Pal, s)
	rows := make([]string, 0, h)
	rows = append(rows, top)
	for i := range h - 2 {
		var content string
		if i < len(body) {
			content = body[i]
		}
		rows = append(rows, bar+" "+s.bg.Width(textW).MaxWidth(textW).Render(content)+" "+bar)
	}
	rows = append(rows, bottom)
	return strings.Join(rows, "\n")
}

// specimenBody composes an illustrative sample: syntax-colored code, a diff, a
// shell prompt, agent output, typography and the ANSI swatch strip. Nothing
// here reports real work: diagnostics live in Health and Effects.
func specimenBody(p core.Palette, s specimenStyles) []string {
	meta := padRight(s.bg.Render("bg "), 3) + swatchCell(p.Background) + "  " +
		padRight(s.bg.Render("fg "), 3) + swatchCell(p.Foreground) + "  " +
		padRight(s.bg.Render("cu "), 3) + swatchCell(p.Cursor) + "  " +
		padRight(s.bg.Render("sel "), 4) + swatchCell(p.Selection)
	return []string{
		s.keyword.Render("package") + s.base.Render(" studio"),
		"",
		s.comment.Render("// a palette, a size, a mood"),
		s.keyword.Render("func") + s.base.Render(" Look(n ") + s.keyword.Render("int") + s.base.Render(") []") + s.keyword.Render("string") + s.base.Render(" {"),
		"    " + s.keyword.Render("return") + s.base.Render(" theme.") + s.str.Render("ANSI") + s.base.Render("[:n]"),
		s.base.Render("}"),
		"",
		s.red.Render("- background = #101518"),
		s.green.Render("+ background = " + hexLabel(p.Background)),
		s.dim.Render("  window-padding-x = 12"),
		"",
		s.green.Render("~/projects/atlas") + s.dim.Render("  main ❯ ") + s.base.Render("go build ./..."),
		s.base.Render("compiled ") + s.green.Render("42") + s.base.Render(" packages in 1.8s"),
		"",
		s.cyan.Render("→") + s.dim.Render("  3 files changed   +48 −12"),
		s.base.Render("0 O o  1 l I |  { [ (  => != <= ") + s.dim.Render("— typography"),
		swatchStrip(p),
		meta,
	}
}

// swatchStrip renders the 16 ANSI colors as background cells.
func swatchStrip(p core.Palette) string {
	var b strings.Builder
	for _, c := range paletteANSI(p) {
		b.WriteString(lipgloss.NewStyle().Background(c).Render(" "))
	}
	return b.String()
}

func swatchCell(hex string) string {
	c, ok := parseHexColor(hex)
	if !ok {
		return " "
	}
	return lipgloss.NewStyle().Background(c).Render(" ")
}

func padRight(s string, n int) string {
	if w := lipgloss.Width(s); w < n {
		return s + strings.Repeat(" ", n-w)
	}
	return s
}

func hexLabel(hex string) string {
	if hex == "" {
		return "#000000"
	}
	return hex
}

// SpecimenStatic renders the specimen with the studio chrome. The CLI
// `specimen` command and tests use it directly.
func SpecimenStatic(p core.Palette, title string, w, h int) string {
	return renderSpecimen(specimenData{Title: title, Pal: p}, w, h, newStyles())
}
