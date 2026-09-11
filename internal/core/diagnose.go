package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Diagnose returns the health findings gathered by the last refresh.
func (e *Engine) Diagnose() []Diagnostic {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]Diagnostic(nil), e.diagnostics...)
}

// diagnostic is a helper for building one finding.
func diagnostic(level, title, detail string) Diagnostic {
	return Diagnostic{Level: level, Title: title, Detail: detail}
}

// buildDiagnostics assembles the health view.
func (e *Engine) buildDiagnostics(ctx context.Context) {
	for _, warning := range e.warnings {
		e.diagnostics = append(e.diagnostics, diagnostic("warn", "Warning", warning))
	}

	version := e.version
	if version == "" {
		version = "unknown"
	}
	executable := e.native.bin
	if executable == "" {
		executable = "not found"
	}
	e.diagnostics = append(e.diagnostics, diagnostic("info", "Ghostty", fmt.Sprintf("version %s · %s", version, executable)))

	if e.configPath != "" {
		detail := e.configPath
		if info, err := os.Stat(e.configPath); err == nil {
			detail = fmt.Sprintf("%s · %s", detail, formatBytes(info.Size()))
		} else if os.IsNotExist(err) {
			detail += " · does not exist yet"
		}
		e.diagnostics = append(e.diagnostics, diagnostic("info", "Configuration file", detail))
	}
	for i, source := range e.sources {
		detail := source
		if info, err := os.Stat(source); err == nil {
			detail = fmt.Sprintf("%s · %s", detail, formatBytes(info.Size()))
		}
		if samePath(source, e.managedFile) {
			detail += " · Ghostty Studio managed include"
		}
		e.diagnostics = append(e.diagnostics, diagnostic("info", fmt.Sprintf("Source %d of %d", i+1, len(e.sources)), detail))
	}
	if e.managedFile != "" {
		detail := e.managedFile
		if _, err := os.Stat(e.managedFile); err == nil {
			detail += " · present"
			if e.graph != nil && e.graph.find(e.managedFile) == nil {
				detail += " · not included by any configuration file, so its values are inactive"
			}
		} else {
			detail += " · not written yet"
		}
		e.diagnostics = append(e.diagnostics, diagnostic("info", "Managed include", detail))
	}

	dark, light, unknown := 0, 0, 0
	for _, theme := range e.themes {
		switch {
		case theme.palette.Background == "":
			unknown++
		case theme.light:
			light++
		default:
			dark++
		}
	}
	e.diagnostics = append(e.diagnostics, diagnostic("info", "Theme catalog",
		fmt.Sprintf("%d themes (%d dark, %d light, %d without colors) from %d directories", len(e.themes), dark, light, unknown, len(themeFileDirs(e.paths.themeDirs())))))

	if e.native.available() {
		listing := e.native.listThemes(ctx)
		if listing.Degraded {
			e.diagnostics = append(e.diagnostics, diagnostic("warn", "Native theme listing is unreliable",
				fmt.Sprintf("ghostty +list-themes returned %d names and failed (%v). Ghostty 1.3.1 exits non-zero on this build, so the catalog uses the theme files on disk instead.", len(listing.Names), listing.Err)))
		}
	}

	e.diagnostics = append(e.diagnostics, diagnostic("info", "Font catalog",
		fmt.Sprintf("%d installed families, %d beyond ghostty +list-fonts", len(e.fonts), len(e.probedFamilies))))
	if len(e.probedFamilies) > 0 {
		e.diagnostics = append(e.diagnostics, diagnostic("info", "Fonts missing from the native listing",
			fmt.Sprintf("ghostty +list-fonts does not report %s, so Ghostty Studio found them with targeted --family probes: %s",
				plural(len(e.probedFamilies), "an installed family", "installed families"), strings.Join(sortedNames(e.probedFamilies), ", "))))
	}
	if len(e.shaders) == 0 {
		e.diagnostics = append(e.diagnostics, diagnostic("warn", "No shader assets found",
			"Searched "+joinPaths(themeFileDirs(e.paths.shaderDirs()))+". Effects will only offer None until shader assets are installed."))
	} else {
		e.diagnostics = append(e.diagnostics, diagnostic("info", "Effect catalog",
			fmt.Sprintf("%d GLSL files. Note: GHOSTTY validates the path, not the shader; GLSL compilation is only proven when the renderer draws it.", len(e.shaders))))
	}

	library := e.looksDir()
	detail := fmt.Sprintf("%d saved Look(s) in %s", len(e.savedLooks), library)
	if len(e.savedLooks) == 0 {
		detail += " · none saved yet"
	}
	e.diagnostics = append(e.diagnostics, diagnostic("info", "Look library", detail))

	if e.configPath != "" && e.native.available() {
		if info, err := os.Stat(e.configPath); err == nil && info.Mode().IsRegular() {
			diagnostics, err := e.native.validate(ctx, e.configPath)
			e.collectNativeWarnings()
			switch {
			case err != nil:
				e.diagnostics = append(e.diagnostics, diagnostic("error", "Configuration validation failed", fmt.Sprintf("%s could not be validated: %v", e.configPath, err)))
			case len(diagnostics) > 0:
				e.diagnostics = append([]Diagnostic{diagnostic("error", "Configuration validation failed", fmt.Sprintf("ghostty +validate-config reported:\n%s", strings.Join(diagnostics, "\n")))}, e.diagnostics...)
			default:
				e.diagnostics = append(e.diagnostics, diagnostic("info", "Configuration validation", fmt.Sprintf("%s passed ghostty +validate-config", e.configPath)))
			}
		}
	}

	if journal, err := e.readJournal(); err == nil {
		detail := fmt.Sprintf("%d applied change(s) can be undone", len(journal.Entries))
		if len(journal.Entries) > 0 {
			last := journal.Entries[len(journal.Entries)-1]
			detail += fmt.Sprintf(" · most recent %s at %s", labelOr(last.Label, "change"), last.Time)
			if changed := changedSince(last.Files); len(changed) > 0 {
				detail += fmt.Sprintf(" · edited outside the studio since: %s (undo will refuse)", strings.Join(changed, ", "))
			}
		}
		e.diagnostics = append(e.diagnostics, diagnostic("info", "Undo stack", detail))
	} else {
		e.diagnostics = append(e.diagnostics, diagnostic("warn", "Undo stack unreadable", err.Error()))
	}
	e.diagnostics = append(e.diagnostics, diagnostic("info", "Backups", e.backupGuidance()))

	if e.paths.stateDir != "" {
		if entries, err := os.ReadDir(filepath.Join(e.paths.stateDir, "preview")); err == nil && len(entries) > 0 {
			e.diagnostics = append(e.diagnostics, diagnostic("info", "Preview files",
				fmt.Sprintf("%d specimen configuration(s) are kept in %s for inspection; delete them when you are done.", len(entries), filepath.Join(e.paths.stateDir, "preview"))))
		}
	}

	if e.configPath == "" {
		e.diagnostics = append(e.diagnostics, diagnostic("error", "No configuration location", "Ghostty Studio could not determine where to write configuration on this machine."))
	}
}

// labelOr substitutes a readable label.
func labelOr(label, fallback string) string {
	if strings.TrimSpace(label) == "" {
		return fallback
	}
	return label
}

// joinPaths renders a path list for a diagnostic.
func joinPaths(dirs []themeDir) string {
	if len(dirs) == 0 {
		return "no directories"
	}
	paths := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		paths = append(paths, dir.path)
	}
	return strings.Join(paths, ", ")
}
