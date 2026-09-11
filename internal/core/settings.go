package core

import (
	"fmt"
	"strconv"
	"strings"
)

// settingDoc is one configuration key documented by
// `ghostty +show-config --default --docs`. Repeated keys such as palette and
// keybind appear once with every default value they carry, in order.
type settingDoc struct {
	key         string
	description string
	defaults    []string
	options     []string
	reload      string
}

// curatedSetting is one key of the studio's typed editing schema. Everything
// outside this schema is cataloged for search and inspection only.
type curatedSetting struct {
	key         string
	group       string
	kind        string
	options     []string
	reload      string
	description string
}

// curatedSettings is the editable schema. Keys outside it are reference-only:
// they can be searched and read, but the studio will not offer a value editor
// for a setting whose value grammar it cannot validate safely.
var curatedSettings = []curatedSetting{
	{key: "font-family", group: GroupTypography, kind: "list", description: "Preferred font families, most preferred first. Repeated values add fallbacks."},
	{key: "font-size", group: GroupTypography, kind: "float", description: "Font size in points."},
	{key: "font-feature", group: GroupTypography, kind: "list", description: "OpenType features to enable or disable, for example -calt."},
	{key: "font-variation", group: GroupTypography, kind: "list", description: "Variable font axis values, for example wght=700."},
	{key: "font-style", group: GroupTypography, kind: "list", description: "Styles to enable, for example bold or italic."},
	{key: "font-thicken", group: GroupTypography, kind: "bool", description: "Draw glyphs with a heavier stroke; useful on low-resolution displays."},
	{key: "font-thicken-strength", group: GroupTypography, kind: "int", description: "Thickening strength from 0 to 255."},
	{key: "adjust-cell-height", group: GroupTypography, kind: "string", description: "Adjust cell height, as a percentage or pixel value."},
	{key: "adjust-cell-width", group: GroupTypography, kind: "string", description: "Adjust cell width, as a percentage or pixel value."},
	{key: "adjust-font-baseline", group: GroupTypography, kind: "string", description: "Shift the font baseline, as a percentage or pixel value."},

	{key: "window-padding-x", group: GroupWindow, kind: "string", description: "Horizontal window padding, as a pixel value or balance mode."},
	{key: "window-padding-y", group: GroupWindow, kind: "string", description: "Vertical window padding, as a pixel value or balance mode."},
	{key: "window-padding-balance", group: GroupWindow, kind: "bool", description: "Distribute leftover padding so cells stay centered."},
	{key: "background-opacity", group: GroupWindow, kind: "float", description: "Background opacity from 0.0 (transparent) to 1.0 (opaque)."},
	{key: "background-blur", group: GroupWindow, kind: "string", description: "Background blur: true, false, or a radius in pixels."},
	{key: "background-image", group: GroupWindow, kind: "path", description: "Background image file, repeated to layer images."},
	{key: "background-image-opacity", group: GroupWindow, kind: "float", description: "Opacity of the background image."},
	{key: "window-decoration", group: GroupWindow, kind: "enum", description: "Window decorations toggled by Ghostty.",
		options: []string{"auto", "none", "client", "server"}},
	{key: "window-save-state", group: GroupWindow, kind: "enum", description: "Whether window state is saved and restored.",
		options: []string{"default", "never", "always"}},
	{key: "window-width", group: GroupWindow, kind: "int", description: "Initial window width in cells."},
	{key: "window-height", group: GroupWindow, kind: "int", description: "Initial window height in cells."},
	{key: "macos-titlebar-style", group: GroupWindow, kind: "enum", description: "Titlebar style used on macOS.",
		options: []string{"native", "transparent", "tabs", "hidden"}},

	{key: "cursor-style", group: GroupCursor, kind: "enum", description: "Default cursor style.",
		options: []string{"block", "bar", "underline", "block_hollow"}},
	{key: "cursor-style-blink", group: GroupCursor, kind: "bool", description: "Blink the cursor when the running program does not choose."},
	{key: "cursor-opacity", group: GroupCursor, kind: "float", description: "Cursor opacity from 0.0 to 1.0."},
	{key: "cursor-color", group: GroupCursor, kind: "color", description: "Cursor color."},
	{key: "cursor-text", group: GroupCursor, kind: "color", description: "Text color drawn under the cursor."},
	{key: "mouse-hide-while-typing", group: GroupCursor, kind: "bool", description: "Hide the mouse pointer while typing."},

	{key: "scrollback-limit", group: GroupBehavior, kind: "int", description: "Scrollback buffer size in bytes."},
	{key: "confirm-close-surface", group: GroupBehavior, kind: "enum", description: "Ask before closing a surface that is still running a process.",
		options: []string{"true", "false", "always"}},
	{key: "copy-on-select", group: GroupBehavior, kind: "enum", description: "Copy the selection to the clipboard when it is released.",
		options: []string{"true", "false", "clipboard"}},
	{key: "shell-integration", group: GroupBehavior, kind: "enum", description: "Shell integration to enable.",
		options: []string{"none", "detect", "bash", "elvish", "fish", "zsh"}},
	{key: "shell-integration-features", group: GroupBehavior, kind: "list", description: "Shell integration features, for example no-cursor,title,path."},
	{key: "quick-terminal-position", group: GroupBehavior, kind: "enum", description: "Edge or corner the quick terminal drops down from."},
	{key: "quick-terminal-autohide", group: GroupBehavior, kind: "bool", description: "Hide the quick terminal when it loses focus."},
	{key: "quick-terminal-screen", group: GroupBehavior, kind: "enum", description: "Display the quick terminal appears on.",
		options: []string{"mouse", "main", "macos-menu-bar"}},
	{key: "quick-terminal-animation-duration", group: GroupBehavior, kind: "duration", description: "Quick terminal animation duration in seconds."},
	{key: "desktop-notifications", group: GroupBehavior, kind: "bool", description: "Allow desktop notifications from the terminal."},
	{key: "notify-on-command-finish", group: GroupBehavior, kind: "enum", description: "Notify when a command finishes.",
		options: []string{"never", "unfocused", "always"}},
	{key: "notify-on-command-finish-after", group: GroupBehavior, kind: "duration", description: "Only notify for commands that ran at least this long."},
	{key: "window-inherit-working-directory", group: GroupBehavior, kind: "bool", description: "New windows inherit the working directory of the current surface."},
	{key: "focus-follows-mouse", group: GroupBehavior, kind: "bool", description: "Focus follows the mouse pointer."},
	{key: "minimum-contrast", group: GroupBehavior, kind: "float", description: "Minimum contrast ratio between foreground and background."},
	{key: "background", group: GroupBehavior, kind: "color", description: "Background color."},
	{key: "foreground", group: GroupBehavior, kind: "color", description: "Foreground color."},
	{key: "selection-background", group: GroupBehavior, kind: "color", description: "Selection background color."},
	{key: "selection-foreground", group: GroupBehavior, kind: "color", description: "Selection foreground color."},
	{key: "theme", group: GroupBehavior, kind: "string", description: "Color theme name, optionally as light:NAME,dark:NAME."},
	{key: "palette", group: GroupBehavior, kind: "list", description: "Palette override in index=#rrggbb form."},
	{key: "custom-shader", group: GroupBehavior, kind: "list", description: "Custom GLSL shader paths, applied in order."},
	{key: "custom-shader-animation", group: GroupBehavior, kind: "enum", description: "Animate custom shaders.",
		options: []string{"true", "false", "always"}},

	{key: "keybind", group: GroupShortcuts, kind: "keybind", description: "Key binding in trigger=action form."},
	{key: "key-remap", group: GroupShortcuts, kind: "keybind", description: "Remap one key to another."},
	{key: "macos-option-as-alt", group: GroupShortcuts, kind: "enum", description: "Send Option as Alt on macOS.",
		options: []string{"true", "false", "left", "right"}},
}

// curatedIndex maps curated keys back to their schema entry.
func curatedIndex() map[string]curatedSetting {
	index := make(map[string]curatedSetting, len(curatedSettings))
	for _, entry := range curatedSettings {
		index[entry.key] = entry
	}
	return index
}

// parseDocDump parses `ghostty +show-config --default --docs` into one record
// per configuration key.
func parseDocDump(content []byte) []settingDoc {
	var docs []settingDoc
	index := make(map[string]int)
	var comments []string
	for _, raw := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimLeft(raw, " \t")
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			comments = append(comments, cleanComment(trimmed))
			continue
		}
		eq := strings.IndexByte(trimmed, '=')
		if eq < 0 {
			comments = nil
			continue
		}
		key := strings.TrimSpace(trimmed[:eq])
		if key == "" {
			comments = nil
			continue
		}
		value := strings.TrimSpace(trimmed[eq+1:])
		// Ghostty prints one line per value, so repeated keys must be folded
		// into a single entry or the reference lists the same key many times.
		if at, ok := index[key]; ok {
			docs[at].defaults = append(docs[at].defaults, value)
			comments = nil
			continue
		}
		index[key] = len(docs)
		docs = append(docs, settingDoc{
			key:         key,
			description: strings.TrimSpace(strings.Join(comments, "\n")),
			defaults:    []string{value},
			reload:      reloadSemantics(comments),
			options:     parseOptions(comments),
		})
		comments = nil
	}
	return docs
}

// cleanComment strips the leading comment marker and one space.
func cleanComment(line string) string {
	line = strings.TrimPrefix(line, "#")
	return strings.TrimPrefix(line, " ")
}

// reloadSemantics reads Ghostty's own statement about reloadability.
func reloadSemantics(comments []string) string {
	for _, line := range comments {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "cannot be reloaded") || strings.Contains(lower, "must fully restart") {
			return "restart"
		}
	}
	return "reload"
}

// parseOptions extracts the bulleted "Valid values" list of a documentation
// block when the block declares one.
func parseOptions(comments []string) []string {
	start := -1
	for i, line := range comments {
		if strings.HasPrefix(strings.TrimSpace(line), "Valid values") {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return nil
	}
	var options []string
	for _, line := range comments[start:] {
		text := strings.TrimSpace(line)
		if text == "" {
			continue
		}
		if !strings.HasPrefix(text, "*") {
			if line != strings.TrimLeft(line, " \t") {
				// Indented prose continuing the previous bullet.
				continue
			}
			break
		}
		value, ok := optionValue(strings.TrimSpace(strings.TrimPrefix(text, "*")))
		if !ok {
			continue
		}
		options = append(options, value)
	}
	return options
}

// optionValue extracts the value a documentation bullet enumerates. Ghostty
// writes enumerated values in backticks, and some keys list accepted value
// *forms* as plain prose instead; prose must never become a value hint.
func optionValue(entry string) (string, bool) {
	if entry == "" {
		return "", false
	}
	if quote := entry[0]; quote == '`' || quote == '"' {
		rest := entry[1:]
		if end := strings.IndexByte(rest, quote); end >= 0 {
			value := strings.TrimSpace(rest[:end])
			return value, value != ""
		}
		// An unclosed quote is prose, not a value.
		return "", false
	}
	fields := strings.Fields(entry)
	if len(fields) != 1 {
		return "", false
	}
	word := strings.TrimRight(fields[0], ".,;:")
	if word == "" || strings.HasPrefix(word, "-") {
		return "", false
	}
	return word, true
}

// inferKind guesses the value grammar of a key when Ghostty's own docs do not
// enumerate it.
func inferKind(key, value string, options []string, repeatable bool) string {
	switch {
	case len(options) > 0:
		return "enum"
	case strings.HasPrefix(key, "keybind") || key == "key-remap":
		return "keybind"
	}
	switch value {
	case "true", "false":
		return "bool"
	}
	if value != "" {
		if _, err := strconv.Atoi(value); err == nil {
			return "int"
		}
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			return "float"
		}
		if isColorValue(value) {
			return "color"
		}
		if isDurationValue(key, value) {
			return "duration"
		}
		if strings.Contains(value, "/") || strings.Contains(value, "~") {
			return "path"
		}
	}
	switch {
	case repeatable:
		return "list"
	case strings.Contains(key, "opacity") || strings.Contains(key, "contrast"):
		return "float"
	case strings.HasSuffix(key, "-size") || strings.HasSuffix(key, "-limit"):
		return "int"
	case strings.Contains(key, "color") || strings.Contains(key, "background") || strings.Contains(key, "foreground"):
		return "color"
	case strings.Contains(key, "path") || strings.HasSuffix(key, "-image") || strings.HasSuffix(key, "-file"):
		return "path"
	case strings.Contains(key, "duration") || strings.HasSuffix(key, "-after") || strings.HasSuffix(key, "-timeout") || strings.HasSuffix(key, "-delay"):
		return "duration"
	}
	return "string"
}

func isColorValue(value string) bool {
	_, ok := NormalizeHexOK(value)
	return ok
}

func isDurationValue(key, value string) bool {
	if value == "" {
		return false
	}
	unit := value[len(value)-1]
	if unit != 's' && unit != 'm' && unit != 'h' && unit != 'd' {
		return false
	}
	_, err := strconv.ParseFloat(value[:len(value)-1], 64)
	if err != nil {
		return false
	}
	return strings.Contains(key, "duration") || strings.HasSuffix(key, "-after") ||
		strings.HasSuffix(key, "-timeout") || strings.HasSuffix(key, "-delay") ||
		strings.HasSuffix(key, "-interval")
}

// settingFromDoc merges Ghostty's documentation with the studio's schema.
func settingFromDoc(doc settingDoc, curated map[string]curatedSetting) Setting {
	setting := Setting{
		Key:         doc.key,
		Name:        humanizeKey(doc.key),
		Description: doc.description,
		Group:       GroupReference,
		Kind:        "reference",
		Reload:      doc.reload,
	}
	if len(doc.defaults) > 0 {
		setting.Default = doc.defaults[0]
	}
	if len(doc.defaults) > 1 {
		setting.Description = strings.TrimSpace(setting.Description +
			fmt.Sprintf(" Ghostty defines %d default values for this repeated key; the first is shown as the default and the current value lists them in order.", len(doc.defaults)))
	}
	if entry, ok := curated[doc.key]; ok {
		setting.Group = entry.group
		setting.Kind = entry.kind
		setting.Options = append(setting.Options, entry.options...)
	}
	if len(doc.options) > 0 {
		setting.Options = doc.options
	}
	if setting.Description == "" {
		if entry, ok := curated[doc.key]; ok {
			setting.Description = entry.description
		} else {
			setting.Description = "Documented by the installed Ghostty build; no description available."
		}
	}
	return setting
}

// settingFromCurated builds a setting purely from the studio's schema, used
// when no Ghostty executable is available to supply documentation.
func settingFromCurated(entry curatedSetting) Setting {
	return Setting{
		Key:         entry.key,
		Name:        humanizeKey(entry.key),
		Description: entry.description + " (Ghostty documentation unavailable: executable not found.)",
		Group:       entry.group,
		Kind:        entry.kind,
		Reload:      entry.reload,
		Options:     append([]string(nil), entry.options...),
	}
}

// humanizeKey turns a configuration key into a readable name.
func humanizeKey(key string) string {
	words := strings.Split(key, "-")
	for i, word := range words {
		if replacement, ok := keyWordNames[word]; ok {
			words[i] = replacement
			continue
		}
		if word == "" {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

// keyWordNames fixes the capitalisation of words that are acronyms or product
// names in Ghostty's own vocabulary.
var keyWordNames = map[string]string{
	"macos":   "macOS",
	"gtk":     "GTK",
	"xdg":     "XDG",
	"url":     "URL",
	"ansi":    "ANSI",
	"ssh":     "SSH",
	"cwd":     "CWD",
	"lt":      "LT",
	"glsl":    "GLSL",
	"opengl":  "OpenGL",
	"csi":     "CSI",
	"utf8":    "UTF-8",
	"osc":     "OSC",
	"id":      "ID",
	"os":      "OS",
	"i":       "I",
	"x":       "X",
	"y":       "Y",
	"keybind": "Keybind",
}
