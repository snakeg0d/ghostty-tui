# Implement Ghostty Studio

Implementation brief retained for provenance. The working release and verified behavior are documented in [README](../README.md) and [verification](VERIFICATION.md). Final ownership was core in one DeepSeek Herdr/OMP pane, UI in the other, with Astra implementing the CLI, integration corrections, assets, review, and installation.

The user now explicitly requests the whole working app, not a plan or website. This replaces the earlier Rust proposal. Deliver an exceptional, usable Bubble Tea v2 terminal application with safe Ghostty configuration and actual local catalogs. Primary platform macOS, installed Ghostty 1.3.1. No web runtime in product.

## Ownership

- Core worker: `internal/core/**` only. Implement every shared API below plus meaningful tests. Standard library preferred. Do not edit UI/main/go.mod.
- UI worker: `internal/ui/**` and its tests. Astra owns `cmd/ghostty-tui/**` and `Makefile` after the integration handoff.
- Astra: this brief, assets, integration fixes, QA, README, install and actual design review. Communicate any necessary API changes through a short file under `.build/`, otherwise obey exact contract.

## Shared Go API contract (package core)

```go
type Values map[string][]string
type Palette struct { Background, Foreground, Cursor, Selection string; ANSI [16]string }
type Choice struct {
 ID, Name, Description, Source, Category string
 Tags []string
 Palette Palette
 Values Values
 Installed bool
 URL, License string
}
type Setting struct {
 Key, Name, Description, Group, Default, Kind, Reload string
 Options []string
}
type Options struct { ConfigPath, StateDir, GhosttyPath, AssetDir string }
type State struct {
 Current Values
 ConfigPath, GhosttyPath, Version string
 Sources, Warnings []string
 Revision string
}
type Change struct { Values Values; Revision string; Label string }
type Result struct { Message, BackupPath string; Warnings []string }
type Diagnostic struct { Level, Title, Detail string }

func Open(Options) (*Engine, error)
func (e *Engine) State() State
func (e *Engine) Refresh() error
func (e *Engine) Catalog(section string) []Choice // themes, fonts, discover, looks, effects, workflows
func (e *Engine) Settings() []Setting
func (e *Engine) Favorites() []string
func (e *Engine) ToggleFavorite(id string) error
func (e *Engine) Diff(Change) string
func (e *Engine) Apply(Change) (Result, error)
func (e *Engine) Undo() (Result, error)
func (e *Engine) Preview(Change) (Result, error) // launch isolated native Ghostty specimen; never mutate config
func (e *Engine) InstallFont(id string) (Result, error) // selected discover item only, bounded subprocess
func (e *Engine) SaveLook(name string, values Values) error
func (e *Engine) ExportLook(name string, values Values, path string) error
func (e *Engine) ImportLook(path string) (Choice, error)
func (e *Engine) Diagnose() []Diagnostic
func (e *Engine) ExportTheme(values Values, format string) (string, error) // ghostty, fzf, terminal palette JSON
```

Choice ID includes section/source identity, e.g. theme:nord or font:Menlo. UI must retain ID from core, not reconstruct it. Catalogs return a copy and cannot race with background refresh/install. State likewise copies. Change.Values is an explicit patch: absent key unchanged; empty slice resets/unsets managed value; repeat values preserve ordering and explicit reset semantics. Change.Revision is State.Revision at staging time. Apply must reject stale revision. Commands return honest saved/reload/new-terminal status; never claim reload if not done. No fake installed statuses or lists. Theme Values includes complete safe palette (rather than named theme alone) so existing personal color overrides cannot mask the specimen. Applying theme must not apply arbitrary commands from theme files.

## Core behavior

Discover binary from explicit flag/environment, PATH, /Applications/Ghostty.app/Contents/MacOS/ghostty. Respect GHOSTTY_CONFIG_PATH and GHOSTTY_TUI_STATE_DIR overrides for fixtures. State dir default user config ghostty-studio (or XDG state) and no writes on mere startup. Config candidates: XDG ghostty config.ghostty/config and Mac Application Support/com.mitchellh.ghostty/config.ghostty/config; include graphs matter. Parent is proving precedence independently. Read raw docs preserving comments and unknown keys. Determine effective values using native show-config. Source graph hashes form Revision. Respect symlinks by resolving target; include relative paths remain relative.

Safe persistence: use a managed block at a verified winning point or managed include. Validate full candidate via Ghostty native validator with config-default-files=false. Include inheritance must not be broken by staging to another directory. Check all source hashes before writes, lock concurrent transactions, unique persistent backup and journal before same-directory atomic rename, original mode preserved. Undo rejects external edits and restores byte-identical original. If complex include graph can't be safely modified, report clear error instead of silently clobbering. Theme patch explicit palette prevents overrides. Fonts reset full regular chain but preserve fallback choices (primary moves to front; do not discard existing fallbacks); bold/italic should follow selected family through proper resets and retain fallback. Native repeat palette syntax must use index=hex. Zero effect on failed validation. Exec.Command argument arrays, timeouts, reject CR/LF injection in values, no sh -c for config/install actions.

Catalog themes from installed resources and local theme directories: /Applications/Ghostty.app/Contents/Resources/ghostty/themes has 463 actual files; +list-themes on this build exits 1 with partial output! Never depend solely on it. Parse palette safely, derive light/dark by luminance and contrast from foreground/background; retain source paths, disambiguate names. Include existing local themes. Curated ordering of Nord, Catppuccin Mocha, TokyoNight, Rose Pine, Kanagawa, Flexoki Dark, Everforest Dark, Gruvbox Dark, GitHub Light when real entries exist; remainder alphabetical. No network needed for startup.

Fonts from native +list-fonts: family lines unindented, styles indented. Embedded JetBrains Mono may be added with accurate source label; verify +show-face. Discover shelf of actual Homebrew font casks (parent verifies IDs): Monaspace, Maple Mono, Commit Mono, Berkeley Mono not free so omit install, Iosevka, JetBrains Mono. Install selected known cask with brew install --cask using exec array, no shell, don't uninstall existing fonts; timeout and honest errors, refresh afterward. Show source/license/install destination in UI before install. No unattended install all.

Looks: five editorial appearance presets built from actual catalog entries and available fonts: Everyday (calm), Daylight (light), Dense (small typography), Presentation (large clear), After Hours (warm). Never touch clipboard permissions/commands. Users save current/staged appearance as named Look, export/import allowlisted JSON. Imported shader paths untrusted must validate before apply; no arbitrary code or external command values. Exports strip unrelated config. fzf output is a shell variable snippet from validated hex colors, never execute or edit shell files.

Effects: parent adds local original/provenanced shader assets. Catalog includes None, and files under assets/shaders + user shader dir; validate file exists before changing. Ordered stack is represented in custom-shader. Reset only on explicit None choice. Whole-stack disable recovery. Animation false default. Explain GLSL compile is verified only by renderer, not validate-config. No fake intensity slider. Native preview may show shaders/fonts through disposable config --config-default-files=false --config-file=PATH launched via /usr/bin/open -na /Applications/Ghostty.app --args ...; sessionrestore disabled, command=self specimen. Parent cannot use Ghostty CUA due tool safety block, so implement route but don't autonomously open GUI during worker tests. Preserve preview temp until process/cleanup, no early delete. CLI specimen renders ANSI colored code/git/typography samples and waits for key if terminal interactive, finite output for non-TTY.

Settings: parse installed +show-config --default --docs for a searchable exhaustive reference; curated editable schema for safe ordinary typed settings, unknown advanced entries reference-only unless safely validated. Groups Typography, Window, Cursor, Behavior, Shortcuts. At least font-size, font-thicken, font-feature, font-family fallbacks, adjust-cell-height, window-padding-x/y, background-opacity, background-blur, cursor-style, cursor-style-blink, mouse-hide-while-typing, scrollback-limit, window-save-state, quick-terminal-position, shell-integration, custom-shader-animation, background-image. Field validates with Ghostty before save; show type and reload semantics, docs. All Settings can search every key, not just a fixed tiny list. Workflows catalog curated typed patches for quick terminal, notifications, cwd inherit, option as alt, scrollback search; never override existing keybind without exposing diff/conflict. Existing docs at /Applications/Ghostty.app/Contents/Resources/ghostty/doc/ghostty.5.md and ghostty.1.md.

Tests required: comments+unknown keys preserved; repeats, includes/relative includes, symlinks, stale revision and Undo conflict; invalid patch no live writes; malicious names cannot execute or inject new keys; file missing; theme fallback discovery; accurate opacity .95. Tests use temp roots and a deterministic fake Ghostty executable or real binary when available. No test should edit personal configuration or install fonts.

## UI design: a terminal atelier

Brand name Ghostty Studio, command ghostty-tui. Dark mineral canvas #101518, ivory text #e5e8df, muted sage #a8b9a5 and amber #dab979; clear quiet contrast, no purple gradient/cyberpunk/no wall of rounded cards. Modern editorial rhythm with generous internal whitespace, fine separators, restrained Unicode, no emoji-dependent alignment. Chrome stays stable while specimen changes palette. Use Lip Gloss v2, ANSI-aware clipping/width, Bubbles textinput/viewport/help/spinner. Genuine terminal rendering only.

At 120x36: top identity at x3/y1, small 'MAKE THE TERMINAL YOURS' descriptor, right native version. Under that tabs '1 Themes   2 Fonts   3 Looks   4 More'. Search field below. Main left list ~32 columns; selected row colored background/small marker, theme swatches at right; favorites/recent at top. Right big terminal specimen framed with thin border and top tab title 'preview / <theme>'. Specimen contains syntax colored small code, git diff, shell prompt and agent output, palette swatches. Footer detail row: light/dark, numeric contrast, source; pending change summary; always visible 'enter stage   a apply   p native preview   / search   ? help'. Adapt help to screen. Avoid decorative border nested inside border everywhere. At 80x24 stay fully usable, list ~27 with compact specimen or switchable details; minimum 60x18 truthful resize message below. Never spill outside size; modal scroll for long content. Theme browse fast with no disk writes.

Progressive disclosure: main is Choose. Enter stages focused item (no write); 't' opens Tune controls for that section; 'i' opens Inspect (diff, current/config sources, docs). 'a' applies staged settings, undo u always available. Escape exits search/modal without quitting. q quits only when not editing, dirty confirm offers discard/cancel. Search handles text first; j/k in text inputs type normally. Tab moves sections or focus in modal consistently, 1..4 direct. '['/']' size quick tune or tabs only if clearly shown. ? helpful full shortcut overlay.

Theme list supports fuzzy/subsequence search, favorite f (persist IDs), filter l cycles all/dark/light/favorites, pin b and compare second sample A/B when width allows. Must have well-designed empty states. Hovering choices updates inline specimen only; pending vs current vs focus labels accurate. Choice.Values merged with previous stage so theme+font compose. Full apply diff in Inspector.

Fonts: installed/discover subview (d toggles; visible hint). Good actual typography specimen with `0 O o  1 l I |  { [ (  => != <=`, regular/bold/italic, box drawing, nerd symbols, emoji, CJK; clearly says inline uses current terminal font, native preview uses selected font. Enter stages installed, discover opens install review sheet with URL/license/destination and explicit I installs; install asynchronous spinner then result, no config change on error. Tune exposes size and features/fallback via settings editor.

Looks: editorial descriptive list, miniature themes; stage multiple cosmetic settings. s saves with name input, e exports with path input, o imports path. Show result/status on every operation. Current and staged values merged for save/export only allowlist core filters.

More: Effects / All settings / Workflows / Health / Export palette. Effects toggles ordered list with space, changes staged not instantly written; show order and x clear. Settings is searchable names/descriptions/keys, shows current value and own docs. Enter opens editable value input with enum hints and validation; reference-only keys clearly explained. Health native version, config source paths, conflicting file warnings, font/asset issues, backup guidance. fzf export uses named path input or CLI stdout, no rc file editing.

CLI (same core, no second mutation implementation): no args launches TUI; --config PATH, --state-dir PATH, --ghostty PATH, --assets PATH flags; --version; doctor; themes [query]; fonts; apply --theme NAME | --font NAME | --set key=value (multiple); undo; diff same action flags; look export/import; export --format fzf|ghostty|json; specimen; snapshot --width N --height N --section themes|fonts|looks|effects|settings --query TEXT emits exact TUI View ANSI to stdout for deterministic visual QA. CLI mutations explicit, stderr errors exit1; stdout machine-readable where appropriate. Snapshot MUST instantiate same UI model and View, not a separate mock. Makefile build/test/install to ~/.local/bin (do not run install as worker).

UI tests: 80x24 and120x36 cell width/height, search typing vs global shortcuts, every displayed binding, long names/unicode, no results, dirty quit, stage/apply/undo failure surfaces. Native PTY tests by parent. Don't block startup on slow docs query; load catalogs once/async if needed. No gratuitous animation except spinner while actually working.

## Evidence and stopping point

Core worker stops after all API implementations and owned tests pass. UI worker stops after complete TUI+CLI compiles when core available and UI tests pass. Do not claim visual perfection or native shader compilation; parent will review actual output. No TODO buttons, placeholders, fake downloads or ignored errors. If API/work is blocked by the other worker, finish independent files then describe precisely. Do not ask user questions; choose reasonable defaults inside brief.

Read current primary docs https://github.com/charmbracelet/bubbletea and installed module sources. v2 Model.View() returns tea.View, tea.NewView(string); input type tea.KeyPressMsg; View.AltScreen=true. go.mod pins verified latest stable BubbleTea2.0.9, LipGloss2.0.6, Bubbles2.2.1. Do not use stale v1 APIs.
