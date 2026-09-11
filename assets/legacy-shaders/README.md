# Ghostty Shaders

> Historical README recovered with the old app. Commands, paths, option names, and performance claims below have not been revalidated and may be obsolete. Use the [current Studio guide](../../README.md) and Ghostty's installed reference. The 21 GLSL files are retained byte for byte; Studio does not automatically enable them or claim they have been renderer-tested.

GPU-accelerated visual effects for Ghostty terminal.

## Directory Structure

```
shaders/
├── cursors/     # Cursor-specific effects (9 shaders)
└── screen/      # Full-screen effects (10 shaders)
```

## Cursor Shaders

Apply to cursor only. Use: `cursor-style-shader = path/to/shader.glsl`

| Shader | Effect | Vibe |
|--------|--------|------|
| `ripple_cursor.glsl` | Ripple on movement | Calm water |
| `sonic_boom_cursor.glsl` | Sonic boom | Explosive |
| `cursor_tail.glsl` | Trailing effect | Comet trail |
| `cursor_sweep.glsl` | Sweep animation | Clean |
| `glow.glsl` | Glowing cursor | Neon |
| `tesla_coil.glsl` | Tesla coil sparks | Electric |
| `cursor_warp.glsl` | Warp effect | Sci-fi |
| `rectangle_boom_cursor.glsl` | Rectangle boom | Geometric |
| `ripple_rectangle_cursor.glsl` | Ripple rectangle | Modern |

**Sources:**
- https://github.com/p424p424/config_ghostty
- https://github.com/sahaj-b/ghostty-cursor-shaders

## Screen Shaders

Apply to entire terminal. Use: `custom-shader = path/to/shader.glsl`

Can **stack multiple shaders** for combined effects!

### Individual Shaders

| Shader | Effect | Source |
|--------|--------|--------|
| `bettercrt.glsl` | CRT monitor (curvature + scanlines) | 0xhckr/ghostty-shaders |
| `tft.glsl` | TFT display effect | 0xhckr/ghostty-shaders |
| `bloom.glsl` | Bloom/glow effect | 0xhckr/ghostty-shaders |
| `retro-terminal.glsl` | Retro terminal aesthetic | 0xhckr/ghostty-shaders |
| `glitchy.glsl` | Glitch/distortion | 0xhckr/ghostty-shaders |
| `drunkard.glsl` | Drunk wobble | 0xhckr/ghostty-shaders |
| `water.glsl` | Water ripple | 0xhckr/ghostty-shaders |
| `matrix-hallway.glsl` | Matrix hallway | 0xhckr/ghostty-shaders |
| `inside-the-matrix.glsl` | Inside the Matrix | 0xhckr/ghostty-shaders |
| `starfield.glsl` | Starfield background | 0xhckr/ghostty-shaders |

### Preset Combos

From [catskull.net](https://catskull.net/fun-with-ghostty-shaders.html):

**crt-classic** (subtle retro):
```bash
~/dev/dotfiles/ghostty/ghostty-manager.sh screen-shader crt-classic
```

**crt-bloom** (retro + glow):
```bash
~/dev/dotfiles/ghostty/ghostty-manager.sh screen-shader crt-bloom
```

**drunk-retro** (wobbly retro):
```bash
~/dev/dotfiles/ghostty/ghostty-manager.sh screen-shader drunk-retro
```

**glitch-water** (chaotic):
```bash
~/dev/dotfiles/ghostty/ghostty-manager.sh screen-shader glitch-water
```

## Management

Use the ghostty-manager script:

```bash
# List available shaders
~/dev/dotfiles/ghostty/ghostty-manager.sh list-shaders
~/dev/dotfiles/ghostty/ghostty-manager.sh list-screen-shaders

# Enable cursor shader
~/dev/dotfiles/ghostty/ghostty-manager.sh shader ripple_cursor

# Enable screen shader (single)
~/dev/dotfiles/ghostty/ghostty-manager.sh screen-shader bettercrt

# Enable screen shader (combo)
~/dev/dotfiles/ghostty/ghostty-manager.sh screen-shader crt-classic

# Disable
~/dev/dotfiles/ghostty/ghostty-manager.sh shader-off
~/dev/dotfiles/ghostty/ghostty-manager.sh screen-shader-off
```

## Performance

- All shaders run on GPU (GLSL)
- Negligible performance impact on modern hardware
- Can disable with `custom-shader-animation = false`

## Credits

- **0xhckr** - https://github.com/0xhckr/ghostty-shaders
- **catskull** - https://catskull.net/fun-with-ghostty-shaders.html
- **p424p424** - https://github.com/p424p424/config_ghostty
- **sahaj-b** - https://github.com/sahaj-b/ghostty-cursor-shaders
