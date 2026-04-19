# Themes

Five hand-curated Ghostty theme files. Drop them into `~/.config/ghostty/themes/` (or use `ghostty-tui --install-assets` once Task 11 lands) and reference them in your config with `theme = <name>`.

| Theme | Mood | Palette source |
|---|---|---|
| `onenord` | Dark, cool-tone | Adapted from [onenord.nvim](https://github.com/rmehri01/onenord.nvim) |
| `onenord-light` | Light, soft pastels | Light variant of OneNord |
| `tokyonight-night` | Dark, high-contrast neon | [folke/tokyonight.nvim](https://github.com/folke/tokyonight.nvim) night |
| `tokyonight-storm` | Dark, muted blue-grey | TokyoNight storm |
| `tokyonight-day` | Light, warm-tone | TokyoNight day |

## Install one theme

```bash
# Example: install "onenord"
mkdir -p ~/.config/ghostty/themes
curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/themes/onenord \
  -o ~/.config/ghostty/themes/onenord
```

Then in `~/Library/Application Support/com.mitchellh.ghostty/config` (macOS) or `~/.config/ghostty/config` (Linux):

```
theme = onenord
```

For adaptive dark/light switching:

```
theme = dark:tokyonight-night,light:tokyonight-day
```

## Install all five

See [`docs/ASSETS.md`](../../docs/ASSETS.md) for the full curl block, or run `ghostty-tui --install-assets`.

## Add your own

Any file in `~/.config/ghostty/themes/` becomes referenceable by filename. Ghostty also ships 200+ built-in themes — see `ghostty +list-themes`.
