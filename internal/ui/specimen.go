package ui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

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
	head              lipgloss.Style
	cursor, selection lipgloss.Style
	base, dim         lipgloss.Style
	comment, str      lipgloss.Style
	keyword           lipgloss.Style
	red, green        lipgloss.Style
	cyan, sand        lipgloss.Style
	plain             lipgloss.Style
	bold, italic      lipgloss.Style
}

func newSpecimenStyles(p core.Palette) specimenStyles {
	bgCol, fgCol := palettePair(p)
	comment, str, keyword, plain, red, green, cyan, sand := specimenSemantics(p)
	with := func(c color.Color) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(c).Background(bgCol)
	}
	return specimenStyles{
		bg:        lipgloss.NewStyle().Background(bgCol).Foreground(fgCol),
		head:      with(fgCol).Bold(true),
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
		sand:      with(sand),
		plain:     with(plain),
		bold:      with(fgCol).Bold(true),
		italic:    with(fgCol).Italic(true),
	}
}

// specGroup is one composable block of the specimen; priority decides which
// blocks survive when the panel is short.
type specGroup struct {
	rows     []string
	priority int
}

// frame draws the thin quiet frame with the mint title. The interior always
// carries the theme background, spaces included.
func frame(title string, w, h int, st *styles, s specimenStyles, body []string) string {
	if w < 10 || h < 4 {
		return ""
	}
	inner := w - 2
	textW := max(inner-2, 1)

	head := "─ " + clipText(title, max(inner-4, 1)) + " "
	top := st.Rule.Render("╭") + st.Mint.Render(head) +
		st.Rule.Render(strings.Repeat("─", max(inner-lipgloss.Width(head), 0))) + st.Rule.Render("╮")
	bar := st.Rule.Render("│")
	bottom := st.Rule.Render("╰") + st.Rule.Render(strings.Repeat("─", inner)) + st.Rule.Render("╯")

	rows := make([]string, 0, h)
	rows = append(rows, top)
	for i := range h - 2 {
		var content string
		if i < len(body) {
			content = body[i]
		}
		// Clip before styling: Width wraps an over-long string, which would
		// push a sample row through the frame and shift every row below it.
		if lipgloss.Width(content) > textW {
			content = ansi.Truncate(content, textW, "")
		}
		// The padding either side of the sample is painted with the theme
		// background too, so the interior reads as one surface.
		rows = append(rows, bar+surfaceLine(s.bg, " "+fitLine(content, textW)+" ", inner)+bar)
	}
	rows = append(rows, bottom)
	return strings.Join(rows, "\n")
}

// renderSpecimen draws the theme specimen: exactly w×h cells.
func renderSpecimen(d specimenData, w, h int, st *styles) string {
	s := newSpecimenStyles(d.Pal)
	return frame("terminal specimen", w, h, st, s, specimenBody(d, s, max(w-4, 1), max(h-2, 0)))
}

// renderTypeSpecimen draws the typography specimen used by the Fonts section.
func renderTypeSpecimen(d specimenData, w, h int, st *styles) string {
	s := newSpecimenStyles(d.Pal)
	return frame("type specimen", w, h, st, s, typeSpecimenBody(d, s, max(w-4, 1), max(h-2, 0)))
}

// composeSpecimen selects whole groups by priority, always keeps the ribbon,
// and never emits a partially drawn block. Rows are never clipped mid-sample.
func composeSpecimen(avail int, groups []specGroup, ribbon []string, blank string) []string {
	if avail <= 0 {
		return nil
	}
	reserve := 0
	if len(ribbon) > 0 {
		reserve = len(ribbon) + 1
	}
	order := make([]int, 0, len(groups))
	for i := range groups {
		order = append(order, i)
	}
	// Stable selection by priority, then by display order.
	for i := 1; i < len(order); i++ {
		for j := i; j > 0; j-- {
			a, b := groups[order[j-1]], groups[order[j]]
			if a.priority < b.priority || (a.priority == b.priority && order[j-1] < order[j]) {
				break
			}
			order[j-1], order[j] = order[j], order[j-1]
		}
	}

	included := make([]bool, len(groups))
	used := 0
	for _, i := range order {
		cost := len(groups[i].rows)
		if used > 0 {
			cost++
		}
		if used+cost <= avail-reserve {
			included[i] = true
			used += cost
		}
	}

	out := make([]string, 0, avail)
	for i, g := range groups {
		if !included[i] {
			continue
		}
		if len(out) > 0 {
			out = append(out, blank)
		}
		out = append(out, g.rows...)
	}
	if len(ribbon) > 0 {
		switch {
		case len(out) == 0:
			out = append(out, ribbon...)
		case len(out)+1+len(ribbon) <= avail:
			out = append(out, blank)
			out = append(out, ribbon...)
		}
	}
	for len(out) < avail {
		out = append(out, blank)
	}
	return out[:avail]
}

// specimenBody composes the theme sample. The selected name leads, the palette
// ribbon is the signature near the bottom, and nothing reports invented work.
func specimenBody(d specimenData, s specimenStyles, textW, avail int) []string {
	blank := ""
	heading := headingRow(d.Title, s, textW, "SAMPLE")

	code := specGroup{priority: 3, rows: []string{
		s.keyword.Render("package") + s.base.Render(" studio"),
		s.comment.Render("// a palette, a size, a mood"),
		s.keyword.Render("func") + s.base.Render(" Look(n ") + s.keyword.Render("int") + s.base.Render(") []") + s.keyword.Render("string") + s.base.Render(" {"),
		"    " + s.keyword.Render("return") + s.base.Render(" theme.") + s.str.Render("ANSI") + s.base.Render("[:n]"),
		s.base.Render("}"),
	}}
	diff := specGroup{priority: 2, rows: []string{
		s.red.Render("- background = #101518"),
		s.green.Render("+ background = " + hexLabel(d.Pal.Background)),
	}}
	prompt := specGroup{priority: 3, rows: []string{
		s.green.Render("~/projects/atlas") + s.dim.Render("  main ❯ ") + s.base.Render("git status"),
		s.dim.Render("nothing to commit, working tree clean"),
	}}
	type0 := specGroup{priority: 4, rows: []string{
		s.base.Render("0 O o  1 l I |  { [ (  => != <= ") + s.dim.Render("— typography"),
	}}
	if textW < 48 {
		type0.rows = []string{s.base.Render("0 O o  1 l I |  => != <=")}
	}

	return reserveHeading(heading, avail, func(budget int) []string {
		return composeSpecimen(budget, []specGroup{code, diff, prompt, type0}, ribbonRows(d.Pal, s, textW), blank)
	})
}

// reserveHeading spends one row on the heading and one on the gap beneath it,
// then hands the remaining budget to the group composer. The heading can never
// overwrite a sample row.
func reserveHeading(heading string, avail int, compose func(budget int) []string) []string {
	if avail <= 0 {
		return nil
	}
	if avail == 1 {
		return []string{heading}
	}
	body := compose(avail - 2)
	out := make([]string, 0, avail)
	out = append(out, heading, "")
	out = append(out, body...)
	for len(out) < avail {
		out = append(out, "")
	}
	return out[:avail]
}

// typeSpecimenBody composes the dedicated typography sample: real letterforms,
// ambiguous pairs and weights, never a claim that the terminal font changed.
func typeSpecimenBody(d specimenData, s specimenStyles, textW, avail int) []string {
	blank := ""
	heading := headingRow(d.Title, s, textW, "SAMPLE")

	upper := specGroup{priority: 2, rows: []string{s.base.Render("ABCDEFGHIJKLMNOPQRSTUVWXYZ")}}
	lower := specGroup{priority: 1, rows: []string{s.base.Render("abcdefghijklmnopqrstuvwxyz")}}
	digits := specGroup{priority: 2, rows: []string{
		s.base.Render("0123456789") + s.dim.Render("     ") + s.base.Render("!?&@#$%*(){}[]"),
	}}
	ambiguous := specGroup{priority: 1, rows: []string{
		s.base.Render("0 O o    1 l I |    rn m    ") + s.dim.Render("ambiguity"),
	}}
	regular := specGroup{priority: 1, rows: []string{
		s.dim.Render("regular   ") + s.base.Render("The quick brown fox jumps over the lazy dog"),
	}}
	bold := specGroup{priority: 2, rows: []string{
		s.dim.Render("bold      ") + s.bold.Render("Sphinx of black quartz, judge my vow"),
	}}
	italic := specGroup{priority: 2, rows: []string{
		s.dim.Render("italic    ") + s.italic.Render("Pack my box with five dozen liquor jugs"),
	}}
	if textW < 55 {
		regular.rows = []string{s.dim.Render("regular   ") + s.base.Render("The quick brown fox")}
		bold.rows = []string{s.dim.Render("bold      ") + s.bold.Render("Sphinx of black quartz")}
		italic.rows = []string{s.dim.Render("italic    ") + s.italic.Render("Pack my box with five dozen")}
	}
	ligatures := specGroup{priority: 4, rows: []string{
		s.base.Render("=> != <= -> ===    ") + s.dim.Render("sequences"),
	}}
	box := specGroup{priority: 4, rows: []string{
		s.base.Render("─────  │  ┼  ┄  ┌─┬─┐  └─┴─┘"),
	}}

	groups := []specGroup{upper, lower, digits, ambiguous, regular, bold, italic, ligatures, box}
	return reserveHeading(heading, avail, func(budget int) []string {
		return composeSpecimen(budget, groups, nil, blank)
	})
}

// headingRow gives the selected name a real typographic heading, tagged so the
// sample can never be mistaken for live output.
func headingRow(title string, s specimenStyles, w int, tag string) string {
	name := title
	if strings.TrimSpace(name) == "" {
		name = "current appearance"
	}
	room := max(w-len(tag)-2, 4)
	name = clipText(name, room)
	pad := max(w-lipgloss.Width(name)-len(tag), 1)
	return s.head.Render(name) + s.bg.Render(strings.Repeat(" ", pad)) + s.dim.Render(tag)
}

// ribbonRows is the signature 16-color ribbon plus concise hex metadata.
func ribbonRows(p core.Palette, s specimenStyles, w int) []string {
	cell, sep := 1, 1
	if w >= 50 {
		cell, sep = 2, 1
	}
	cols := paletteANSI(p)
	parts := make([]string, 0, len(cols))
	for _, c := range cols {
		parts = append(parts, lipgloss.NewStyle().Background(c).Render(strings.Repeat(" ", cell)))
	}
	gap := lipgloss.NewStyle().Background(validColorOr(p.Background, colorHex(mustBg(p)))).Render(strings.Repeat(" ", sep))
	ribbon := strings.Join(parts, gap)

	meta := "bg " + hexLabel(p.Background) + "   fg " + hexLabel(p.Foreground)
	return []string{ribbon, s.dim.Render(clipText(meta, w))}
}

func mustBg(p core.Palette) color.Color {
	bg, _ := palettePair(p)
	return bg
}

// ----- compatibility helpers used by panes and tests -----

// SpecimenStatic renders the theme specimen with the studio chrome. The CLI
// `specimen` command and tests use it directly.
func SpecimenStatic(p core.Palette, title string, w, h int) string {
	return renderSpecimen(specimenData{Title: title, Pal: p}, w, h, newStyles())
}

func hexLabel(hex string) string {
	if hex == "" {
		return "#000000"
	}
	return hex
}
