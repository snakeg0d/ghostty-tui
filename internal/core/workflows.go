package core

import (
	"fmt"
	"strings"
)

// workflow is a curated, typed patch for a common Ghostty workflow.
type workflow struct {
	slug        string
	name        string
	description string
	tags        []string
	// bindings lists trigger=action pairs the workflow adds, used to report
	// conflicts with the user's existing bindings.
	bindings []string
	values   Values
}

// workflows is the curated workflow catalog. Every patch is a plain typed
// configuration change; none of them run commands.
var workflows = []workflow{
	{
		slug:        "quick-terminal",
		name:        "Quick terminal",
		description: "A drop-down terminal on Cmd+`. Staged as a global keybinding plus the drop-down geometry.",
		tags:        []string{"keybind", "quick-terminal"},
		bindings:    []string{"global:super+backquote=toggle_quick_terminal"},
		values: Values{
			"quick-terminal-position":           []string{"top"},
			"quick-terminal-autohide":           []string{"true"},
			"quick-terminal-screen":             []string{"mouse"},
			"quick-terminal-animation-duration": []string{"0.15"},
		},
	},
	{
		slug:        "command-notifications",
		name:        "Command notifications",
		description: "Desktop notifications when a long command finishes while the terminal is unfocused.",
		tags:        []string{"notifications"},
		values: Values{
			"desktop-notifications":          []string{"true"},
			"notify-on-command-finish":       []string{"unfocused"},
			"notify-on-command-finish-after": []string{"5s"},
		},
	},
	{
		slug:        "inherit-working-directory",
		name:        "Inherit working directory",
		description: "New windows, tabs and splits open where the current surface is, instead of the home directory.",
		tags:        []string{"workflow"},
		values: Values{
			"window-inherit-working-directory": []string{"true"},
			"tab-inherit-working-directory":    []string{"true"},
			"split-inherit-working-directory":  []string{"true"},
			"window-inherit-font-size":         []string{"true"},
		},
	},
	{
		slug:        "option-as-alt",
		name:        "Option key as Alt",
		description: "Send the Option key as Alt so shell and editor meta bindings work on macOS.",
		tags:        []string{"macos"},
		values: Values{
			"macos-option-as-alt": []string{"true"},
		},
	},
	{
		slug:        "scrollback-search",
		name:        "Scrollback search",
		description: "Bind Cmd+F to Ghostty's scrollback search and keep a large scrollback so history is searchable.",
		tags:        []string{"keybind", "search"},
		bindings:    []string{"super+f=start_search"},
		values: Values{
			"scrollback-limit": []string{"10000000"},
		},
	},
}

// workflowChoices builds the workflow catalog, reporting conflicts with the
// user's existing bindings rather than silently overriding them.
//
// A workflow that binds keys stages exactly the bindings the studio already
// manages, minus the triggers it takes over, plus its own bindings last. The
// user's own bindings are never rewritten or cleared: their file keeps every
// line, and Ghostty's documented rule that "duplicate triggers will overwrite
// previously set values" makes the workflow's binding win for the triggers it
// requests. Emitting a reset here would erase the user's bindings and Ghostty's
// defaults, because the managed include is the last file Ghostty loads.
func workflowChoices(current, managed Values) []Choice {
	choices := make([]Choice, 0, len(workflows))
	for _, item := range workflows {
		description := item.description
		tags := append([]string(nil), item.tags...)
		if conflicts := bindingConflicts(item.bindings, current); len(conflicts) > 0 {
			description += " Conflict: " + strings.Join(conflicts, "; ") + "."
			tags = append(tags, "conflict")
		}
		values := item.values.Clone()
		if len(item.bindings) > 0 {
			values["keybind"] = mergedBindings(managed["keybind"], item.bindings)
		}
		choices = append(choices, Choice{
			ID:          "workflow:" + item.slug,
			Name:        item.name,
			Description: description,
			Source:      "Ghostty Studio",
			Category:    "workflow",
			Tags:        tags,
			Values:      values,
		})
	}
	return choices
}

// mergedBindings rebuilds the managed binding list: the bindings the studio
// already manages, minus the triggers this workflow takes over, followed by the
// workflow's bindings. Nothing else is listed, so the user's configuration is
// left exactly as they wrote it and stays editable.
func mergedBindings(existing, add []string) []string {
	taken := make(map[string]bool, len(add))
	for _, binding := range add {
		if trigger, _ := splitBinding(binding); trigger != "" {
			taken[trigger] = true
		}
	}
	out := make([]string, 0, len(existing)+len(add))
	for _, binding := range existing {
		trigger, _ := splitBinding(binding)
		if trigger == "" || taken[trigger] {
			continue
		}
		out = append(out, binding)
	}
	return append(out, add...)
}

// bindingConflicts explains which of the requested bindings already exist with
// a different action.
func bindingConflicts(bindings []string, current Values) []string {
	if len(bindings) == 0 {
		return nil
	}
	existing := make(map[string]string)
	for _, value := range current["keybind"] {
		trigger, action := splitBinding(value)
		if trigger == "" {
			continue
		}
		existing[trigger] = action
	}
	var conflicts []string
	for _, want := range bindings {
		trigger, action := splitBinding(want)
		if trigger == "" {
			continue
		}
		if have, ok := existing[trigger]; ok && have != action {
			conflicts = append(conflicts, fmt.Sprintf("%s is bound to %s", trigger, have))
		}
	}
	return conflicts
}

// splitBinding splits a keybind value into trigger and action. Sequences use
// '>' separators and a leading global: marker; both are preserved in the
// trigger so conflicts are detected on the exact binding.
func splitBinding(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}
	idx := strings.LastIndexByte(value, '=')
	if idx <= 0 {
		return "", ""
	}
	return value[:idx], value[idx+1:]
}
