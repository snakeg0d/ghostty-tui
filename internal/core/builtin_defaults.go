package core

// builtinDefaults holds the default values of the curated schema taken from an
// installed Ghostty 1.3.1 build. It is only used when no Ghostty executable is
// available to report the real defaults, so the studio can still show and stage
// values honestly instead of guessing at render time.
var builtinDefaults = map[string]string{
	"font-family":                       "",
	"font-size":                         "13",
	"font-feature":                      "",
	"font-variation":                    "",
	"font-style":                        "default",
	"font-thicken":                      "false",
	"font-thicken-strength":             "255",
	"adjust-cell-height":                "",
	"adjust-cell-width":                 "",
	"adjust-font-baseline":              "",
	"window-padding-x":                  "2",
	"window-padding-y":                  "2",
	"window-padding-balance":            "false",
	"background-opacity":                "1",
	"background-blur":                   "false",
	"background-image":                  "",
	"background-image-opacity":          "1",
	"window-decoration":                 "auto",
	"window-save-state":                 "default",
	"window-width":                      "0",
	"window-height":                     "0",
	"macos-titlebar-style":              "transparent",
	"cursor-style":                      "block",
	"cursor-style-blink":                "",
	"cursor-opacity":                    "1",
	"cursor-color":                      "",
	"cursor-text":                       "",
	"mouse-hide-while-typing":           "false",
	"scrollback-limit":                  "10000000",
	"confirm-close-surface":             "true",
	"copy-on-select":                    "true",
	"shell-integration":                 "detect",
	"shell-integration-features":        "cursor,no-sudo,title,no-ssh-env,no-ssh-terminfo,path",
	"quick-terminal-position":           "top",
	"quick-terminal-autohide":           "true",
	"quick-terminal-screen":             "main",
	"quick-terminal-animation-duration": "0.2",
	"desktop-notifications":             "true",
	"notify-on-command-finish":          "never",
	"notify-on-command-finish-after":    "5s",
	"window-inherit-working-directory":  "true",
	"focus-follows-mouse":               "false",
	"minimum-contrast":                  "1",
	"background":                        "#282c34",
	"foreground":                        "#ffffff",
	"selection-background":              "",
	"selection-foreground":              "",
	"theme":                             "",
	"palette":                           "0=#1d1f21",
	"custom-shader":                     "",
	"custom-shader-animation":           "true",
	"keybind":                           "super+shift+,=reload_config",
	"key-remap":                         "",
	"macos-option-as-alt":               "",
}

// builtinDefaultValues converts the map into Values.
func builtinDefaultValues() Values {
	out := make(Values, len(builtinDefaults))
	for key, value := range builtinDefaults {
		if value == "" {
			out[key] = []string{""}
			continue
		}
		out[key] = []string{value}
	}
	return out
}
