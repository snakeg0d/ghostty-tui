package ui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// modalKind selects the modal layout.
type modalKind int

const (
	modalNone modalKind = iota
	modalInfo
	modalForm
	modalList
)

type modalField struct {
	label    string
	hint     string
	input    textinput.Model
	validate func(string) error
}

// modal is a blocking overlay. It never mutates configuration by itself; the
// root model owns every side effect through action identifiers.
type modal struct {
	kind     modalKind
	title    string
	subtitle string
	body     []string
	fields   []*modalField
	buttons  []string
	action   string
	payload  any

	items   []string
	itemIDs []string
	itemIdx int

	focusKind int // 0 = fields or list, 1 = buttons
	fieldIdx  int
	btnIdx    int
	scroll    int
}

// active reports whether any modal is showing.
func (md *modal) active() bool { return md != nil && md.kind != modalNone }

func (md *modal) selectedField() *modalField {
	if len(md.fields) == 0 || md.focusKind != 0 {
		return nil
	}
	if md.fieldIdx < 0 || md.fieldIdx >= len(md.fields) {
		return nil
	}
	return md.fields[md.fieldIdx]
}

// values returns the current field values in order.
func (md *modal) values() []string {
	out := make([]string, len(md.fields))
	for i, f := range md.fields {
		out[i] = strings.TrimSpace(f.input.Value())
	}
	return out
}

// fieldError reports the first invalid field, if any.
func (md *modal) fieldError() (string, error) {
	for _, f := range md.fields {
		if f.validate == nil {
			continue
		}
		if err := f.validate(strings.TrimSpace(f.input.Value())); err != nil {
			return f.label, err
		}
	}
	return "", nil
}

func (md *modal) focusField(i int) {
	md.focusKind = 0
	md.fieldIdx = clampInt(i, 0, max(len(md.fields)-1, 0))
	for idx, f := range md.fields {
		if idx == md.fieldIdx {
			f.input.Focus()
		} else {
			f.input.Blur()
		}
	}
}

func (md *modal) focusButtons() {
	md.focusKind = 1
	for _, f := range md.fields {
		f.input.Blur()
	}
	md.btnIdx = clampInt(md.btnIdx, 0, max(len(md.buttons)-1, 0))
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func inputStyles() textinput.Styles {
	var s textinput.Styles
	mk := func(fg lipgloss.Style) textinput.StyleState {
		return textinput.StyleState{
			Text:        lipgloss.NewStyle().Foreground(colText).Background(colSlate),
			Placeholder: lipgloss.NewStyle().Foreground(colMuted).Background(colSlate),
			Prompt:      fg.Background(colSlate),
			Suggestion:  lipgloss.NewStyle().Foreground(colMint).Background(colSlate),
		}
	}
	s.Focused = mk(lipgloss.NewStyle().Foreground(colMint))
	s.Blurred = mk(lipgloss.NewStyle().Foreground(colMuted))
	s.Cursor = textinput.CursorStyle{Color: colMint, Shape: tea.CursorBlock, Blink: true}
	return s
}

func newField(label, value, placeholder string, width int, validate func(string) error) *modalField {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = placeholder
	ti.SetStyles(inputStyles())
	ti.CharLimit = 0
	ti.SetValue(value)
	ti.CursorEnd()
	ti.SetWidth(max(width, 8))
	return &modalField{label: label, input: ti, validate: validate}
}

// renderModal lays the modal inside exactly w×h cells, centred.
func renderModal(md *modal, st *styles, w, h int, busy bool, spinnerFrame string) []string {
	if w <= 0 || h <= 0 {
		return blankLines(max(h, 0))
	}
	boxW := clampInt(w-8, 32, 76)
	boxW = min(boxW, w)
	inner := boxW - 4

	var content []string
	if md.title != "" {
		content = append(content, st.ModalTitle.Render(clipText(md.title, inner)))
	}
	if md.subtitle != "" {
		content = append(content, st.Dim.Render(clipText(md.subtitle, inner)))
	}

	fieldLines := 0
	for range md.fields {
		fieldLines += 2
	}
	if md.kind == modalList {
		fieldLines += len(md.items)
	}
	// Reserve actual chrome before allocating body rows. A fixed allowance
	// clipped the buttons and scrolling hint at 80×24.
	chrome := len(content) + fieldLines + 3 // borders and hint
	if len(md.buttons) > 0 {
		chrome += 2 // spacer and button row
	}
	bodyRoom := max(h-chrome, 0)
	if len(md.body) > bodyRoom && bodyRoom > 0 {
		bodyRoom = max(bodyRoom-2, 0) // indicators above and below a scrolled body
	}

	body := md.body
	md.scroll = clampInt(md.scroll, 0, max(len(body)-bodyRoom, 0))
	end := min(md.scroll+bodyRoom, len(body))
	overflow := len(body) > bodyRoom
	if overflow && md.scroll > 0 {
		content = append(content, st.Dim.Render("▲ more"))
	}
	for _, l := range body[md.scroll:end] {
		content = append(content, clipText(l, inner))
	}
	if len(body) > end {
		content = append(content, st.Dim.Render("▼ more"))
	}

	if md.kind == modalList {
		for i, item := range md.items {
			if md.focusKind == 0 && i == md.itemIdx {
				content = append(content, st.ModalSel.Render(" "+clipText(item, inner-2)+" "))
				continue
			}
			content = append(content, " "+clipText(item, inner-2))
		}
	}

	for i, f := range md.fields {
		focused := md.focusKind == 0 && i == md.fieldIdx
		labelStyle := st.Dim
		if focused {
			labelStyle = st.Mint
		}
		content = append(content, labelStyle.Render(clipText(f.label, inner)))
		view := f.input.View()
		if focused {
			view = st.Mint.Render("❯") + " " + view
		} else {
			view = st.Dim.Render(" ") + " " + view
		}
		content = append(content, clipText(view, inner))
	}

	if len(md.buttons) > 0 {
		content = append(content, "")
		content = append(content, md.buttonRow(st))
	}

	hint := "enter confirm   esc cancel"
	if len(md.fields) > 1 {
		hint = "tab next field   " + hint
	}
	if md.kind == modalList {
		hint = "j/k choose   " + hint
	}
	if len(md.buttons) > 1 {
		hint = "←/→ choose   " + hint
	}
	if md.kind == modalInfo && len(md.body) > bodyRoom {
		hint = "PgUp/PgDn scroll · ←/→ choose · enter · esc"
	}
	if busy {
		hint = spinnerFrame + " working…"
	}
	content = append(content, st.Dim.Render(clipText(hint, inner)))

	boxH := min(len(content)+2, h)
	contentLines := fitLines(content, inner, boxH-2)
	built := []string{st.Rule.Render("╭" + strings.Repeat("─", inner+2) + "╮")}
	for _, l := range contentLines {
		built = append(built, st.Rule.Render("│")+surfaceLine(st.Modal, " "+l+" ", inner+2)+st.Rule.Render("│"))
	}
	built = append(built, st.Rule.Render("╰"+strings.Repeat("─", inner+2)+"╯"))

	placed := placeCenter(strings.Join(built, "\n"), w, h)
	return fitLines(splitLines(placed), w, h)
}

func (md *modal) buttonRow(st *styles) string {
	parts := make([]string, 0, len(md.buttons))
	for i, b := range md.buttons {
		if md.focusKind == 1 && i == md.btnIdx {
			parts = append(parts, st.ButtonSel.Render(" "+b+" "))
			continue
		}
		parts = append(parts, st.Button.Render(" "+b+" "))
	}
	return strings.Join(parts, " ")
}
