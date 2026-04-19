# Architecture

`ghostty-tui` is a three-layer tool. Understanding the layer contract makes extension, debugging, and forking for other terminals straightforward.

```
┌─────────────────────────────────────────────────────────────────┐
│                         USER INPUT                              │
│                      (keyboard events)                          │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│   Layer 1 — ratatui TUI  (Rust, src/main.rs)                    │
│   ─────────────────────────────────────────                     │
│   • Event loop + crossterm keypress handling                    │
│   • 11 tabs × option lists (Vec<String>)                        │
│   • ListState per tab, separator-skip logic                     │
│   • On Enter: parses option label → emits a command vocabulary  │
│     string (e.g. `theme "Dracula"`, `font-size 14`,              │
│     `screen-shader "crt-classic"`)                              │
└────────────────────────┬────────────────────────────────────────┘
                         │ Command::new(&script).args(&parsed)
                         │ (direct argv — NOT `sh -c`)
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│   Layer 2 — ghostty-manager.sh  (bash, scripts/)                │
│   ─────────────────────────────────────────                     │
│   • ~35 subcommands, one per TUI capability                     │
│   • Platform detection (Darwin/Linux sed flavor)                │
│   • Asset-root resolution                                       │
│       GHOSTTY_TUI_ASSETS → adjacent to script →                 │
│       share/ghostty-tui/assets → XDG_DATA_HOME fallback         │
│   • Translates command vocabulary → Ghostty config-key writes   │
│     via line-append or sed-in-place                             │
│   • Returns status to Layer 1 via stdout/stderr + exit code     │
└────────────────────────┬────────────────────────────────────────┘
                         │ reads + writes (flat-KV config file)
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│   Layer 3 — Ghostty config file                                 │
│   ─────────────────────────────────────────                     │
│   • macOS: ~/Library/Application Support/com.mitchellh.ghostty  │
│            /config                                              │
│   • Linux: $XDG_CONFIG_HOME/ghostty/config                      │
│            or ~/.config/ghostty/config                          │
│   • Windows: %APPDATA%/ghostty/config                           │
│   • Flat-KV format: one `key = value` per line                  │
│   • Ghostty re-reads on: new window, `reload_config` keybind    │
└─────────────────────────────────────────────────────────────────┘
```

## Layer contract

**Layer 1 never touches the config file directly.** This keeps the config-file schema — which evolves with Ghostty — localized in Layer 2. When Ghostty renames `scrollback-limit` → `scrollback-bytes` (hypothetically), only the bash script needs to change; the Rust UI is untouched.

**Layer 2 never knows about ratatui or the TUI.** It's runnable standalone:

```bash
./scripts/ghostty-manager.sh theme Dracula
./scripts/ghostty-manager.sh font-size 14
./scripts/ghostty-manager.sh screen-shader crt-classic
./scripts/ghostty-manager.sh current    # prints current config summary
```

This means the command vocabulary is testable, scriptable, and reusable from other frontends (Alfred workflow, Raycast script, `fzf` wrapper, etc.).

**Layer 3 is the source of truth.** The TUI does not cache Ghostty state — on Enter it writes through Layer 2, then reads the full config back to refresh the Cfg tab.

## Command vocabulary

Layer 1 → Layer 2 protocol. Full list in the `case "$command" in` block at the bottom of `scripts/ghostty-manager.sh`. Sample:

| Vocab | Args | Writes |
|---|---|---|
| `theme` | `<name>` | `theme = <name>` |
| `font-family` | `<family>` | `font-family = <family>` |
| `font-size` | `<pt>` | `font-size = <pt>` |
| `font-weight` | `<style>` | `font-style = <style>` (Regular/Bold/Italic/Bold Italic) |
| `cursor-style` | `block\|bar\|underline\|block_hollow` | `cursor-style = <v>` |
| `cursor-color` | `theme\|#hex` | `cursor-color = <v>` (blank removes) |
| `cursor-opacity` | `<0.0-1.0>` | `cursor-opacity = <v>` |
| `shader` | `<name>` | `custom-shader = <SHADERS_DIR>/<name>.glsl` |
| `screen-shader` | `<combo-or-individual>` | one-or-more `custom-shader = ...` |
| `opacity` | `<0.0-1.0>` | `background-opacity = <v>` |
| `blur` | `<0-40>` | `background-blur = <v>` |
| `current` | — | prints formatted summary to stdout |

## Asset library

Bundled at `assets/`, resolved at runtime by both layers. Layer 1 uses `resolve_assets_dir()` (Rust); Layer 2 uses the `ASSETS_DIR=` block at the top of the script. Both share the same priority chain — explicit env override, adjacent to binary/script, installed layout, XDG fallback.

The bundled assets are also published as plain files in the repo so users can `curl` individual themes/shaders without installing the Rust binary. See `docs/ASSETS.md`.

## What's NOT in this design (by choice)

- **No config-schema awareness in Layer 1.** Rust doesn't know that `theme` writes to the `theme =` key — it just emits the vocabulary. This keeps the TUI agnostic to config-format changes.
- **No async / background work.** Every action is a blocking subprocess call. Ghostty config mutations are fast (<50ms). If latency ever matters, `tokio::process` is a drop-in swap.
- **No persistent state in the TUI.** The ListState positions reset on restart — intentional, since Ghostty's config is authoritative.
- **No live re-render on file change.** Layer 3 edits don't auto-refresh the Cfg tab — press `r` or reopen the tool.

## Extending the tool

Adding a new setting (e.g. a new `Display` tab section):
1. Add an entry to the relevant `Vec<String>` options in `App::new()`
2. Add a `starts_with("...")` branch in `select_item()` that emits a new command vocabulary string
3. Add a new subcommand in `scripts/ghostty-manager.sh` that writes the Ghostty config key
4. Add a unit test in `src/main.rs` for the option parser
5. Add a smoke test in `.github/workflows/ci.yml` if the new key has install-time side effects

For forking this tool to a different terminal emulator, see [`ADOPTING.md`](ADOPTING.md) — the swap checklist is per-layer and mechanical.
