# Asset library — install individual files

Every theme, shader, and config preset bundled with `ghostty-tui` is independently downloadable from the repo. Pick only what you want.

Three ways to install:
1. **Bulk via the TUI:** `ghostty-tui --install-assets` — copies every bundled asset to `~/.config/ghostty/{themes,shaders}/`
2. **One-liner curl** — see tables below, copy the line for the asset you want
3. **Clone + symlink** — `git clone https://github.com/snakeg0d/ghostty-tui && ln -sfn ghostty-tui/assets/themes/onenord ~/.config/ghostty/themes/onenord`

Base URL for all raw assets:

```
https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/
```

---

## Themes (5)

Drop into `~/.config/ghostty/themes/<name>` then reference in config with `theme = <name>`.

| Theme | One-liner |
|---|---|
| onenord | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/themes/onenord -o ~/.config/ghostty/themes/onenord` |
| onenord-light | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/themes/onenord-light -o ~/.config/ghostty/themes/onenord-light` |
| tokyonight-night | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/themes/tokyonight-night -o ~/.config/ghostty/themes/tokyonight-night` |
| tokyonight-storm | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/themes/tokyonight-storm -o ~/.config/ghostty/themes/tokyonight-storm` |
| tokyonight-day | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/themes/tokyonight-day -o ~/.config/ghostty/themes/tokyonight-day` |

Install all five at once:

```bash
mkdir -p ~/.config/ghostty/themes
for theme in onenord onenord-light tokyonight-night tokyonight-storm tokyonight-day; do
  curl -fsSL "https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/themes/$theme" \
    -o "$HOME/.config/ghostty/themes/$theme"
done
```

---

## Cursor shaders (9)

Drop into `~/.config/ghostty/shaders/cursors/<name>.glsl` then reference with `custom-shader = ~/.config/ghostty/shaders/cursors/<name>.glsl`.

| Shader | One-liner |
|---|---|
| glow | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/cursors/glow.glsl -o ~/.config/ghostty/shaders/cursors/glow.glsl` |
| cursor_tail | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/cursors/cursor_tail.glsl -o ~/.config/ghostty/shaders/cursors/cursor_tail.glsl` |
| cursor_sweep | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/cursors/cursor_sweep.glsl -o ~/.config/ghostty/shaders/cursors/cursor_sweep.glsl` |
| cursor_warp | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/cursors/cursor_warp.glsl -o ~/.config/ghostty/shaders/cursors/cursor_warp.glsl` |
| ripple_cursor | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/cursors/ripple_cursor.glsl -o ~/.config/ghostty/shaders/cursors/ripple_cursor.glsl` |
| ripple_rectangle_cursor | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/cursors/ripple_rectangle_cursor.glsl -o ~/.config/ghostty/shaders/cursors/ripple_rectangle_cursor.glsl` |
| rectangle_boom_cursor | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/cursors/rectangle_boom_cursor.glsl -o ~/.config/ghostty/shaders/cursors/rectangle_boom_cursor.glsl` |
| sonic_boom_cursor | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/cursors/sonic_boom_cursor.glsl -o ~/.config/ghostty/shaders/cursors/sonic_boom_cursor.glsl` |
| tesla_coil | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/cursors/tesla_coil.glsl -o ~/.config/ghostty/shaders/cursors/tesla_coil.glsl` |

Install all nine:

```bash
mkdir -p ~/.config/ghostty/shaders/cursors
for s in glow cursor_tail cursor_sweep cursor_warp ripple_cursor \
         ripple_rectangle_cursor rectangle_boom_cursor sonic_boom_cursor tesla_coil; do
  curl -fsSL "https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/cursors/$s.glsl" \
    -o "$HOME/.config/ghostty/shaders/cursors/$s.glsl"
done
```

---

## Screen shaders (12)

Drop into `~/.config/ghostty/shaders/screen/<name>.glsl`. Multiple stack — order matters.

| Shader | One-liner |
|---|---|
| bettercrt | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/screen/bettercrt.glsl -o ~/.config/ghostty/shaders/screen/bettercrt.glsl` |
| tft | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/screen/tft.glsl -o ~/.config/ghostty/shaders/screen/tft.glsl` |
| retro-terminal | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/screen/retro-terminal.glsl -o ~/.config/ghostty/shaders/screen/retro-terminal.glsl` |
| amber-glow | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/screen/amber-glow.glsl -o ~/.config/ghostty/shaders/screen/amber-glow.glsl` |
| bloom | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/screen/bloom.glsl -o ~/.config/ghostty/shaders/screen/bloom.glsl` |
| glitchy | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/screen/glitchy.glsl -o ~/.config/ghostty/shaders/screen/glitchy.glsl` |
| drunkard | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/screen/drunkard.glsl -o ~/.config/ghostty/shaders/screen/drunkard.glsl` |
| water | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/screen/water.glsl -o ~/.config/ghostty/shaders/screen/water.glsl` |
| starfield | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/screen/starfield.glsl -o ~/.config/ghostty/shaders/screen/starfield.glsl` |
| matrix-hallway | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/screen/matrix-hallway.glsl -o ~/.config/ghostty/shaders/screen/matrix-hallway.glsl` |
| inside-the-matrix | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/screen/inside-the-matrix.glsl -o ~/.config/ghostty/shaders/screen/inside-the-matrix.glsl` |
| fireworks | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/screen/fireworks.glsl -o ~/.config/ghostty/shaders/screen/fireworks.glsl` |

Install all twelve:

```bash
mkdir -p ~/.config/ghostty/shaders/screen
for s in bettercrt tft retro-terminal amber-glow bloom glitchy drunkard water \
         starfield matrix-hallway inside-the-matrix fireworks; do
  curl -fsSL "https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/screen/$s.glsl" \
    -o "$HOME/.config/ghostty/shaders/screen/$s.glsl"
done
```

---

## Config presets (10)

Full ghostty configs — each is a complete replacement for your active config. Install by copying to your active config location or by symlinking.

**macOS active config:** `~/Library/Application Support/com.mitchellh.ghostty/config`
**Linux active config:** `~/.config/ghostty/config`

| Preset | One-liner (macOS) |
|---|---|
| minimal | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/configs/minimal.config -o "$HOME/Library/Application Support/com.mitchellh.ghostty/config"` |
| focus | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/configs/focus.config -o "$HOME/Library/Application Support/com.mitchellh.ghostty/config"` |
| presentation | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/configs/presentation.config -o "$HOME/Library/Application Support/com.mitchellh.ghostty/config"` |
| agentic | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/configs/agentic.config -o "$HOME/Library/Application Support/com.mitchellh.ghostty/config"` |
| tmux-style | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/configs/tmux-style.config -o "$HOME/Library/Application Support/com.mitchellh.ghostty/config"` |
| aecsar-wide | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/configs/aecsar-wide.config -o "$HOME/Library/Application Support/com.mitchellh.ghostty/config"` |
| retro | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/configs/retro.config -o "$HOME/Library/Application Support/com.mitchellh.ghostty/config"` |
| gruvbox-monolisa | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/configs/gruvbox-monolisa.config -o "$HOME/Library/Application Support/com.mitchellh.ghostty/config"` |
| zapman-tokyonight | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/configs/zapman-tokyonight.config -o "$HOME/Library/Application Support/com.mitchellh.ghostty/config"` |
| plutov-cyberdream | `curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/configs/plutov-cyberdream.config -o "$HOME/Library/Application Support/com.mitchellh.ghostty/config"` |

> ⚠️ Installing a preset replaces your entire config. Back up first:
> `cp "$HOME/Library/Application Support/com.mitchellh.ghostty/config" ~/ghostty.config.bak`

---

## Bulk install

Once `ghostty-tui` v0.2.0 is installed, a single subcommand installs every theme + shader to your user config dir:

```bash
ghostty-tui --install-assets
```

Effect:
- Copies `assets/themes/*` → `~/.config/ghostty/themes/`
- Copies `assets/shaders/cursors/*` → `~/.config/ghostty/shaders/cursors/`
- Copies `assets/shaders/screen/*` → `~/.config/ghostty/shaders/screen/`
- Does **not** overwrite existing config presets — configs are always opt-in per-file.

---

## Reload after install

Ghostty reloads config on new windows, or explicitly via `super+r` (`cmd+r` on macOS by default). Your current window keeps its old config until you reload or open a new one.
