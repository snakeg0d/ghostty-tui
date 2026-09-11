# Ghostty Studio — verification

## 0.2.1 · visual refinement · 11 September 2026

Installed to `~/.local/bin/ghostty-tui`; the installed binary matches the build. The palette browser now has separated swatches, adaptive two-line entries and a full-width selection rail. Stable ink/mint chrome, numbered tabs and keycaps give navigation clearer hierarchy. Fonts have a dedicated specimen with complete alphabets, ambiguity checks and three weight samples. Compact specimens use complete shorter phrases. Looks show meaningful composition values before raw configuration details. Nested ANSI resets now restore their enclosing background, keeping selections and modal surfaces continuous.

`go test ./...`, `go vet ./...` and the UI race checks passed. New regression checks cover reserved specimen headings, intact sample groups, complete alphabets, compact frames, palette rhythm, roomy names, single favorite markers and visible save feedback. Geometry checks include 60×18, 80×24, 120×36 and larger terminals.

Actual PTY acceptance at 120×36 covered theme search/stage/apply/undo, font search/staging, Help and dirty-quit cancel/discard. Undo restored the disposable config byte for byte. The installed binary was also exercised from `/tmp` at 80×24, including Help paging and quit. Both personal Ghostty config hashes remain unchanged. No fonts or personal settings were changed in this pass.

The refreshed screenshots render the production Bubble Tea View through Charm Freeze with JetBrains Mono. Full and compact Themes, Fonts, Looks, Effects and Help, plus a light palette, were inspected. A fresh Astra peer reviewed the diff and frames, identified duplicate favorite markers and clipped Look subtitles, and confirmed both repairs. The existing native-window and shader-rendering limitations below still apply; these images are terminal renders, not photographs of a Ghostty window.

Implementation reused the existing visible Herdr/OMP UI session with the verified exact `deepseek/deepseek-v4.1-flash` model, high effort, personal OpenRouter account. Astra owned design, render review, bounded integration corrections, PTY verification and installation. The native session, briefs, before/after images and PTY transcripts are retained under `.build/`.

## 0.2.0 · original release

11 September 2026 · macOS · Ghostty 1.3.1 (CoreText / Metal).

## Delivered

- Native Go TUI using Bubble Tea v2, Bubbles v2, and Lip Gloss v2.
- Installed command: `~/.local/bin/ghostty-tui`. Verified from `/tmp`, outside the source checkout.
- Installed assets: `~/.local/share/ghostty-studio/assets`.
- Real catalogs on this machine: **463 themes**, **25 installed font families**, **200 distinct native settings**, **5 composed Looks**, **3 new effects plus None**, and **5 workflow presets**.
- Maple Mono 7.9 and Monaspace 1.400 installed through the app's Homebrew integration. Their native families were verified through Ghostty. Existing manual CommitMono files were preserved; targeted family lookup finds them despite their omission from the unfiltered native listing.

## Checks completed

| Check | Result |
|---|---|
| `go test ./...` | Passed across CLI, core, and UI. Native tests used disposable configurations. |
| `go test -race ./...` | Passed. UI race checks were repeated after the final asynchronous import/refresh integration. |
| `go vet ./...` | Passed. |
| Native configuration acceptance | 20/20 checks passed: catalog discovery, exact opacity, include preservation, validation, byte-identical undo, invalid input, injection rejection, and outside edits. |
| Native preset acceptance | 17/17 cases passed: every Look, every workflow, None and three effect configurations, CommitMono, Maple Mono, and Monaspace Neon. Each case applied and undid its configuration while preserving the existing shortcut. |
| CLI lifecycle | Real native validation of diff/apply/rejected edit/undo; saved Look export/import/reopen round-trip; command-bearing Look rejected. |
| Actual input | PTY walkthroughs at 120×36 and 80×24: theme/font search, staging, apply, undo, More/settings, help paging, and quit. Installed-binary font apply/undo restored the original fixture bytes. |
| Visual review | Actual Bubble Tea `View` captured with Charm Freeze. Full and compact views were inspected; screenshots are not HTML mockups. |
| Preservation | Both personal Ghostty configs and four archived source/package files match their original SHA-256 hashes. All 21 recovered GLSL files remain byte-identical. |

Regression checks cover the issues found during integration: include order in preview, preview of removed overrides, fallback-chain replacement, successful font-install return without deadlock, saved Look discovery, workflow keybinding preservation, compact help controls, visible reload feedback, stale-edit recovery, and repeated-value editing.

## Scope of proof

Computer-use access to the native Ghostty and Codex windows was rejected by automatic approval review. No alternate UI-control route was used. Native preview's generated configuration, precedence, command quoting, validation-before-launch, and launcher arguments are covered by injected-runner tests. Its actual window rendering remains unverified.

Configuration validation does **not** compile GLSL or measure GPU cost. The three new effects have validated configuration paths, but their renderer output needs inspection in Ghostty. Inline specimens show palette colors using the font of their host terminal. The `p` action is the native font/effect comparison route.

The tool environment sets `NO_COLOR=1`; the PTY respected it. A separate truecolor PTY run clears that variable for visual protocol verification. Freeze snapshots render the actual ANSI View with a monospace font.

## Implementation provenance

Most core and UI implementation ran on the verified exact model `deepseek/deepseek-v4.1-flash`, using the existing personal OpenRouter account in two interactive Herdr + OMP panes. GMICloud was returned by the initial model identity probe; individual worker requests' upstream routing is not asserted. The sessions reached their time caps and were resumed; final integration and acceptance were completed by Astra. No substitute implementation model was used.

Astra handled the design, legacy recovery, current documentation research, CLI, original shader assets, integration corrections, test review, visual review, and installation. The local `.build/` folder retains acceptance receipts, PTY recordings, and runtime records; it is intentionally excluded from Git.
