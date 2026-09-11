package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"ghostty.studio/internal/core"
)

// choiceList is the shared left-hand list: catalog order preserved, filtered by
// a fuzzy query and an optional appearance filter, with favorites and
// session-recent entries lifted to the top of their group.
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
}

func newChoiceList(section string) *choiceList {
	return &choiceList{section: section, favs: map[string]bool{}}
}

func (l *choiceList) SetExtent(w, h int) {
	l.width, l.height = w, h
}

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
	step := max(l.height/2, 1)
	l.Move(dir * step)
}

func (l *choiceList) ensureVisible() {
	if l.height <= 0 {
		l.offset = 0
		return
	}
	if l.cursor < l.offset {
		l.offset = l.cursor
	}
	if l.cursor >= l.offset+l.height {
		l.offset = l.cursor - l.height + 1
	}
	if maxOff := len(l.view) - l.height; l.offset > maxOff {
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
	end := min(l.offset+l.height, len(l.view))
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

// rowOpts configures per-section row decoration.
type rowOpts struct {
	swatch bool
	suffix func(core.Choice) string
}

// View renders exactly l.height rows of exactly l.width cells.
func (l *choiceList) View(st *styles, opts rowOpts) []string {
	rows := make([]string, 0, l.height)
	visible, offset := l.Visible()
	for i, c := range visible {
		rows = append(rows, l.row(st, c, offset+i == l.cursor, opts))
	}
	for len(rows) < l.height {
		rows = append(rows, strings.Repeat(" ", l.width))
	}
	return rows
}

func (l *choiceList) row(st *styles, c core.Choice, selected bool, opts rowOpts) string {
	w := l.width
	marker := " "
	markStyle := st.Marker
	rowStyle := st.Row
	if selected {
		marker = "▸"
		markStyle = st.MarkerSel
		rowStyle = st.RowSel
	}

	right := ""
	if opts.swatch {
		right = swatch(c)
	} else if opts.suffix != nil {
		right = opts.suffix(c)
	} else {
		right = choiceGlyphs(c, l.favs[c.ID])
	}
	rightW := lipgloss.Width(right)

	nameW := max(w-2-rightW-1, 4)
	name := clipText(c.Name, nameW)
	nameStyled := styleName(st, name, l.query, selected)
	pad := strings.Repeat(" ", max(nameW-lipgloss.Width(name), 0))

	line := markStyle.Render(marker) + rowStyle.Render(" "+nameStyled+pad+" ") + right
	return fitLine(line, w)
}

// styleName highlights fuzzy matches inside a row label.
func styleName(st *styles, name, query string, selected bool) string {
	base := st.Text
	if selected {
		base = st.RowSel
	}
	idx := matchIndexes(query, name)
	if len(idx) == 0 {
		return base.Render(name)
	}
	var b strings.Builder
	runes := []rune(name)
	next := 0
	for i, r := range runes {
		matched := next < len(idx) && idx[next] == i
		if matched {
			next++
		}
		if matched {
			b.WriteString(st.Amber.Render(string(r)))
			continue
		}
		b.WriteString(base.Render(string(r)))
	}
	return b.String()
}

// choiceGlyphs renders compact status glyphs for a row.
func choiceGlyphs(c core.Choice, fav bool) string {
	var b strings.Builder
	if fav {
		b.WriteString("★")
	} else {
		b.WriteString(" ")
	}
	if c.Installed {
		b.WriteString("•")
	} else {
		b.WriteString(" ")
	}
	return b.String()
}

// swatch renders 8 palette cells from a theme choice.
func swatch(c core.Choice) string {
	p := choicePalette(c)
	if p.Background == "" {
		return strings.Repeat(" ", 8)
	}
	var b strings.Builder
	for i := 0; i < 8; i++ {
		hex := p.ANSI[i]
		col, ok := parseHexColor(hex)
		if !ok {
			b.WriteString(" ")
			continue
		}
		b.WriteString(lipgloss.NewStyle().Background(col).Render(" "))
	}
	return b.String()
}
