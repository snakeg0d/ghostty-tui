# Shaders

GLSL shaders for Ghostty's `custom-shader` setting. Nine cursor-effect shaders and twelve full-screen effect shaders, bundled as the default set.

Ghostty accepts multiple stacked `custom-shader` lines — order matters. Set `custom-shader-animation = true` for animated effects.

## Cursor effects (`shaders/cursors/`)

| File | Effect |
|---|---|
| `glow.glsl` | Soft glow aura around the cursor |
| `cursor_tail.glsl` | Motion-trail following rapid movement |
| `cursor_sweep.glsl` | Sweep highlight on cursor move |
| `cursor_warp.glsl` | Warp-in/out animation |
| `ripple_cursor.glsl` | Water-ripple on cursor blink |
| `ripple_rectangle_cursor.glsl` | Rectangular-ripple variant |
| `rectangle_boom_cursor.glsl` | Boom-burst rectangular |
| `sonic_boom_cursor.glsl` | Sonic-boom radial burst |
| `tesla_coil.glsl` | Tesla-coil arc around cursor |

## Screen effects (`shaders/screen/`)

| File | Effect |
|---|---|
| `bettercrt.glsl` | Classic CRT scanlines + barrel distortion |
| `tft.glsl` | Flat-panel TFT subpixel simulation |
| `retro-terminal.glsl` | Phosphor-green retro terminal |
| `amber-glow.glsl` | Amber-monochrome glow |
| `bloom.glsl` | Soft bloom post-process |
| `glitchy.glsl` | Digital-glitch displacement |
| `drunkard.glsl` | Wavy sway animation |
| `water.glsl` | Underwater wobble |
| `starfield.glsl` | Moving starfield background |
| `matrix-hallway.glsl` | Matrix-style falling characters (hallway depth) |
| `inside-the-matrix.glsl` | Immersive matrix effect |
| `fireworks.glsl` | Burst-pattern fireworks |

## Install one shader

```bash
# Example: install "glow" cursor shader
mkdir -p ~/.config/ghostty/shaders/cursors
curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/shaders/cursors/glow.glsl \
  -o ~/.config/ghostty/shaders/cursors/glow.glsl
```

Then in your ghostty config:

```
custom-shader = ~/.config/ghostty/shaders/cursors/glow.glsl
custom-shader-animation = true
```

## Install all 21 shaders

See [`docs/ASSETS.md`](../../docs/ASSETS.md) or run `ghostty-tui --install-assets`.

## Stacking shaders

Multiple `custom-shader` lines stack in order. Example CRT preset:

```
custom-shader = ~/.config/ghostty/shaders/screen/tft.glsl
custom-shader = ~/.config/ghostty/shaders/screen/retro-terminal.glsl
custom-shader = ~/.config/ghostty/shaders/screen/bloom.glsl
custom-shader-animation = true
```
