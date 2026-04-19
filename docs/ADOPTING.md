# Adopting ghostty-tui for another terminal

**Audience:** AI coding agents (Claude Code, Codex, Aider, etc.) asked to fork this tool for **Alacritty**, **Kitty**, **WezTerm**, **foot**, or any other terminal emulator with a file-based config. Humans can follow the same checklist.

This document is structured so an agent can run it in a single plan-mode session without additional context from the operator.

---

## 1. What this tool is

A three-layer TUI that writes terminal config:

- **Layer 1 (Rust/ratatui):** tab-and-list UI, emits a command vocabulary
- **Layer 2 (bash script):** translates vocabulary to config-file writes
- **Layer 3 (config file):** the terminal's on-disk config

Full diagram: [`ARCHITECTURE.md`](ARCHITECTURE.md).

---

## 2. Layer map — what to copy, what to rewrite

| Component | Disposition for a fork |
|---|---|
| `src/main.rs` event loop (crossterm keypress → App state) | **Copy verbatim.** |
| `src/main.rs` ratatui styling + TokyoNight palette | **Copy verbatim** (users can re-theme later). |
| `src/main.rs` `get_<terminal>_config_path()` per-OS cfg blocks | **Rewrite** — each terminal has its own platform paths. |
| `src/main.rs` tab list + option `Vec<String>`s per tab | **Rewrite** — match the target terminal's capability surface. Drop tabs that don't apply; add ones that do (e.g. Alacritty has mouse-bindings that Ghostty handles differently). |
| `src/main.rs` `select_item()` parser chains (`starts_with("…")` branches) | **Rewrite** in lockstep with the option lists. |
| `src/main.rs` separator-skip logic in `next_item` / `previous_item` | **Copy verbatim** — it's generic. |
| `src/main.rs` `resolve_assets_dir()` + install logic | **Copy verbatim.** |
| `src/main.rs` CLI flags (`--help`, `--version`, `--install-assets`, `--config-path`, `--assets-path`) | **Copy verbatim** — just update the help text strings. |
| `scripts/ghostty-manager.sh` | **Rewrite entirely** as `scripts/<terminal>-manager.sh`. This is where the terminal-specific schema lives. |
| `assets/themes/`, `assets/shaders/`, `assets/configs/` | **Replace** — source themes/shaders/configs native to the target terminal. |
| `.github/workflows/ci.yml` | **Copy verbatim** — rename binary paths. |
| `tests/cli.rs` | **Copy verbatim** — adjust expected asset counts to match new bundle. |
| `Cargo.toml` | Rename `name`, `description`, `repository`, `keywords`. Keep deps. |
| `README.md`, `CONTRIBUTING.md`, `docs/ASSETS.md` | **Rewrite** with fork's identity. |
| `LICENSE` | Keep MIT + original attribution to `snakeg0d/ghostty-tui`. |

---

## 3. Swap checklist — one PR per step

Ordered. Each step ends with a passing `cargo test` + a single commit.

1. **Fork the repo.** `gh repo fork snakeg0d/ghostty-tui <terminal>-tui` — or clone + init a fresh repo and copy the files.
2. **Rename.** In `Cargo.toml`: `name = "<terminal>-tui"`; update `description`, `repository`, `homepage`, `keywords`. Build should still succeed (only names changed).
3. **Swap Layer 3 path.** In `src/main.rs`, replace `get_ghostty_config_path()` with `get_<terminal>_config_path()`. Preserve the per-OS `#[cfg]` structure; only the paths change.
4. **Swap Layer 2 (manager script).** Rewrite `scripts/<terminal>-manager.sh` command-by-command. Section 4 below has the full schema. Verify via `bash -n` + a manual smoke test against a throwaway config.
5. **Swap Layer 1 option lists.** In `App::new()`: rewrite each `Vec<String>` to reflect target terminal's capabilities. Delete tabs that don't map; add ones that do.
6. **Swap `select_item()` parsers.** Update the `starts_with(…)` chains to match new option labels and emit the new command vocabulary.
7. **Regenerate asset bundle.** Replace `assets/themes/`, `assets/shaders/`, `assets/configs/` with target-native content. Update `assets/**/README.md`.
8. **Update `--help` text.** In `print_help()`: new binary name, new keybinds section, new `RELOAD <TERMINAL>` section.
9. **Update tests.** `tests/cli.rs` asset counts; inline unit tests' option-parser cases.
10. **Update CI.** `.github/workflows/ci.yml`: binary name, install-assets counts, bash syntax check for the new manager script.
11. **Rewrite docs.** README + `docs/ASSETS.md` URLs point to the new repo. `docs/ARCHITECTURE.md` can be mostly reused; update the terminal name.
12. **Ship.** Tag `v0.1.0`, `gh release create` with the built ZIP.

---

## 4. Command vocabulary schema

This is the Layer 1 → Layer 2 protocol. An agent should read this table + the target terminal's config reference and emit a new `<terminal>-manager.sh` by translating each entry.

```yaml
# command_vocabulary: the strings Layer 1 emits to Layer 2.
# Each entry specifies the args it takes and the config-key
# it should produce on the target terminal's config file.
commands:
  - name: theme
    args: [theme_name]
    target_key: theme        # Ghostty
    # Alacritty: import "<path>"; Kitty: include <path>; WezTerm: Lua call.

  - name: current
    args: []
    action: print_summary    # not a write — prints formatted stdout

  - name: font-family
    args: [family_name]
    target_key: font-family

  - name: font-size
    args: [size_pt]
    target_key: font-size

  - name: font-weight           # legacy name; writes font-style on Ghostty
    args: [style_name]          # Regular | Bold | Italic | Bold Italic
    target_key: font-style

  - name: ligatures
    args: [on_or_off]
    target_key: font-feature    # on -> empty; off -> `-calt`

  - name: cursor-style
    args: [block | bar | underline | block_hollow]
    target_key: cursor-style

  - name: cursor-color
    args: [theme | "#hex"]
    target_key: cursor-color    # empty string removes the key

  - name: cursor-blink
    args: [true | false]
    target_key: cursor-style-blink

  - name: cursor-opacity
    args: [0.0 - 1.0]
    target_key: cursor-opacity

  - name: shader                # cursor shader
    args: [shader_name]
    target_key: custom-shader   # writes SHADERS_DIR/<name>.glsl

  - name: shader-off
    args: []
    action: remove_keys         # strips custom-shader lines

  - name: screen-shader
    args: [combo_or_name]
    target_key: custom-shader   # may write multiple lines for combos

  - name: screen-shader-off
    args: []
    action: remove_keys

  - name: opacity
    args: [0.0 - 1.0]
    target_key: background-opacity

  - name: blur
    args: [0 - 40]
    target_key: background-blur

  - name: padding-x
    args: [px]
    target_key: window-padding-x

  - name: padding-y
    args: [px]
    target_key: window-padding-y

  - name: mouse-hide
    args: [true | false]
    target_key: mouse-hide-while-typing

  - name: scroll-multiplier
    args: [float]
    target_key: mouse-scroll-multiplier

  - name: copy-on-select
    args: [true | false | clipboard]
    target_key: copy-on-select

  - name: focus-follows-mouse
    args: [true | false]
    target_key: focus-follows-mouse

  - name: clipboard-read
    args: [ask | allow | deny]
    target_key: clipboard-read

  - name: clipboard-write
    args: [ask | allow | deny]
    target_key: clipboard-write

  - name: clipboard-trim
    args: [true | false]
    target_key: clipboard-trim-trailing-spaces

  - name: paste-protection
    args: [true | false]
    target_key: clipboard-paste-protection

  - name: size
    args: [small | medium | large | ultrawide | custom-name]
    # writes both window-width and window-height

  - name: window-decoration
    args: [auto | none | client | server]
    target_key: window-decoration

  - name: tab-bar
    args: [auto | top | hidden]
    target_key: gtk-tabs-location   # Ghostty/Linux; other terminals differ

  - name: vsync
    args: [true | false]
    target_key: window-vsync

  - name: fullscreen
    args: [true | false]
    target_key: fullscreen

  - name: save-state
    args: [default | always | never]
    target_key: window-save-state

  - name: shell-integration
    args: [detect | bash | fish | zsh | elvish | none]
    target_key: shell-integration

  - name: working-directory
    args: [home | inherit | <path>]
    target_key: working-directory

  - name: confirm-close
    args: [true | false | always]
    target_key: confirm-close-surface

  - name: scrollback
    args: [bytes]
    target_key: scrollback-limit

  - name: appearance
    args: [default | solid]
    # macro — writes multiple keys (opacity + blur presets)
```

---

## 5. Terminal-specific pitfalls

| Terminal | Config format | Gotcha |
|---|---|---|
| **Alacritty** | TOML | Use `toml_edit` crate — hand-edited TOML breaks comments. Themes are imported via `[general] import = [...]`, not a `theme =` key. |
| **Kitty** | flat-KV | Very similar to Ghostty — `sed -i` works. Theme via `include` directive. `kitty +kitten themes --reload-in=all` is the reload mechanism. |
| **WezTerm** | Lua | Line-append sed will NOT work. Either generate a `<terminal>-tui.lua` helper file that user's `wezterm.lua` `require`s, or use a Lua parser. |
| **foot** | INI | Section headers matter — use the `ini` crate or careful section-aware sed. |
| **iTerm2** | binary plist | Much harder — use `plutil` or `defaults` commands, not text edits. |
| **Windows Terminal** | JSON | Use `serde_json` + field-merge; preserve comments with `json5` or `jsonc`. |

---

## 6. Testing a fork

After each swap step, `cargo test` should still pass. The inline unit tests and CLI integration tests transfer with minimal changes:

- **Inline tests** (`src/main.rs`): update option-parser test cases to the new label/vocabulary shape. Keep the precedence tests (e.g. "Bold Italic" before "Italic") — these are universal gotchas.
- **CLI tests** (`tests/cli.rs`): update expected asset counts in `install_assets_copies_into_test_home`. Keep the idempotence check.

New tests worth adding per-fork:
- Config-format round-trip test — write a key, parse it back, verify value
- Reload-mechanism test if the target terminal has one

---

## 7. Publishing a fork

1. Keep MIT attribution to `snakeg0d/ghostty-tui` in `LICENSE`
2. Credit the parent project in your README
3. Use the same halt-gate discipline from the original execution plan — don't push until tests pass
4. Tag v0.1.0 for the first fork release; follow semver after

---

## 8. Upstreaming back

If your fork discovers an improvement to the core architecture (not terminal-specific), consider a PR back to `snakeg0d/ghostty-tui`. Good candidates:

- New CLI flags that are terminal-agnostic
- Improvements to `resolve_assets_dir()` fallback chain
- Additional inline unit test patterns
- Shared `docs/ADOPTING.md` refinements based on what you learned

Terminal-specific changes stay in your fork — we don't want the upstream to carry Alacritty-specific logic.

---

## 9. Agent execution hints

When running this checklist autonomously:

- **One commit per step.** Atomic, reversible. Halt on test failure.
- **Use plan mode** to scaffold the full sequence before touching files — this doc is meant to hydrate a plan, not replace one.
- **Verify each Layer-2 subcommand** with a throwaway config file (`GHOSTTY_CONFIG_PATH=/tmp/test.conf ./scripts/<terminal>-manager.sh <cmd>`) before wiring Layer 1.
- **Don't touch `src/main.rs` event loop** unless you find a bug — it's generic across terminals. Any diff there deserves a separate commit.
- **Scrub for leaked user paths** before publishing — reuse the CI `grep -rn "/Users/[a-z]"` check.
