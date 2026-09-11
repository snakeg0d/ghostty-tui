package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Engine is the Ghostty Studio configuration engine. It is safe for concurrent
// use: every exported method takes the engine lock, and catalogs returned to
// callers are deep copies that cannot race with a refresh.
type Engine struct {
	opts     Options
	paths    paths
	runner   Runner
	native   *native
	brew     string
	exePath  string
	curated  map[string]curatedSetting
	groupIdx map[string]int

	mu sync.Mutex

	// resolved configuration
	roots       []string
	configPath  string
	managedFile string
	graph       *sourceGraph
	revision    string

	defaults Values
	docs     []settingDoc
	settings []Setting
	current  Values

	themes  []themeEntry
	fonts   []installedFont
	shaders []shaderFile
	// probedFamilies are installed families that ghostty +list-fonts does not
	// report without a targeted --family probe.
	probedFamilies  []string
	themeChoices    []Choice
	fontChoices     []Choice
	discoverChoices []Choice
	effectChoices   []Choice
	lookChoices     []Choice
	workflowChoices []Choice
	// savedLooks is the user's Look library, validated at load time.
	savedLooks []Choice

	favorites []string
	sources   []string

	version      string
	warnings     []string
	diagnostics  []Diagnostic
	previewGi    int
	lastValidate []string
}

// Open builds an engine for the process environment.
func Open(opts Options) (*Engine, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return open(opts, osRunner{}, os.Getenv, home, exec.LookPath)
}

// open is the injectable constructor used by tests.
func open(opts Options, runner Runner, getenv func(string) string, home string, lookPath func(string) (string, error)) (*Engine, error) {
	if runner == nil {
		return nil, errors.New("core: nil command runner")
	}
	p := resolvePaths(opts, getenv, home, lookPath)
	if p.configOverride != "" && !filepath.IsAbs(p.configOverride) {
		if abs, err := filepath.Abs(p.configOverride); err == nil {
			p.configOverride = abs
		}
	}
	bin, err := discoverGhostty(p)
	if err != nil {
		return nil, err
	}
	p.ghosttyBin = bin
	e := &Engine{
		opts:     opts,
		paths:    p,
		runner:   runner,
		native:   &native{runner: runner},
		brew:     discoverBrew(p),
		curated:  curatedIndex(),
		groupIdx: groupIndex(),
	}
	if exe, err := os.Executable(); err == nil {
		e.exePath = exe
	}
	e.paths.assetDir = discoverAssetDir(p, e.exePath)
	if err := e.Refresh(); err != nil {
		return nil, err
	}
	return e, nil
}

// State returns a snapshot of the discovered configuration state.
func (e *Engine) State() State {
	e.mu.Lock()
	defer e.mu.Unlock()
	return State{
		Current:     e.current.Clone(),
		ConfigPath:  e.configPath,
		GhosttyPath: e.native.bin,
		Version:     e.version,
		Sources:     append([]string(nil), e.sources...),
		Warnings:    append([]string(nil), e.warnings...),
		Revision:    e.revision,
	}
}

// Refresh re-reads the configuration graph, the native catalogs and the
// favorites file. It never writes anything.
func (e *Engine) Refresh() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.refresh(context.Background())
}

// refresh is the locked implementation behind Refresh.
func (e *Engine) refresh(ctx context.Context) error {
	e.warnings = nil
	e.diagnostics = nil
	if err := e.resolveConfig(); err != nil {
		return err
	}
	e.loadNative(ctx)
	e.loadThemes(ctx)
	// loadCurrent runs before the font catalog because the targeted font probes
	// follow the families the user's configuration actually asks for.
	e.loadCurrent(ctx)
	e.loadFonts(ctx)
	e.loadSettings(ctx)
	e.loadFavorites()
	e.loadSavedLooks()
	e.buildCatalogs()
	e.buildDiagnostics(ctx)
	return nil
}

// resolveConfig discovers which configuration files the studio manages.
func (e *Engine) resolveConfig() error {
	p := e.paths
	if p.configExplicit {
		e.configPath = p.configOverride
		e.managedFile = managedPath(e.configPath)
		info, err := os.Stat(e.configPath)
		switch {
		case err == nil && info.IsDir():
			return fmt.Errorf("configuration path %s is a directory", e.configPath)
		case err == nil:
			e.roots = []string{e.configPath}
		case os.IsNotExist(err):
			e.roots = nil
			e.graph = &sourceGraph{}
			e.revision = e.graph.revision()
			e.sources = nil
			e.warn("configuration file %s does not exist yet; Ghostty Studio will create it when you apply a change", e.configPath)
			return nil
		default:
			return fmt.Errorf("configuration path %s: %w", e.configPath, err)
		}
	} else {
		files, shadowed := p.existingConfigFiles()
		for _, path := range shadowed {
			e.warn("%s is present but Ghostty reads config.ghostty in the same directory, so this file has no effect", path)
		}
		if len(files) == 0 {
			e.configPath = p.defaultConfigPath()
			e.managedFile = managedPath(e.configPath)
			e.roots = nil
			e.graph = &sourceGraph{}
			e.revision = e.graph.revision()
			e.sources = nil
			e.warn("no Ghostty configuration file was found; Ghostty Studio will create %s when you apply a change", e.configPath)
			return nil
		}
		e.roots = files
		// The winning file is the last one Ghostty loads.
		e.configPath = files[len(files)-1]
		e.managedFile = managedPath(e.configPath)
		if len(files) > 1 {
			e.warn("Ghostty reads %d configuration files on this machine; %s wins for single-value settings", len(files), e.configPath)
		}
	}

	graph, err := resolveGraph(e.roots)
	if err != nil {
		return err
	}
	e.graph = graph
	e.revision = graph.revision()
	e.sources = graph.paths()
	if graph.unsafe != "" {
		e.warn("the configuration include graph cannot be rewritten safely: %s", graph.unsafe)
	}
	for _, missing := range graph.missing {
		if missing.optional {
			e.warn("optional include %s (from %s) does not exist", missing.path, missing.from)
			continue
		}
		e.warn("include %s (from %s) does not exist; Ghostty will fail to load this configuration graph", missing.path, missing.from)
	}
	if e.managedFile != "" && graph.find(e.managedFile) != nil && !samePath(graph.last().path, e.managedFile) {
		e.warn("the Ghostty Studio include %s is not the last configuration file loaded, so later files can mask staged values", e.managedFile)
	}
	return nil
}

// loadNative discovers the Ghostty executable and reads its defaults.
func (e *Engine) loadNative(ctx context.Context) {
	e.native = &native{bin: e.paths.ghosttyBin, runner: e.runner}
	if !e.native.available() {
		e.defaults = builtinDefaultValues()
		e.warn("the ghostty executable was not found; native validation, font discovery and previews are unavailable")
		e.diagnostics = append(e.diagnostics, Diagnostic{
			Level:  "error",
			Title:  "Ghostty executable not found",
			Detail: "Searched GHOSTTY_BIN, GHOSTTY_TUI_GHOSTTY_PATH, PATH and " + bundleBinary + ". Browsing still works; applying changes and native previews need the executable.",
		})
		return
	}
	e.version = e.native.detectVersion(ctx)
	defaults, err := e.native.defaults(ctx)
	if err != nil {
		e.defaults = builtinDefaultValues()
		e.warn("Ghostty defaults could not be read (%v); using the built-in defaults of Ghostty 1.3.1", err)
	} else {
		e.defaults = normalizeDefaults(defaults)
	}
	e.collectNativeWarnings()
}

// collectNativeWarnings folds the native layer's warnings into the state.
func (e *Engine) collectNativeWarnings() {
	for _, warning := range e.native.takeWarnings() {
		e.warnings = append(e.warnings, warning)
	}
}

// loadThemes reads theme files from the user and bundle directories. The native
// listing is consulted only as a cross-check: Ghostty 1.3.1 exits non-zero with
// a truncated list on macOS, so the files on disk are authoritative.
func (e *Engine) loadThemes(ctx context.Context) {
	entries, warnings := loadThemes(themeFileDirs(e.paths.themeDirs()))
	sortThemes(entries)
	e.themes = entries
	e.warnings = append(e.warnings, warnings...)
	if !e.native.available() {
		return
	}
	listing := e.native.listThemes(ctx)
	if listing.Degraded {
		e.warn("ghostty +list-themes returned %d names and failed; the theme catalog was built from the theme files on disk instead", len(listing.Names))
	}
}

// loadFonts reads the installed font families from the native build.
func (e *Engine) loadFonts(ctx context.Context) {
	if !e.native.available() {
		e.fonts = nil
		e.warn("installed fonts are listed by ghostty +list-fonts, which is unavailable")
		return
	}
	families, err := e.native.listFonts(ctx)
	if err != nil {
		e.fonts = nil
		e.warn("ghostty +list-fonts failed (%v); the font catalog is unavailable", err)
		e.collectNativeWarnings()
		return
	}
	installed := make([]installedFont, 0, len(families))
	for _, family := range families {
		installed = append(installed, installedFont{name: family.Name, styles: family.Styles})
	}
	e.collectNativeWarnings()

	// The unfiltered listing is incomplete: families installed by hand can be
	// missing from it entirely. Probe the curated family names and the families
	// the user's configuration asks for, and add whatever the native build
	// reports so those fonts stay selectable.
	e.probedFamilies = nil
	listed := installed
	probes := fontProbeNames(listed, e.current)
	if len(probes) > 0 {
		probed, failures := e.native.probeFamilies(ctx, probes)
		for _, family := range probed {
			installed = append(installed, installedFont{name: family.Name, styles: family.Styles})
			if !familyMatches(listed, family.Name) {
				e.probedFamilies = append(e.probedFamilies, family.Name)
			}
		}
		if len(failures) > 0 {
			e.warn("targeted font probes failed for %s; some installed families may be missing from the catalog", strings.Join(failures, ", "))
		}
	}
	sortInstalledFonts(installed)
	e.fonts = installed

	shaders, shaderWarnings := loadShaders(themeFileDirs(e.paths.shaderDirs()))
	e.shaders = shaders
	e.warnings = append(e.warnings, shaderWarnings...)
}

// loadSettings reads the exhaustive configuration reference. The docs query is
// deliberately the last native call so a slow machine cannot delay themes.
func (e *Engine) loadSettings(ctx context.Context) {
	if !e.native.available() {
		e.settings = settingsFromCurated()
		return
	}
	docs, err := e.native.docs(ctx)
	if err != nil {
		e.settings = settingsFromCurated()
		e.warn("Ghostty configuration documentation could not be read (%v); showing the curated schema only", err)
		e.collectNativeWarnings()
		return
	}
	e.docs = parseDocDump(docs)
	e.settings = settingsFromDocs(e.docs, e.curated)
	e.collectNativeWarnings()
}

// loadCurrent computes the effective configuration.
func (e *Engine) loadCurrent(ctx context.Context) {
	e.current = e.evaluateGraph()
	if e.paths.configExplicit || !e.native.available() {
		return
	}
	changes, err := e.native.changes(ctx)
	if err != nil {
		e.warn("ghostty +show-config could not report the effective configuration (%v); computed locally from the discovered files", err)
		e.collectNativeWarnings()
		return
	}
	e.collectNativeWarnings()
	e.current = overlayChanges(normalizeDefaults(e.defaults), normalizeChanges(changes))
}

// evaluateGraph resolves the include graph without the managed include, then
// folds the managed values on top.
func (e *Engine) evaluateGraph() Values {
	base := e.graphValues()
	return e.applyManaged(base, e.managedState())
}

// graphValues evaluates the discovered files only.
func (e *Engine) graphValues() Values {
	var files []*configFile
	for _, file := range e.graph.files {
		if e.managedFile != "" && samePath(file.path, e.managedFile) {
			continue
		}
		files = append(files, file)
	}
	return evaluate(e.defaults, files, themeIndex(e.themes))
}

// applyManaged folds managed directives onto a base value set.
func (e *Engine) applyManaged(base, managed Values) Values {
	out := base.Clone()
	if out == nil {
		out = make(Values, len(managed))
	}
	for _, key := range managed.Keys() {
		for _, value := range managed[key] {
			applyDirective(out, e.defaults, key, value)
		}
	}
	return out
}

// valuesWith reports the effective configuration with a candidate managed value
// set. It is the base for diffs and for the change preview.
func (e *Engine) valuesWith(managed Values) Values {
	return e.applyManaged(e.graphValues(), managed)
}

// loadFavorites reads the persisted favorite identifiers.
func (e *Engine) loadFavorites() {
	favorites, err := e.readFavorites()
	if err != nil {
		e.warn("favorites could not be read: %v", err)
		return
	}
	e.favorites = favorites
}

// loadSavedLooks reads the user's Look library.
func (e *Engine) loadSavedLooks() {
	choices, warnings := e.readLookLibrary()
	e.savedLooks = choices
	e.warnings = append(e.warnings, warnings...)
}

// reloadLookLibrary re-reads the library and rebuilds the Look catalog, so a
// Look saved from this session is immediately selectable.
func (e *Engine) reloadLookLibrary() {
	e.loadSavedLooks()
	e.lookChoices = lookChoices(e.themes, e.fonts, e.current, e.savedLooks)
}

// buildCatalogs rebuilds every catalog from the current state.
func (e *Engine) buildCatalogs() {
	e.themeChoices = themeChoices(e.themes)
	e.fontChoices = fontChoices(e.fonts, e.current)
	e.discoverChoices = discoverChoices(e.fonts)
	e.effectChoices = effectChoices(e.shaders)
	e.lookChoices = lookChoices(e.themes, e.fonts, e.current, e.savedLooks)
	e.workflowChoices = workflowChoices(e.current, e.managedState())
}

// Catalog returns a deep copy of one catalog section.
func (e *Engine) Catalog(section string) []Choice {
	e.mu.Lock()
	defer e.mu.Unlock()
	switch strings.ToLower(strings.TrimSpace(section)) {
	case SectionThemes:
		return copyChoices(e.themeChoices)
	case SectionFonts:
		return copyChoices(e.fontChoices)
	case SectionDiscover:
		return copyChoices(e.discoverChoices)
	case SectionLooks:
		return copyChoices(e.lookChoices)
	case SectionEffects:
		return copyChoices(e.effectChoices)
	case SectionWorkflows:
		return copyChoices(e.workflowChoices)
	default:
		return nil
	}
}

// Settings returns the configuration reference in group order.
func (e *Engine) Settings() []Setting {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]Setting, 0, len(e.settings))
	for _, setting := range e.settings {
		clone := setting
		clone.Options = append([]string(nil), setting.Options...)
		out = append(out, clone)
	}
	return out
}

// Favorites returns the persisted favorite identifiers.
func (e *Engine) Favorites() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.favorites...)
}

// ToggleFavorite adds or removes an identifier from the favorites list.
func (e *Engine) ToggleFavorite(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("cannot favorite an empty identifier")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	next := make([]string, 0, len(e.favorites)+1)
	found := false
	for _, existing := range e.favorites {
		if existing == id {
			found = true
			continue
		}
		next = append(next, existing)
	}
	if !found {
		next = append(next, id)
	}
	sort.Strings(next)
	if err := e.writeFavorites(next); err != nil {
		return err
	}
	e.favorites = next
	return nil
}

// Diff renders the effect of a staged change on the effective configuration.
func (e *Engine) Diff(change Change) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.diff(change)
}

// diff is the locked implementation behind Diff.
func (e *Engine) diff(change Change) string {
	if err := checkValues(change.Values); err != nil {
		return "Cannot stage this change: " + err.Error() + "\n"
	}
	current := e.managedState()
	candidate := Patch(current, change.Values)
	before := e.applyManaged(e.graphValues(), current)
	after := e.applyManaged(e.graphValues(), candidate)

	keys := make([]string, 0, len(before)+len(after))
	seen := make(map[string]bool, len(before)+len(after))
	for _, key := range append(append(change.Values.Keys(), before.Keys()...), after.Keys()...) {
		if seen[key] {
			continue
		}
		seen[key] = true
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString(diffValues(before, after, keys))
	for _, warning := range e.changeWarnings(change, before) {
		b.WriteString("! " + warning + "\n")
	}
	if change.Revision != "" && change.Revision != e.revision {
		b.WriteString("! this change was staged against an older revision of the configuration; stage it again before applying\n")
	}
	return b.String()
}

// changeWarnings reports conflicts a patch would create.
func (e *Engine) changeWarnings(change Change, before Values) []string {
	var warnings []string
	if bindings, ok := change.Values["keybind"]; ok {
		var wanted []string
		for _, binding := range bindings {
			if binding != "" {
				wanted = append(wanted, binding)
			}
		}
		if conflicts := bindingConflicts(wanted, before); len(conflicts) > 0 {
			warnings = append(warnings, fmt.Sprintf("this patch rebinds a key you already use: %s (later bindings win, so the staged value takes effect)", strings.Join(conflicts, "; ")))
		}
	}
	var restart []string
	for _, key := range change.Values.Keys() {
		if setting, ok := e.settingByKey(key); ok && setting.Reload == "restart" {
			restart = append(restart, key)
		}
	}
	if len(restart) > 0 {
		warnings = append(warnings, "these settings need a full Ghostty restart, not just a reload: "+strings.Join(restart, ", "))
	}
	return warnings
}

// settingByKey finds one setting in the loaded reference.
func (e *Engine) settingByKey(key string) (Setting, bool) {
	for _, setting := range e.settings {
		if setting.Key == key {
			return setting, true
		}
	}
	return Setting{}, false
}

// Apply validates and writes a staged change.
func (e *Engine) Apply(change Change) (Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.apply(context.Background(), change)
}

// apply is the locked implementation behind Apply.
func (e *Engine) apply(ctx context.Context, change Change) (Result, error) {
	if err := checkValues(change.Values); err != nil {
		return Result{}, err
	}
	if len(change.Values) == 0 {
		return Result{Message: "Nothing was staged, so nothing was written."}, nil
	}
	if err := e.checkKnownKeys(change.Values); err != nil {
		return Result{}, err
	}
	if err := checkPathValues(change.Values); err != nil {
		return Result{}, err
	}
	// Re-read the graph so a stale revision is caught even when the file
	// changed after the last Refresh.
	graph, err := resolveGraph(e.roots)
	if err != nil {
		return Result{}, err
	}
	revision := graph.revision()
	if change.Revision != "" && change.Revision != revision {
		return Result{}, fmt.Errorf("this change was staged against revision %s but the configuration is now at %s; stage the change again", shortHash(change.Revision), shortHash(revision))
	}
	if graph.unsafe != "" {
		return Result{}, fmt.Errorf("refusing to write: %s", graph.unsafe)
	}
	currentManaged := e.managedState()
	candidate := Patch(currentManaged, change.Values)
	if Equal(candidate, currentManaged) {
		return Result{Message: "These values already match what Ghostty Studio manages; nothing was written."}, nil
	}

	target, targetContent, managedContent, err := e.prepareWrite(graph, change.Values, candidate)
	if err != nil {
		return Result{}, err
	}
	if err := e.validateCandidate(ctx, graph, managedContent); err != nil {
		return Result{}, err
	}

	txn := &transaction{id: newTransactionID(), label: change.Label}
	isRemoval := len(candidate) == 0
	if e.managedFile != "" {
		if err := txn.stage(e, e.managedFile); err != nil {
			return Result{}, err
		}
	}
	if target != "" {
		if err := txn.stage(e, target); err != nil {
			return Result{}, err
		}
	}
	if err := txn.backup(e); err != nil {
		return Result{}, err
	}
	if len(txn.files) == 0 {
		return Result{}, errors.New("refusing to write: no write target could be determined")
	}

	if isRemoval {
		if err := os.Remove(e.managedFile); err != nil && !os.IsNotExist(err) {
			return Result{}, err
		}
	} else if err := writeFileAtomic(e.managedFile, managedContent, e.managedMode()); err != nil {
		return Result{}, fmt.Errorf("writing %s: %w", e.managedFile, err)
	}
	if target != "" && targetContent != nil {
		if err := writeFileAtomic(target, targetContent, e.targetMode(target)); err != nil {
			return Result{}, fmt.Errorf("writing %s: %w", target, err)
		}
	}

	updated, err := resolveGraph(e.roots)
	if err != nil {
		return Result{}, fmt.Errorf("re-reading the configuration graph after writing: %w", err)
	}
	if err := txn.commit(e, revision, updated.revision()); err != nil {
		return Result{}, err
	}

	display := e.displayPath(target)
	result := Result{
		Message:    e.applyMessage(change, candidate, isRemoval, display),
		BackupPath: txn.backupPathFor(),
	}
	if result.BackupPath != "" {
		result.Warnings = append(result.Warnings, "backups for this transaction are in "+result.BackupPath)
	}
	if err := e.refresh(ctx); err != nil {
		result.Warnings = append(result.Warnings, "the configuration was written but re-reading it failed: "+err.Error())
		return result, nil
	}
	result.Warnings = append(result.Warnings, e.changeWarnings(change, e.current)...)
	if updated.last() != nil && e.managedFile != "" && !samePath(updated.last().path, e.managedFile) {
		result.Warnings = append(result.Warnings, "the managed include is not the last file Ghostty loads, so later files may mask these values")
	}
	return result, nil
}

// applyMessage describes exactly what was written and what the user must do.
func (e *Engine) applyMessage(change Change, managed Values, removal bool, target string) string {
	keys := change.Values.Keys()
	noun := plural(len(keys), "setting", "settings")
	if removal {
		return fmt.Sprintf("Removed all %d managed %s. %s no longer contains a Ghostty Studio include, so your own configuration applies again. Reload Ghostty (Cmd+Shift+,) to pick the change up.", len(keys), noun, target)
	}
	message := fmt.Sprintf("Saved %d %s into %s and included it from %s. Reload Ghostty (Cmd+Shift+,) to pick them up.", len(keys), noun, e.displayPath(e.managedFile), target)
	return message
}

// managedMode returns the permission bits for the managed include file.
func (e *Engine) managedMode() os.FileMode {
	if info, err := os.Stat(e.managedFile); err == nil {
		return info.Mode().Perm()
	}
	return 0o644
}

// targetMode returns the permission bits for a configuration file.
func (e *Engine) targetMode(path string) os.FileMode {
	if info, err := os.Stat(path); err == nil {
		return info.Mode().Perm()
	}
	return 0o644
}

// displayPath renders a path with the home directory abbreviated.
func (e *Engine) displayPath(path string) string {
	if path == "" {
		return "(none)"
	}
	if e.paths.home != "" && strings.HasPrefix(path, e.paths.home+"/") {
		return "~" + strings.TrimPrefix(path, e.paths.home)
	}
	return path
}

// prepareWrite computes the managed file content and the include target.
//
// The include goes at the end of the winning root configuration file, which is
// the file Ghostty loads last among the top-level candidates. Ghostty processes a
// file's own directives before the files it includes and processes includes in
// declaration order, so a directive appended at the end of the last root loads
// after everything else in the graph, including the root's own includes. Files
// the user included are therefore never rewritten.
func (e *Engine) prepareWrite(graph *sourceGraph, patch Values, candidate Values) (target string, targetContent []byte, managedContent []byte, err error) {
	root := graph.find(e.configPath)
	if len(candidate) == 0 {
		// Everything managed was unset: drop the include and the file.
		if root == nil {
			return "", nil, nil, nil
		}
		content := stripManagedInclude(root.content, e.managedFile)
		if string(content) == string(root.content) {
			return "", nil, nil, nil
		}
		return root.real, content, nil, nil
	}
	managedContent = renderManagedFile(candidate, "")
	if root == nil {
		// First write: create the configuration file Ghostty reads.
		if e.configPath == "" {
			return "", nil, nil, errors.New("no configuration file location is available")
		}
		if err := ensureDir(filepath.Dir(e.configPath)); err != nil {
			return "", nil, nil, fmt.Errorf("creating %s: %w", filepath.Dir(e.configPath), err)
		}
		return e.configPath, withManagedInclude(nil, e.managedFile), managedContent, nil
	}
	content, changed := ensureManagedInclude(root.content, e.managedFile)
	if !changed {
		return "", nil, managedContent, nil
	}
	return root.real, content, managedContent, nil
}

// validateCandidate runs Ghostty's own validator over a faithful reproduction of
// the configuration graph that the write will produce.
func (e *Engine) validateCandidate(ctx context.Context, graph *sourceGraph, managedContent []byte) error {
	if !e.native.available() {
		return errors.New("cannot validate the staged configuration because the ghostty executable was not found; nothing was written")
	}
	dir, err := os.MkdirTemp("", "ghostty-studio-validate-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	var b strings.Builder
	b.WriteString("# Ghostty Studio validation harness. This file is validated with ghostty +validate-config and never installed.\n")
	for _, file := range graph.files {
		if e.managedFile != "" && samePath(file.path, e.managedFile) {
			continue
		}
		fmt.Fprintf(&b, "config-file = ?%s\n", file.path)
	}
	for _, missing := range graph.missing {
		if !missing.optional {
			// A required include that is missing must fail validation loudly,
			// exactly as it fails when Ghostty loads the real configuration.
			fmt.Fprintf(&b, "config-file = %s\n", missing.path)
		}
	}
	staged := filepath.Join(dir, managedFileName)
	if err := os.WriteFile(staged, managedContent, 0o600); err != nil {
		return err
	}
	fmt.Fprintf(&b, "config-file = ?%s\n", staged)
	wrapper := filepath.Join(dir, "candidate.ghostty")
	if err := os.WriteFile(wrapper, []byte(b.String()), 0o600); err != nil {
		return err
	}
	diagnostics, err := e.native.validate(ctx, wrapper)
	e.collectNativeWarnings()
	if err != nil {
		return fmt.Errorf("ghostty could not validate the staged configuration: %w", err)
	}
	if len(diagnostics) > 0 {
		e.lastValidate = diagnostics
		return fmt.Errorf("ghostty rejected the staged configuration, so nothing was written:\n  %s", strings.Join(diagnostics, "\n  "))
	}
	e.lastValidate = nil
	return nil
}

// Undo reverses the most recent apply.
func (e *Engine) Undo() (Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	result, err := e.undoLast()
	if err != nil {
		return Result{}, err
	}
	if refreshErr := e.refresh(context.Background()); refreshErr != nil {
		result.Warnings = append(result.Warnings, "undo succeeded but re-reading the configuration failed: "+refreshErr.Error())
		return result, nil
	}
	result.Message += " Reload Ghostty (Cmd+Shift+,) to pick the restored configuration up."
	return result, nil
}

// Preview opens an isolated Ghostty window that renders the staged appearance.
// The user's configuration is only read: the specimen runs from a temporary
// configuration file that is kept on disk for inspection.
func (e *Engine) Preview(change Change) (Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.preview(context.Background(), change)
}

// preview is the locked implementation behind Preview.
func (e *Engine) preview(ctx context.Context, change Change) (Result, error) {
	if err := checkValues(change.Values); err != nil {
		return Result{}, err
	}
	if err := e.checkKnownKeys(change.Values); err != nil {
		return Result{}, err
	}
	if err := checkPathValues(change.Values); err != nil {
		return Result{}, err
	}
	app := ghosttyBundlePath(e.native.bin)
	if app == "" {
		return Result{}, fmt.Errorf("native preview needs the Ghostty application bundle; %s is not inside one", e.native.bin)
	}
	candidate := Patch(e.managedState(), change.Values)

	dir := filepath.Join(e.paths.stateDir, "preview", e.previewID())
	if err := ensureDir(dir); err != nil {
		return Result{}, err
	}
	// The staged appearance, the specimen command and the disposable-window
	// overrides all live in the last file Ghostty loads, so nothing in the
	// original configuration can override them.
	// A removed override still exists in the original managed include. Mask it
	// with its inherited value in this disposable preview; Apply will actually
	// remove it from the managed file. Repeated values need an explicit reset.
	previewValues := candidate.Clone()
	inherited := e.valuesWith(candidate)
	for key, values := range change.Values {
		if len(values) != 0 {
			continue
		}
		restored := append([]string(nil), inherited[key]...)
		if IsRepeatable(key) {
			restored = append([]string{""}, restored...)
		} else if len(restored) == 0 {
			restored = []string{""}
		}
		previewValues[key] = restored
	}
	staged := filepath.Join(dir, stagedFileName)
	if err := writeFileAtomic(staged, e.previewStagedConfig(previewValues), 0o600); err != nil {
		return Result{}, err
	}
	path := filepath.Join(dir, "specimen.ghostty")
	if err := writeFileAtomic(path, e.previewConfig(staged), 0o600); err != nil {
		return Result{}, err
	}
	// Never open a window on a harness Ghostty would reject: a rejected harness
	// would leave the specimen running the user's real configuration.
	if err := e.validatePreview(ctx, path); err != nil {
		return Result{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, launchTimeout)
	defer cancel()
	args := []string{"-na", app, "--args", "--config-default-files=false", "--config-file=" + path}
	stdout, stderr, err := e.runner.Run(ctx, Command{Path: openBinary, Args: args})
	if err != nil {
		detail := strings.TrimSpace(string(stderr))
		if detail == "" {
			detail = strings.TrimSpace(string(stdout))
		}
		return Result{}, fmt.Errorf("opening the Ghostty specimen failed (exit %d): %s", exitCode(err), firstLine(detail))
	}
	result := Result{
		Message: fmt.Sprintf("Opened an isolated Ghostty specimen with the staged appearance. The temporary configuration is kept at %s; your own configuration was not modified.", e.displayPath(path)),
	}
	if change.Revision != "" && change.Revision != e.revision {
		result.Warnings = append(result.Warnings, "the change was staged against an older revision; the specimen shows the staged patch applied to the configuration as it is now")
	}
	return result, nil
}

// stagedFileName is the last file a specimen harness loads.
const stagedFileName = "specimen-staged.ghostty"

// previewConfig renders the harness parent: the original configuration ROOTS
// only, then the staged include. Ghostty loads a file's own directives before
// the files it includes and processes includes in declaration order, so the
// staged file is the last thing loaded and beats every original value.
//
// The roots are included individually rather than every file in the resolved
// graph: a nested include belongs to its own root and the parent would otherwise
// load it twice, which masks the specimen and doubles repeated keys.
func (e *Engine) previewConfig(stagedPath string) []byte {
	var b strings.Builder
	b.WriteString("# Ghostty Studio specimen harness. Temporary file; the studio never installs it.\n")
	for _, root := range e.roots {
		fmt.Fprintf(&b, "config-file = ?%s\n", root)
	}
	fmt.Fprintf(&b, "config-file = ?%s\n", stagedPath)
	return []byte(b.String())
}

// previewStagedConfig renders the last file Ghostty loads for a specimen: the
// staged appearance, the specimen command, and the overrides that keep the
// window disposable.
func (e *Engine) previewStagedConfig(managed Values) []byte {
	var b strings.Builder
	b.WriteString("# Staged specimen appearance. Loaded after the original configuration.\n")
	b.Write(renderManagedFile(managed, ""))
	b.WriteString("window-save-state = never\n")
	if e.exePath != "" {
		fmt.Fprintf(&b, "command = %s specimen\n", commandDirective(e.exePath))
	}
	return []byte(b.String())
}

// commandDirective renders the specimen command for Ghostty's own parsing.
//
// Ghostty runs a command with arguments through /bin/sh -c unless it is prefixed
// with `direct:`, and direct mode is documented not to support arguments with
// spaces. A path that needs no quoting therefore uses direct mode, which never
// touches a shell, and a path that does need quoting is placed in single quotes
// behind an explicit `shell:` prefix: the quoting makes the shell see the
// literal path, so nothing in the path can be expanded or executed.
func commandDirective(exePath string) string {
	if isDirectSafePath(exePath) {
		return "direct:" + exePath
	}
	return "shell:" + shellSingleQuote(exePath)
}

// shellMetacharacters are the characters that would change meaning if a path
// reached a shell unquoted, plus the whitespace direct mode cannot express.
const shellMetacharacters = " \t\n\r\"'\\$`*?[]{}()<>;|&!#~"

// isDirectSafePath reports whether a path survives direct mode unchanged.
func isDirectSafePath(path string) bool {
	if path == "" {
		return false
	}
	return !strings.ContainsAny(path, shellMetacharacters)
}

// shellSingleQuote wraps a path in single quotes for /bin/sh, escaping any
// embedded single quote with the standard '"'"' sequence.
func shellSingleQuote(path string) string {
	return "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
}

// validatePreview runs Ghostty's validator over a generated harness.
func (e *Engine) validatePreview(ctx context.Context, path string) error {
	if !e.native.available() {
		return errors.New("cannot validate the specimen harness because the ghostty executable was not found")
	}
	diagnostics, err := e.native.validate(ctx, path)
	e.collectNativeWarnings()
	if err != nil {
		return fmt.Errorf("ghostty could not validate the specimen harness at %s: %w", path, err)
	}
	if len(diagnostics) > 0 {
		return fmt.Errorf("ghostty rejected the specimen harness at %s, so no window was opened:\n  %s", path, strings.Join(diagnostics, "\n  "))
	}
	return nil
}

// previewID names one preview directory.
func (e *Engine) previewID() string {
	e.previewGi++
	return fmt.Sprintf("%s-%d-%d", time.Now().UTC().Format("20060102T150405"), os.Getpid(), e.previewGi)
}

// ghosttyBundlePath returns the .app bundle that contains the executable.
func ghosttyBundlePath(binary string) string {
	const suffix = ".app/Contents/MacOS/ghostty"
	if !strings.HasSuffix(binary, suffix) {
		return ""
	}
	return strings.TrimSuffix(binary, "/Contents/MacOS/ghostty")
}

// samePath compares two paths after cleaning.
func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

// shortHash abbreviates a revision for display.
func shortHash(hash string) string {
	if len(hash) <= 8 {
		return hash
	}
	return hash[:8]
}

// warn records a non-fatal condition.
func (e *Engine) warn(format string, args ...any) {
	e.warnings = append(e.warnings, fmt.Sprintf(format, args...))
}

// checkKnownKeys rejects keys the installed Ghostty does not define.
func (e *Engine) checkKnownKeys(values Values) error {
	var unknown []string
	for _, key := range values.Keys() {
		if _, ok := e.defaults[key]; ok {
			continue
		}
		if _, ok := e.curated[key]; ok {
			continue
		}
		unknown = append(unknown, key)
	}
	if len(unknown) > 0 {
		return fmt.Errorf("unknown Ghostty setting %s; nothing was written", strings.Join(unknown, ", "))
	}
	return nil
}

// pathishKeys are the configuration keys whose value is a filesystem path. The
// studio only ever writes absolute paths so a value can never depend on which
// file it happens to live in.
var pathishKeys = map[string]bool{
	"background-image":    true,
	"custom-shader":       true,
	"cursor-style-shader": true,
	"icon":                true,
	"window-title-font":   true,
	"bell-audio-path":     true,
}

// checkPathValues rejects relative paths in staged values.
func checkPathValues(values Values) error {
	for key, list := range values {
		if !pathishKeys[key] {
			continue
		}
		for _, value := range list {
			if value == "" || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "~") {
				continue
			}
			return fmt.Errorf("%s must be an absolute path (or start with ~), got %q; nothing was written", key, value)
		}
	}
	return nil
}

// normalizeDefaults drops the empty list entries Ghostty prints for repeated
// keys, so a default list is genuinely empty.
func normalizeDefaults(values Values) Values {
	out := make(Values, len(values))
	for key, list := range values {
		cp := make([]string, 0, len(list))
		for _, value := range list {
			if value == "" && IsRepeatable(key) {
				continue
			}
			cp = append(cp, value)
		}
		out[key] = cp
	}
	return out
}

// normalizeChanges keeps show-config output order untouched.
func normalizeChanges(values Values) Values { return values.Clone() }

// copyChoices deep-copies a catalog.
func copyChoices(in []Choice) []Choice {
	out := make([]Choice, 0, len(in))
	for _, choice := range in {
		clone := choice
		clone.Tags = append([]string(nil), choice.Tags...)
		clone.Values = choice.Values.Clone()
		out = append(out, clone)
	}
	return out
}

// groupIndex gives the display order of setting groups.
func groupIndex() map[string]int {
	return map[string]int{
		GroupTypography: 0,
		GroupWindow:     1,
		GroupCursor:     2,
		GroupBehavior:   3,
		GroupShortcuts:  4,
		GroupReference:  5,
	}
}

// settingsFromDocs merges the native documentation with the curated schema.
func settingsFromDocs(docs []settingDoc, curated map[string]curatedSetting) []Setting {
	settings := make([]Setting, 0, len(docs))
	seen := make(map[string]bool, len(docs))
	for _, doc := range docs {
		settings = append(settings, settingFromDoc(doc, curated))
		seen[doc.key] = true
	}
	for _, entry := range curatedSettings {
		if seen[entry.key] {
			continue
		}
		settings = append(settings, settingFromCurated(entry))
	}
	sortSettings(settings)
	return settings
}

// settingsFromCurated builds the schema without native documentation.
func settingsFromCurated() []Setting {
	settings := make([]Setting, 0, len(curatedSettings))
	for _, entry := range curatedSettings {
		settings = append(settings, settingFromCurated(entry))
	}
	sortSettings(settings)
	return settings
}

// sortSettings orders settings by group and then by key.
func sortSettings(settings []Setting) {
	index := groupIndex()
	sort.SliceStable(settings, func(i, j int) bool {
		gi, gj := index[settings[i].Group], index[settings[j].Group]
		if gi != gj {
			return gi < gj
		}
		return settings[i].Key < settings[j].Key
	})
}
