package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"ghostty.studio/internal/core"
)

// choiceList is the shared left-hand list: catalog order preserved, filtered by
// a fuzzy query and an optional appearance filter, with favorites and
// session-recent entries lifted to the top of their group. Entries may occupy
// one or two terminal rows.
type choiceList struct {
	section string
	items   []core.Choice
	view    []core.Choice
	cursor  int
	offset  int
	query   string
	filter  FilterMode
	favs    map[string]bool
	recent  []string
	width   int
	height  int
	lines   int
}

func newChoiceList(section string) *choiceList {
	return &choiceList{section: section, favs: map[string]bool{}, lines: 1}
}

// SetExtent gives the list its exact cell budget and row height.
func (l *choiceList) SetExtent(w, h, lines int) {
	l.width, l.height = w, h
	l.lines = clampInt(lines, 1, 2)
	l.ensureVisible()
}

// cap is how many entries fit in the current extent.
func (l *choiceList) cap() int { return max(l.height/l.lines, 1) }

func (l *choiceList) SetItems(items []core.Choice) {
	l.items = items
	l.refilter()
}

func (l *choiceList) SetQuery(q string) {
	if l.query == q {
		return
	}
	l.query = q
	l.refilter()
}

func (l *choiceList) SetFilter(f FilterMode) {
	if l.filter == f {
		return
	}
	l.filter = f
	l.refilter()
}

func (l *choiceList) SetFavorites(favs map[string]bool) {
	l.favs = favs
	l.refilter()
}

// Items returns the unfiltered catalog slice for this list.
func (l *choiceList) Items() []core.Choice { return l.items }

// MarkRecent records an item as recently used for this session.
func (l *choiceList) MarkRecent(id string) {
	if id == "" {
		return
	}
	out := []string{id}
	for _, r := range l.recent {
		if r != id {
			out = append(out, r)
		}
	}
	if len(out) > 8 {
		out = out[:8]
	}
	l.recent = out
	l.refilter()
}

func (l *choiceList) Filtered() []core.Choice { return l.view }

func (l *choiceList) Len() int { return len(l.view) }

func (l *choiceList) Total() int { return len(l.items) }

func (l *choiceList) Cursor() int { return l.cursor }

func (l *choiceList) Filter() FilterMode { return l.filter }

func (l *choiceList) Selected() (core.Choice, int, bool) {
	if len(l.view) == 0 || l.cursor < 0 || l.cursor >= len(l.view) {
		return core.Choice{}, -1, false
	}
	return l.view[l.cursor], l.cursor, true
}

// Find returns the index of a choice ID inside the filtered view.
func (l *choiceList) Find(id string) int {
	for i, c := range l.view {
		if c.ID == id {
			return i
		}
	}
	return -1
}

// Select moves the cursor to a choice ID when it is present in the view.
func (l *choiceList) Select(id string) bool {
	if i := l.Find(id); i >= 0 {
		l.cursor = i
		l.ensureVisible()
		return true
	}
	return false
}

func (l *choiceList) Move(delta int) {
	if len(l.view) == 0 {
		l.cursor, l.offset = 0, 0
		return
	}
	l.cursor += delta
	if l.cursor < 0 {
		l.cursor = 0
	}
	if l.cursor > len(l.view)-1 {
		l.cursor = len(l.view) - 1
	}
	l.ensureVisible()
}

func (l *choiceList) MoveTop() {
	l.cursor = 0
	l.ensureVisible()
}

func (l *choiceList) MoveBottom() {
	if len(l.view) > 0 {
		l.cursor = len(l.view) - 1
	}
	l.ensureVisible()
}

func (l *choiceList) HalfPage(dir int) {
	l.Move(dir * max(l.cap()/2, 1))
}

func (l *choiceList) ensureVisible() {
	if l.height <= 0 {
		l.offset = 0
		return
	}
	visible := l.cap()
	if l.cursor < l.offset {
		l.offset = l.cursor
	}
	if l.cursor >= l.offset+visible {
		l.offset = l.cursor - visible + 1
	}
	if maxOff := len(l.view) - visible; l.offset > maxOff {
		l.offset = max(maxOff, 0)
	}
	if l.offset < 0 {
		l.offset = 0
	}
}

// Visible reports the slice of the view currently rendered.
func (l *choiceList) Visible() ([]core.Choice, int) {
	if l.height <= 0 {
		return nil, 0
	}
	end := min(l.offset+l.cap(), len(l.view))
	return l.view[l.offset:end], l.offset
}

// refilter recomputes the visible ordering. Catalog order is preserved inside
// each group: favorites first, then session-recent, then everything else.
func (l *choiceList) refilter() {
	matched := make([]core.Choice, 0, len(l.items))
	for _, c := range l.items {
		if !l.matches(c) {
			continue
		}
		matched = append(matched, c)
	}
	l.view = orderChoices(matched, l.favs, l.recent)
	if l.cursor >= len(l.view) {
		l.cursor = max(len(l.view)-1, 0)
	}
	l.ensureVisible()
}

func (l *choiceList) matches(c core.Choice) bool {
	switch l.filter {
	case FilterDark:
		if themeIsLight(c) {
			return false
		}
	case FilterLight:
		if !themeIsLight(c) {
			return false
		}
	case FilterFavorites:
		if !l.favs[c.ID] {
			return false
		}
	}
	return matchesQuery(l.query, c)
}

// matchesQuery keeps the two match strengths apart. Names and IDs accept a
// subsequence, so "tn" finds TokyoNight; prose fields (description, tags,
// category, source) only match contiguously, so "tokyo" no longer drags in a
// theme whose scattered letters happen to appear across combined fields.
func matchesQuery(query string, c core.Choice) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	if fuzzyMatch(q, strings.ToLower(c.Name)) || fuzzyMatch(q, strings.ToLower(c.ID)) {
		return true
	}
	for _, field := range []string{c.Description, c.Source, c.Category, strings.Join(c.Tags, " ")} {
		if strings.Contains(strings.ToLower(field), q) {
			return true
		}
	}
	return false
}

// fuzzyMatch reports whether every rune of query appears in target in order.
func fuzzyMatch(query, target string) bool {
	q := []rune(strings.ToLower(strings.TrimSpace(query)))
	t := []rune(strings.ToLower(target))
	if len(q) == 0 {
		return true
	}
	i := 0
	for _, r := range t {
		if r == q[i] {
			i++
			if i == len(q) {
				return true
			}
		}
	}
	return false
}

// matchIndexes returns the target rune indexes matched by the query.
func matchIndexes(query, target string) []int {
	q := []rune(strings.ToLower(strings.TrimSpace(query)))
	if len(q) == 0 {
		return nil
	}
	t := []rune(strings.ToLower(target))
	out := make([]int, 0, len(q))
	i := 0
	for idx, r := range t {
		if i < len(q) && r == q[i] {
			out = append(out, idx)
			i++
		}
	}
	if i != len(q) {
		return nil
	}
	return out
}

func orderChoices(items []core.Choice, favs map[string]bool, recent []string) []core.Choice {
	if len(items) == 0 {
		return nil
	}
	recentRank := make(map[string]int, len(recent))
	for i, id := range recent {
		recentRank[id] = i
	}
	var favsOut, recentOut, rest []core.Choice
	for _, c := range items {
		switch {
		case favs[c.ID]:
			favsOut = append(favsOut, c)
		case isRecent(recentRank, c.ID):
			recentOut = append(recentOut, c)
		default:
			rest = append(rest, c)
		}
	}
	sortByRecent(recentOut, recentRank)
	out := make([]core.Choice, 0, len(items))
	out = append(out, favsOut...)
	out = append(out, recentOut...)
	out = append(out, rest...)
	return out
}

func sortByRecent(items []core.Choice, rank map[string]int) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && rank[items[j-1].ID] > rank[items[j].ID]; j-- {
			items[j-1], items[j] = items[j], items[j-1]
		}
	}
}

func isRecent(rank map[string]int, id string) bool {
	_, ok := rank[id]
	return ok
}

// rowOpts configures per-section row decoration.
type rowOpts struct {
	swatch  bool
	subtype func(core.Choice) string
}

// View renders exactly l.height rows of exactly l.width cells.
func (l *choiceList) View(st *styles, opts rowOpts) []string {
	rows := make([]string, 0, l.height)
	visible, offset := l.Visible()
	for i, c := range visible {
		rows = append(rows, l.row(st, c, offset+i == l.cursor, opts)...)
	}
	for len(rows) < l.height {
		rows = append(rows, strings.Repeat(" ", l.width))
	}
	return rows[:max(l.height, 0)]
}

// swatchPlan keeps the palette rhythm readable: a few separated swatches, never
// a continuous barcode, and fewer of them when the column is narrow. Two-line
// entries always show the full six.
func swatchPlan(listW int) (count, cell, sep int) {
	switch {
	case listW >= 34:
		return 6, 1, 1
	case listW >= 30:
		return 5, 1, 1
	case listW >= 26:
		return 4, 1, 1
	default:
		return 3, 1, 1
	}
}

func (l *choiceList) row(st *styles, c core.Choice, selected bool, opts rowOpts) []string {
	w := l.width
	rowStyle := st.Row
	railStyle := st.Rail
	subStyle := st.Sub
	if selected {
		rowStyle = st.RowSel
		railStyle = st.RailOn
		subStyle = st.SubSel
	}
	rail := railStyle.Render("▌")
	if !selected {
		rail = rowStyle.Render(" ")
	}

	star := l.star(st, c, selected)
	starW := lipgloss.Width(star)

	if l.lines >= 2 && opts.subtype != nil {
		// Roomy entries: the name owns the first line, the palette and the
		// entry's real subtype share the second. Nothing is drawn twice.
		nameW := max(w-3-starW, 4)
		shown := clipText(c.Name, nameW)
		pad := max(w-2-lipgloss.Width(shown)-starW, 0)
		first := rail + rowStyle.Render(" "+styleName(st, shown, l.query, selected)+strings.Repeat(" ", pad)) + star

		swatches := l.swatches(st, c, selected)
		head := " " + swatches
		used := 1 + lipgloss.Width(swatches)
		sub := opts.subtype(c)
		meta := ""
		if sub != "" {
			meta = "  " + clipText(sub, max(w-used-3, 4))
			used += lipgloss.Width(meta)
		}
		tail := strings.Repeat(" ", max(w-1-used, 0))
		return []string{surfaceLine(rowStyle, first, w), surfaceLine(rowStyle, rail+rowStyle.Render(head)+subStyle.Render(meta)+rowStyle.Render(tail), w)}
	}

	right := ""
	if opts.swatch {
		right = l.swatches(st, c, selected)
	}
	if star != "" {
		if right != "" {
			right += " "
		}
		right += star
	}
	rightW := lipgloss.Width(right)
	gap := 0
	if rightW > 0 {
		gap = 1
	}
	nameW := max(w-3-rightW-gap, 4)
	shown := clipText(c.Name, nameW)
	pad := max(w-2-lipgloss.Width(shown)-rightW, 0)
	return []string{surfaceLine(rowStyle, rail+rowStyle.Render(" "+styleName(st, shown, l.query, selected)+strings.Repeat(" ", pad))+right, w)}
}

// star renders the favorite marker for a row.
func (l *choiceList) star(st *styles, c core.Choice, selected bool) string {
	if !l.favs[c.ID] {
		return ""
	}
	if selected {
		return st.StarSel.Render("★")
	}
	return st.Star.Render("★")
}

// swatchStrip renders the separated swatch group for a two-line entry.
func (l *choiceList) swatchStrip(st *styles, c core.Choice, selected bool) string {
	return l.swatches(st, c, selected)
}

// swatches renders separated one-cell square glyphs. Colored glyphs on the
// chrome background break the continuous vertical barcode a solid block row
// would create, and they read as swatches in every terminal.
func (l *choiceList) swatches(st *styles, c core.Choice, selected bool) string {
	count, _, sep := swatchPlan(l.width)
	if l.lines >= 2 {
		count = 6
	}
	p := choicePalette(c)
	if p.Background == "" {
		return ""
	}
	bg := colCanvas
	if selected {
		bg = colSlate
	}
	sepStyle := lipgloss.NewStyle().Background(bg)
	parts := make([]string, 0, count)
	for i := range count {
		col, ok := parseHexColor(p.ANSI[1+i])
		if !ok {
			parts = append(parts, sepStyle.Render(" "))
			continue
		}
		parts = append(parts, lipgloss.NewStyle().Foreground(col).Background(bg).Render("▪"))
	}
	return strings.Join(parts, sepStyle.Render(strings.Repeat(" ", sep)))
}

// styleName highlights fuzzy matches inside a row label.
func styleName(st *styles, name, query string, selected bool) string {
	base := st.Text
	if selected {
		base = st.Text.Bold(true)
	}
	idx := matchIndexes(query, name)
	if len(idx) == 0 {
		return base.Render(name)
	}
	var b strings.Builder
	for i, r := range []rune(name) {
		if containsIndex(idx, i) {
			b.WriteString(st.Mint.Render(string(r)))
			continue
		}
		b.WriteString(base.Render(string(r)))
	}
	return b.String()
}

func containsIndex(idx []int, i int) bool {
	for _, v := range idx {
		if v == i {
			return true
		}
	}
	return false
}
