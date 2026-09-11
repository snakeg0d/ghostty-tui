package ui

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"ghostty.studio/internal/core"
)

// View implements tea.Model. The same View backs the snapshot CLI command, so
// what the parent reviews is exactly what the app renders.
func (m *Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.WindowTitle = "Ghostty Studio"
	v.BackgroundColor = colCanvas
	return v
}

// render composes exactly height lines of exactly width cells.
func (m *Model) render() string {
	w, h := m.width, m.height
	if w <= 0 || h <= 0 {
		w, h = 100, 30
	}
	if w < minWidth || h < minHeight {
		return strings.Join(fitLines(splitLines(m.tooSmall(w, h)), w, h), "\n")
	}
	header := m.headerLines(w)
	footer := m.footerLines(w)
	bodyH := h - len(header) - len(footer)
	if bodyH < 3 {
		bodyH = 3
	}
	body := m.bodyLines(w, bodyH)
	lines := make([]string, 0, h)
	lines = append(lines, header...)
	lines = append(lines, body...)
	lines = append(lines, footer...)
	return strings.Join(fitLines(lines, w, h), "\n")
}

// tooSmall is the truthful resize message below the supported minimum.
func (m *Model) tooSmall(w, h int) string {
	st := m.styles
	msg := []string{
		st.Amber.Render("Ghostty Studio"),
		"",
		st.Text.Render("needs at least 60 × 18 cells"),
		st.Dim.Render("current size " + strconv.Itoa(w) + " × " + strconv.Itoa(h)),
		"",
		st.Dim.Render("resize to continue    q quits"),
	}
	return placeCenter(strings.Join(msg, "\n"), w, h)
}

// ----- header -----

func (m *Model) headerLines(w int) []string {
	st := m.styles
	blank := st.App.Render(strings.Repeat(" ", w))
	return []string{
		blank,
		m.identityLine(w),
		blank,
		m.tabLine(w),
		m.searchLine(w),
		rule(st.Rule, "─", w),
	}
}

func (m *Model) identityLine(w int) string {
	st := m.styles
	left := "   " + st.Identity.Render("Ghostty Studio")
	if w >= 78 {
		left += "   " + st.Descriptor.Render("MAKE THE TERMINAL YOURS")
	}
	right := st.Version.Render(m.versionLabel())
	pad := max(w-lipgloss.Width(left)-lipgloss.Width(right)-3, 1)
	return st.App.Render(left + strings.Repeat(" ", pad) + right + "   ")
}

func (m *Model) versionLabel() string {
	if m.state.Version != "" {
		return "ghostty " + m.state.Version
	}
	return "ghostty unknown"
}

func (m *Model) tabLine(w int) string {
	st := m.styles
	var b strings.Builder
	b.WriteString("  ")
	for i, name := range sectionNames {
		sec := Section(i)
		label := name
		if sec == SecFonts && m.fontsDiscover {
			label = name + " · discover"
		}
		if sec == m.sec {
			b.WriteString(st.Amber.Render(" " + strconv.Itoa(i+1) + " " + label + " "))
			b.WriteString(st.Dim.Render(" "))
			continue
		}
		b.WriteString(st.TabIndex.Render(" " + strconv.Itoa(i+1) + " "))
		b.WriteString(st.Tab.Render(label + "   "))
	}
	return box(st.App, b.String(), w)
}

func (m *Model) searchLine(w int) string {
	st := m.styles
	label := st.Dim.Render(" / ")
	if m.searchOn {
		label = st.Amber.Render(" / ")
	}
	value := m.search.View()
	if m.searchOn {
		value = st.Amber.Render(value)
	}
	line := "  " + label + value
	if m.search.Value() != "" {
		line += st.Dim.Render("   enter keeps · esc clears")
	}
	return box(st.App, line, w)
}

// ----- footer -----

func (m *Model) footerLines(w int) []string {
	st := m.styles
	return []string{
		rule(st.RuleSoft, "─", w),
		m.detailLine(w),
		m.pendingLine(w),
		m.bindingLine(w),
	}
}

func (m *Model) detailLine(w int) string {
	st := m.styles
	left := m.detailText()
	right := m.countText()
	pad := max(w-lipgloss.Width(left)-lipgloss.Width(right)-3, 1)
	return st.App.Render("  " + left + strings.Repeat(" ", pad) + right + "   ")
}

func (m *Model) detailText() string {
	st := m.styles
	title, pal := m.previewTarget()
	if pal.Background == "" {
		return st.Dim.Render("no theme selected")
	}
	mode := "dark"
	if bg, ok := parseHexColor(pal.Background); ok && isLightColor(bg) {
		mode = "light"
	}
	parts := []string{mode, "contrast " + contrastLabel(pal)}
	if source := m.previewSource(); source != "" {
		parts = append(parts, source)
	}
	if title != "" {
		parts = append(parts, title)
	}
	return st.Sage.Render(strings.Join(parts, "  ·  "))
}

func (m *Model) countText() string {
	st := m.styles
	l := m.activeList()
	shown := l.Len()
	total := l.Total()
	switch m.sec {
	case SecThemes:
		return st.Dim.Render(strconv.Itoa(shown) + " of " + strconv.Itoa(total) + " themes")
	case SecFonts:
		if m.fontsDiscover {
			return st.Dim.Render(strconv.Itoa(shown) + " discover items")
		}
		return st.Dim.Render(strconv.Itoa(shown) + " of " + strconv.Itoa(total) + " families")
	default:
		return st.Dim.Render(strconv.Itoa(shown) + " of " + strconv.Itoa(total))
	}
}

func (m *Model) pendingLine(w int) string {
	st := m.styles
	left := st.Dim.Render("no staged changes")
	if m.dirty() {
		left = st.Amber.Render("staged  ") + st.Text.Render(m.pendingSummary())
	}
	if m.busy {
		left = st.Amber.Render(m.spinner.View() + " " + m.busyFor + "…")
	}
	right := ""
	if m.status != "" {
		if m.statusErr {
			right = st.Err.Render("✗ " + m.status)
		} else {
			right = st.OK.Render("✓ " + m.status)
		}
	}
	if right != "" {
		// Keep the result visible even while a long staged summary exists.
		right = clipText(right, max(w-4, 1))
		left = clipText(left, max(w-lipgloss.Width(right)-6, 0))
	}
	pad := max(w-lipgloss.Width(left)-lipgloss.Width(right)-3, 1)
	return st.App.Render("  " + left + strings.Repeat(" ", pad) + right + "   ")
}

func (m *Model) bindingLine(w int) string {
	st := m.styles
	m.help.SetWidth(max(w-4, 20))
	view := m.help.ShortHelpView(m.shortBindings())
	return "  " + box(st.App, view, max(w-4, 1)) + "  "
}

// shortBindings lists the footer bindings for the active focus.
func (m *Model) shortBindings() []key.Binding {
	b := m.keys
	extra := []key.Binding{}
	switch m.sec {
	case SecThemes:
		extra = []key.Binding{b.Fav, b.Filter, b.Pin}
	case SecFonts:
		extra = []key.Binding{b.Discover}
	case SecLooks:
		extra = []key.Binding{b.Save, b.Export, b.Import}
	default:
		extra = append(extra, b.PanePrev)
		switch m.more {
		case PaneEffects:
			extra = append(extra, b.Toggle, b.Clear)
		case PaneHealth:
			extra = append(extra, b.Refresh)
		}
	}
	base := []key.Binding{b.Stage, b.Apply, b.Preview, b.Search, b.Help}
	return append(base, extra...)
}

// ----- body -----

func (m *Model) bodyLines(w, h int) []string {
	if m.modal.active() {
		return renderModal(m.modal, m.styles, w, h, m.busy, m.spinner.View())
	}
	gutter := 2
	listW := m.listWidth()
	gap := 3
	rightW := max(w-gutter-listW-gap-2, 20)

	left := m.leftColumn(listW, h)
	right := m.rightPane(rightW, h)

	rows := make([]string, 0, h)
	for i := range h {
		var l, r string
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		rows = append(rows, strings.Repeat(" ", gutter)+fitLine(l, listW)+strings.Repeat(" ", gap)+fitLine(r, rightW))
	}
	return rows
}

func (m *Model) leftColumn(w, h int) []string {
	st := m.styles
	rows := []string{box(st.App, m.leftHeading(), w)}
	if m.sec == SecMore {
		rows = append(rows, box(st.App, m.paneStrip(), w))
	}
	listRows := max(h-len(rows), 0)
	list := m.activeList()
	list.SetExtent(w, listRows)
	body := list.View(st, m.rowOpts())
	if list.Len() == 0 {
		body = m.emptyState(listRows)
	}
	return append(rows, body...)
}

func (m *Model) leftHeading() string {
	st := m.styles
	switch m.sec {
	case SecThemes:
		h := st.Header.Render("THEMES")
		meta := []string{strconv.Itoa(m.themes.Len()) + " shown"}
		if m.themes.Filter() != FilterAll {
			meta = append(meta, m.themes.Filter().String())
		}
		if m.search.Value() != "" {
			meta = append(meta, "“"+clipText(m.search.Value(), 12)+"”")
		}
		return h + st.Dim.Render("   "+strings.Join(meta, " · "))
	case SecFonts:
		h := st.Header.Render("FONTS")
		if m.fontsDiscover {
			return h + st.Dim.Render("   discover")
		}
		return h + st.Dim.Render("   installed")
	case SecLooks:
		return st.Header.Render("LOOKS") + st.Dim.Render("   presets")
	default:
		return st.Header.Render("MORE")
	}
}

// paneStrip names the active More pane and the keys that reach the others.
// The left column is ~24–34 cells wide, so the strip stays short enough to
// keep both the pane name and the bracket hint on screen.
func (m *Model) paneStrip() string {
	st := m.styles
	return "   " + st.Amber.Render("[ "+shortPaneName(m.more)+" ] ") +
		st.Dim.Render("[ ]  ") +
		st.Sage.Render(strconv.Itoa(int(m.more)+1)+"/"+strconv.Itoa(len(morePaneNames)))
}

func (m *Model) rowOpts() rowOpts {
	switch m.sec {
	case SecThemes:
		return rowOpts{swatch: true}
	case SecLooks:
		return rowOpts{swatch: true}
	case SecFonts:
		return rowOpts{suffix: fontSuffix}
	default:
		return rowOpts{suffix: plainSuffix}
	}
}

func fontSuffix(c core.Choice) string {
	if c.Installed {
		return "●"
	}
	return "○"
}

func plainSuffix(c core.Choice) string {
	if c.Installed {
		return "●"
	}
	return " "
}

// emptyState is the well-designed no-results message.
func (m *Model) emptyState(h int) []string {
	st := m.styles
	var msg string
	switch {
	case m.search.Value() != "":
		msg = "No matches for “" + m.search.Value() + "”."
	case m.sec == SecThemes && m.themes.Filter() == FilterFavorites:
		msg = "No favorites yet. Press f on a theme to favorite it."
	case m.busy:
		msg = "Loading…"
	default:
		msg = "Nothing here yet."
	}
	out := make([]string, 0, h)
	out = append(out, "")
	out = append(out, st.Empty.Render(msg))
	if m.search.Value() != "" {
		out = append(out, "", st.Dim.Render("esc clears the search"))
	}
	for len(out) < h {
		out = append(out, "")
	}
	return out[:max(h, 0)]
}

// rightPane renders the active section's detail surface.
func (m *Model) rightPane(w, h int) []string {
	switch m.sec {
	case SecThemes:
		return m.themePane(w, h)
	case SecFonts:
		return m.fontPane(w, h)
	case SecLooks:
		return m.lookPane(w, h)
	default:
		return m.morePane(w, h)
	}
}

// previewTarget returns the title and palette the specimen should show.
func (m *Model) previewTarget() (string, core.Palette) {
	c, _, ok := m.activeList().Selected()
	if ok {
		if p := choicePalette(c); p.Background != "" && m.sec != SecFonts {
			return c.Name, p
		}
		if p := paletteFromValues(m.mergedValues()); p.Background != "" {
			return c.Name, p
		}
	}
	return "", paletteFromValues(m.mergedValues())
}

// previewSource describes where the focused item came from.
func (m *Model) previewSource() string {
	c, _, ok := m.activeList().Selected()
	if !ok {
		return ""
	}
	if c.Source != "" {
		return c.Source
	}
	return c.Category
}
