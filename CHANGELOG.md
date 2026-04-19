# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] — 2026-04-19

First public release. Source extracted from dotfiles into a standalone repo.

### Added
- **Bundled asset library** (`assets/`) — 5 themes + 9 cursor shaders + 12 screen shaders + 10 config presets, installable independently via `curl` one-liners from the raw-github URL
- **`--install-assets` subcommand** — copies bundled themes + shaders to `~/.config/ghostty/{themes,shaders/{cursors,screen}}/`, idempotent
- **`--assets-path` subcommand** — prints the resolved assets directory (debug aid)
- **Dynamic theme discovery** via `ghostty +list-themes` (~460 themes on 1.3.1); falls back to a curated list if ghostty isn't in `$PATH`
- **Dynamic shader discovery** — walks `assets/shaders/{cursors,screen}/` for `*.glsl` at startup instead of a hardcoded list
- **Reload hotkey detection** — TUI surfaces the user's active `reload_config` keybind in the status bar
- **Ghostty version warning** — amber status-bar note if Ghostty < 1.2 is detected
- **Integration tests** — 10 inline unit tests + 5 CLI subprocess tests
- **GitHub Actions CI** — fmt / clippy / test / build / smoke matrix on macOS + Ubuntu, plus a repo-wide scrub check that fails on any leaked `/Users/<name>` path
- **`docs/ASSETS.md`** — per-asset curl install guide
- **`docs/ARCHITECTURE.md`** — three-layer design explainer
- **`docs/ADOPTING.md`** — agent-first fork guide for Alacritty / Kitty / WezTerm / foot + machine-readable command-vocabulary schema
- **`scripts/release.sh`** — builds + zips platform-specific release archives under `dist/`
- **shell-words** dependency (1.1) for safe command tokenization
- Release profile optimizations: thin LTO, strip, codegen-units=1

### Changed
- **`scripts/ghostty-manager.sh`** relocated into the repo; hardcoded `$HOME/dev/dotfiles/ghostty` replaced with a four-tier `GHOSTTY_TUI_ASSETS` resolution chain
- Font-tab labels renamed from `Weight: Normal / Bold` to `Style: Regular / Bold / Italic / Bold Italic` to accurately reflect Ghostty's `font-style` semantics (Ghostty has no numeric weight key)
- `set_font_weight` bash handler accepts all four face-name styles + empty (which strips the override)
- Repo identity: `authors = ["snakeg0d <danielgosek@gmail.com>"]`, `repository = "https://github.com/snakeg0d/ghostty-tui"`
- Version 0.1.0 → 0.2.0
- Rust MSRV 1.70 → 1.75 (current ratatui 0.29 floor)
- Release profile hardened (thin LTO, strip, single codegen unit)

### Fixed
- **Shell-injection surface removed.** `load_current_config()` and `execute_command()` no longer pipe interpolated strings through `sh -c`; they invoke the manager script directly with tokenized `args(&parts)` via `shell-words`
- Manager-script path resolution now climbs an extra parent for `target/release/` dev layout so `--assets-path` and adjacent-binary lookup both work from a Cargo build
- `src/main.rs` hardcoded `~/dev/dotfiles/ghostty/ghostty-manager.sh` dev-path fallback removed (leaked author's local layout)

### Removed
- `IMPROVEMENTS.md` and `QUICKSTART.md` — content merged into README + this changelog
- Hardcoded custom theme list (`tokyonight-night`, etc.) — now discovered from disk or via `ghostty +list-themes`
- `config` crates.io category (invalid)

## [0.1.0] — 2025-12-30

### Added
- Initial release
- 11 comprehensive configuration tabs
- 200+ built-in theme support (hardcoded shortlist)
- Cursor and screen shader effects
- Live config updates
- Keyboard-driven navigation
- TokyoNight color scheme
