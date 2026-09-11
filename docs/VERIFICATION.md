# Ghostty Studio 0.2.0 — verification

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
