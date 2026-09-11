# Ghostty TUI audit

11 September 2026 · macOS · source inspection, current upstream documentation, native CLI probes, build, and isolated runtime checks.

## Recovered artifacts

- TUI source: `/Users/danielgosek/dev/_archive/dev-root-gade-era-20260707/dotfiles/ghostty/ghostty-tui/src/main.rs` — 1,238 lines.
- Manager: `/Users/danielgosek/dev/_archive/dev-root-gade-era-20260707/dotfiles/ghostty/ghostty-manager.sh` — 976 lines.
- Cargo package: `ghostty-tui` 0.1.0; Rust 2021; Ratatui 0.29; Crossterm 0.28; serde/serde_json.
- Existing corpus: 9 cursor shaders, 12 screen shaders, 5 custom themes, 10 preset configs.
- Historical test report dated 2025-12-30. Its manual checklist is unchecked; its automated claims are build/structural checks rather than functional tests. It describes an older 1,019-line source.
- Live installed terminal: `/Applications/Ghostty.app/Contents/MacOS/ghostty`, stable 1.3.1, CoreText, Metal.

No independent repository was found inside the archived project. No live config, archived source, or archived assets were edited. Build artifacts went to `/tmp/ghostty-tui-audit-20260911-target`. Public Cargo dependencies were downloaded with the existing lockfile after an offline build lacked a cached dependency.

## Findings

| Priority | Finding and evidence | Consequence |
|---|---|---|
| High | `main.rs:85` searches obsolete manager paths; `ghostty-manager.sh:28` hardcodes the old asset root. A freshly built default launch exits 1 with “Could not find ghostty-manager.sh.” | Recover an active app location and resolve assets relative to a supported installation. |
| High | `main.rs:733` tests `0.9` before `0.95`. In a real 80×24 PTY, five Tabs, five `j` presses and Enter selected 0.95 but wrote `background-opacity = 0.9` to a fixture. | Use typed values; labels must not drive behavior. |
| High | `ghostty-manager.sh:359` replaces every `font-family` occurrence. A two-family Iosevka/Menlo fixture became two identical JetBrains Mono entries. | Preserve and deliberately edit fallback lists. |
| High | `ghostty-manager.sh:148` removes the previous screen stack before checking new assets. The original script's missing-asset test exited 1 after deleting the old shader and animation entries. | Validate and stage before committing. |
| High | `main.rs:806` composes an unquoted script path and interpolated command into `sh -c`. Current names are hardcoded, limiting exposure; future imported names would reach this path. | Replace the shell mutation path before adding imports. No exploit was run. |
| Medium | `main.rs:191` has 16 actual theme names; `main.rs:193` has 10 font families. | README's “200+ themes” does not describe the TUI's selectable catalog. |
| Medium | The 11-tab header clips at 80 columns. The PTY showed “Shel” and omitted the Config tab label. Numeric labels are displayed but the event handler has no numeric tab shortcuts. | Fewer top-level choices and real shortcut contracts. |
| Medium | Theme/font rendering functions display names, not appearance previews; selection invokes the writer immediately. | Browsing is not a comparison workflow. |
| Medium | `get_ghostty_config_path` selects one legacy path; the Bash layer separately honors `GHOSTTY_CONFIG_PATH`. Its “same directory” manager probe also ascends one directory too far. | Unify path and config-graph discovery, including `.ghostty` filenames and XDG/macOS layers. |
| Medium | `ghostty-manager.sh:628` maps tab-bar controls to a GTK setting on macOS. `show_current:738` summarizes six values, uses `cursor-style-shader` for cursor reporting, and labels all `custom-shader` entries as screen shaders. | Display platform-appropriate settings and actual effective config; the Config view is not a full-file inspector. |
| Medium | Config writes have no native validation, concurrency protection, or transactional undo. Whole-preset loading backs up then replaces the full file. | Preserve unrelated user settings and provide reversible scoped changes. |

## Current-machine observations

Both `/Users/danielgosek/.config/ghostty/config` and `/Users/danielgosek/Library/Application Support/com.mitchellh.ghostty/config` exist. Both passed an explicit `+validate-config --config-file=...` invocation. The default configuration validation also returned 0. This is parser acceptance, not proof that every platform-specific setting has a visual effect.

`+show-config --changes-only` reported Nord, 14-point text, Bold, disabled `calt`, 0.9 opacity, and two font-family preferences. `+show-face --string='Aa0O1l→'` resolved those codepoints to JetBrains Mono. This lookup is not full grapheme/shaping validation.

The installed resources directory contains **463 regular theme files**. `+list-fonts` succeeded and listed **18 family headings**, including Iosevka, Iosevka Fixed/Term, Martian Mono, Director, and Nerd Font variants. Ghostty's embedded faces are not necessarily all represented by those headings.

**Native catalog caveat:** `+list-themes --plain` and `--path` returned exit 1 with partial output and no stderr in this environment; a regular-file stdout probe also failed. The root cause was not diagnosed. The 463 figure comes from filesystem inventory, not a successful CLI listing. Discovery must check exit status/completeness and offer a resource-file fallback.

The bundled manpage's path-order wording differs from the current website. Precedence must be covered by version-specific executable fixtures before writing a resolver. The effective output was used only to describe observed values here.

## Verification performed

- `cargo test --locked --target-dir /tmp/ghostty-tui-audit-20260911-target`: build succeeds; **0 tests** run.
- `cargo build --locked --target-dir /tmp/ghostty-tui-audit-20260911-target`: succeeds.
- Fresh binary, default launch: manager discovery fails.
- Fresh binary with explicit manager and temporary config: starts; navigate to Appearance; reproduce 0.95→0.9; quit cleanly.
- Original manager on temporary configs: theme replacement and append succeed; fallback-list loss reproduced; missing-shader removal reproduced.
- Audit-only manager copy with its asset root relocated to the recovered corpus: `crt-classic` writes both shaders successfully on this Mac. That copy is a fixture aid, not a patch to the app.
- Deliberately invalid config fixture: native validator returns 1 with both unknown-field and invalid-font-size diagnostics.
- No shader was executed on the GPU, font installed, live config reloaded, app integration enabled, or real terminal appearance changed.

`fixture-results.json` preserves the exact shell test inputs, outputs, and resulting config text. `before-sha256.json` records source/config hashes; `verification.json` records the final preservation check. The HTML study is a simulated interaction concept, with six real bundled color palettes used for the color specimen. It does not demonstrate Ghostty font, shader, reload, or installation behavior.

## Research that shaped the proposal

| Primary source | Useful conclusion |
|---|---|
| [Ghostty configuration](https://ghostty.org/docs/config) and [reference](https://ghostty.org/docs/config/reference) | Resolve includes and repeatable settings, respect reload requirements, and validate against installed capabilities. |
| [Theme documentation](https://ghostty.org/docs/features/theme) and [iTerm2 catalog](https://github.com/mbadolato/iTerm2-Color-Schemes) | Reuse the bundled/upstream ecosystem; avoid duplicated catalogs and review imported configuration. |
| [Ghostty 1.3 release notes](https://ghostty.org/docs/install/release-notes/1-3-0), [1.3.1 notes](https://ghostty.org/docs/install/release-notes/1-3-1) | Make new native workflows discoverable; AppleScript remains a version-sensitive adapter. |
| [ghostty.styles.tui](https://github.com/mcfearsome/ghostty.styles.tui) | Search, tags, collections and OSC color preview are already ecosystem features. The proposed advantage is coherent typography/Looks, config provenance, and reliable native preview. Repository review only; not installed or benchmarked. |
| [Theme/font mixer](https://github.com/mcvalosborne/ghostty-theme-mixer) | Combining theme and type is an effective exploration model. A browser specimen is not native rendering proof. Repository review only. |
| Font authors and shader repositories linked in PLAN.md | Curate distinct choices with metadata and per-asset provenance, rather than merely inflating the list. |

Also read the installed Ghostty 1.3.1 `ghostty.5.md`, native action help, and `Ghostty.sdef`. The shared UI catalog was consulted; its web component resources did not justify a framework change for this Rust TUI.
