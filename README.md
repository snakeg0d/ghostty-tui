# ghostty-tui

A keyboard-driven terminal UI for [Ghostty](https://ghostty.org). Browse 460+ themes, apply shader effects, tweak cursors, fonts, windows, clipboard, shell integration — all without leaving the terminal. Writes directly to your Ghostty config; new windows pick up every change.

![status](https://img.shields.io/badge/status-v0.2.0-blue)
![ghostty](https://img.shields.io/badge/ghostty-1.3.x-success)
![platforms](https://img.shields.io/badge/platforms-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)
![license](https://img.shields.io/badge/license-MIT-green)

> **Ghostty compatibility:** tested on Ghostty 1.3.x. Theme discovery calls `ghostty +list-themes`; if Ghostty is older than 1.2 a warning is shown in the status bar. Falls back to a curated theme list if ghostty isn't in `$PATH`.

<!-- Demo GIF — record with `vhs docs/demo.tape` and check in docs/demo.gif -->
<!-- ![demo](docs/demo.gif) -->

## Install

```bash
git clone https://github.com/snakeg0d/ghostty-tui
cd ghostty-tui
cargo build --release
./target/release/ghostty-tui
```

Or install straight from the checkout:

```bash
cargo install --path .
ghostty-tui
```

Requires **Rust 1.75+** and **Ghostty 1.3.x** (1.2+ works, older may no-op some keys).

## Use it

```bash
ghostty-tui
```

| Key | Action |
|---|---|
| `Tab` / `Shift+Tab` | Switch between tabs |
| `↑` / `↓` or `j` / `k` | Navigate options |
| `Enter` | Select — writes the change to your Ghostty config |
| `r` | Re-read the config file into the Cfg tab |
| `q` / `Esc` | Quit |

Changes are applied to the config file immediately, but Ghostty itself only re-reads the config on reload. Three ways to reload:

- **Reload keybind** — bind it in your config: `keybind = super+r=reload_config`
- **Open a new window/tab** — new surfaces re-read the config on launch
- `ghostty-tui` surfaces your current reload keybind in the status bar if set

## 11 configuration tabs

1. **Theme** — 460+ themes via `ghostty +list-themes`, with a curated Popular header
2. **Font** — family / size / style (Regular, Bold, Italic, Bold Italic) / ligatures
3. **Cursor** — style, color (match-theme or hex), blink, opacity
4. **CFX** — cursor shader effects (glow, tail, warp, ripple, boom, tesla…)
5. **SFX** — screen shader effects (CRT, bloom, matrix, glitch, water, starfield…) + named combos
6. **Look** — opacity, blur, padding presets
7. **Mouse** — hide-while-typing, scroll-multiplier, copy-on-select, focus-follows-mouse
8. **Clip** — clipboard read/write perms, paste-protection, trim whitespace
9. **Win** — window size presets, decorations, tab bar, vsync, fullscreen, save-state
10. **Shell** — integration per shell, working-directory, close-confirm, scrollback limit
11. **Cfg** — view the current resolved config

## Bundled asset library

36 assets ship with the binary:

- **5 themes** — onenord, onenord-light, tokyonight-{day,night,storm}
- **9 cursor shaders** — glow, tesla_coil, cursor_tail, cursor_sweep, ripple_cursor, …
- **12 screen shaders** — bettercrt, tft, retro-terminal, amber-glow, bloom, matrix-hallway, …
- **10 config presets** — minimal, focus, presentation, agentic, tmux-style, aecsar-wide, retro, gruvbox-monolisa, zapman-tokyonight, plutov-cyberdream

Install them in one shot:

```bash
ghostty-tui --install-assets
```

This copies every theme + shader (not configs — those are opt-in per-file) into `~/.config/ghostty/{themes,shaders/{cursors,screen}}/`. Idempotent — re-running skips existing files.

Or install individual assets via curl one-liners — see [`docs/ASSETS.md`](docs/ASSETS.md).

## CLI

```
ghostty-tui                      # Launch the TUI
ghostty-tui --help               # Full help + keybinds
ghostty-tui --version            # Print version
ghostty-tui --config-path        # Print Ghostty's config path on this platform
ghostty-tui --assets-path        # Print where bundled assets resolved from
ghostty-tui --install-assets     # Copy bundled themes + shaders to user config
```

Environment variables:

| Var | What it does |
|---|---|
| `GHOSTTY_MANAGER_PATH` | Override path to `ghostty-manager.sh` |
| `GHOSTTY_TUI_ASSETS` | Override bundled assets dir |
| `GHOSTTY_THEMES_DIR` | Override user themes dir (default `~/.config/ghostty/themes`) |
| `GHOSTTY_CONFIG_PATH` | Override active Ghostty config path |

## Platform support

| Platform | Status |
|---|---|
| macOS | ✅ Primary development platform |
| Linux | ✅ Full support (XDG Base Directory spec honored) |
| Windows | ⚠️ Implemented — community testing welcome |

## Design

Three-layer:

1. **ratatui TUI** (Rust) — event loop, styling, tab/option model
2. **`ghostty-manager.sh`** (bash) — write-through config-file manipulation
3. **Ghostty's config file** — plain flat-KV; TUI edits it, Ghostty re-reads on new window

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for the full diagram and layer contract.

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md). PRs welcome.

Want to fork this for **Alacritty / Kitty / WezTerm / foot**? See [`docs/ADOPTING.md`](docs/ADOPTING.md) — a machine-readable schema and per-step swap checklist for agents (Claude Code, Codex, etc.) to perform the fork autonomously.

## Changelog

See [`CHANGELOG.md`](CHANGELOG.md).

## License

MIT — see [`LICENSE`](LICENSE).

## Acknowledgments

- [Ghostty](https://ghostty.org) by Mitchell Hashimoto
- [Ratatui](https://github.com/ratatui/ratatui) for the TUI primitives
- Shader authors credited individually in `assets/shaders/README.md`

## Related

- [Ghostty](https://ghostty.org) — the terminal
- [Spectre](https://spectre-ghostty-config.vercel.app/) — a web UI for Ghostty config
