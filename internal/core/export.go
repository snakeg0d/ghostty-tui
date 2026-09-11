package core

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Export formats accepted by ExportTheme.
const (
	FormatGhostty = "ghostty"
	FormatFZF     = "fzf"
	FormatJSON    = "json"
)

// ExportTheme renders the palette of a value set in one of the supported
// formats. Colors are validated hex values, so the output can be pasted into a
// configuration file or a shell profile without executing anything.
func (e *Engine) ExportTheme(values Values, format string) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	palette := values.Palette()
	name := strings.TrimSpace(values.Last("theme"))
	if name == "" {
		name = "Ghostty Studio palette"
	}
	switch strings.ToLower(strings.TrimSpace(format)) {
	case FormatGhostty, "":
		return exportGhostty(palette), nil
	case FormatFZF:
		return exportFZF(palette), nil
	case FormatJSON:
		return exportJSON(name, palette)
	default:
		return "", fmt.Errorf("unknown export format %q; use ghostty, fzf or json", format)
	}
}

// exportGhostty renders a Ghostty configuration fragment.
func exportGhostty(palette Palette) string {
	var b strings.Builder
	b.WriteString("# Palette exported from Ghostty Studio\n")
	writeColor := func(key, value string) {
		if value == "" {
			return
		}
		fmt.Fprintf(&b, "%s = %s\n", key, value)
	}
	writeColor("background", palette.Background)
	writeColor("foreground", palette.Foreground)
	writeColor("cursor-color", palette.Cursor)
	writeColor("selection-background", palette.Selection)
	for i, colour := range palette.ANSI {
		if colour == "" {
			continue
		}
		fmt.Fprintf(&b, "palette = %d=%s\n", i, colour)
	}
	return b.String()
}

// exportFZF renders a shell variable snippet for fzf. Only validated hex colors
// are emitted, and nothing here is executed by the studio.
func exportFZF(palette Palette) string {
	slots := []struct {
		name  string
		value string
	}{
		{"bg", palette.Background},
		{"fg", palette.Foreground},
		{"hl", palette.Selection},
		{"fg+", palette.ANSI[15]},
		{"bg+", palette.ANSI[0]},
		{"hl+", palette.ANSI[5]},
		{"info", palette.ANSI[4]},
		{"prompt", palette.ANSI[2]},
		{"pointer", palette.ANSI[6]},
		{"marker", palette.ANSI[3]},
		{"spinner", palette.ANSI[6]},
		{"header", palette.ANSI[2]},
	}
	var parts []string
	for _, slot := range slots {
		colour, ok := NormalizeHexOK(slot.value)
		if !ok {
			continue
		}
		parts = append(parts, slot.name+":"+colour)
	}
	if len(parts) == 0 {
		return "# The current values do not contain a usable palette.\n"
	}
	return "# Palette exported from Ghostty Studio.\n" +
		"# Paste this into your shell profile; Ghostty Studio never edits it for you.\n" +
		`export FZF_DEFAULT_OPTS="$FZF_DEFAULT_OPTS --color=` + strings.Join(parts, ",") + "\"\n"
}

// paletteDocument is the JSON export schema.
type paletteDocument struct {
	Name                string   `json:"name"`
	Background          string   `json:"background"`
	Foreground          string   `json:"foreground"`
	CursorColor         string   `json:"cursorColor"`
	SelectionBackground string   `json:"selectionBackground"`
	ANSI                []string `json:"ansi"`
}

// exportJSON renders a machine readable palette document.
func exportJSON(name string, palette Palette) (string, error) {
	ansi := make([]string, 0, 16)
	for _, colour := range palette.ANSI {
		ansi = append(ansi, colour)
	}
	doc := paletteDocument{
		Name:                name,
		Background:          palette.Background,
		Foreground:          palette.Foreground,
		CursorColor:         palette.Cursor,
		SelectionBackground: palette.Selection,
		ANSI:                ansi,
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}
