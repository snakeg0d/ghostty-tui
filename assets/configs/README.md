# Config presets

Ten complete Ghostty config presets. Each is a full, standalone config — you can symlink one into place and it becomes your active config.

| Preset | Mood | Use when |
|---|---|---|
| `minimal.config` | Stripped to essentials | You want a clean slate |
| `focus.config` | Low-chrome, distraction-free | Deep-work sessions |
| `presentation.config` | Large font, high contrast | Screen-sharing / demos |
| `agentic.config` | TUI-optimized size, glass aesthetic | AI-assisted dev sessions (Claude Code, Codex, etc.) |
| `tmux-style.config` | Keybind mappings mirror tmux | You're migrating from tmux workflows |
| `aecsar-wide.config` | Ultra-wide window | 34"+ monitors |
| `retro.config` | CRT vibes | Nostalgia mode |
| `gruvbox-monolisa.config` | Gruvbox theme + MonoLisa font | If you've already licensed MonoLisa |
| `zapman-tokyonight.config` | Zapman font + TokyoNight | Quiet reading setup |
| `plutov-cyberdream.config` | Plutov font + Cyberdream palette | Synthwave aesthetic |

## Install one preset

```bash
# Symlink (changes apply immediately; edits in the repo follow through)
ln -sfn ~/path/to/ghostty-tui/assets/configs/minimal.config \
  "$HOME/Library/Application Support/com.mitchellh.ghostty/config"

# Or copy as a static snapshot
cp ~/path/to/ghostty-tui/assets/configs/minimal.config \
  "$HOME/Library/Application Support/com.mitchellh.ghostty/config"
```

On Linux: substitute `~/.config/ghostty/config` for the macOS path.

## Remote install (no clone)

```bash
curl -fsSL https://raw.githubusercontent.com/snakeg0d/ghostty-tui/main/assets/configs/minimal.config \
  -o "$HOME/Library/Application Support/com.mitchellh.ghostty/config"
```

See [`docs/ASSETS.md`](../../docs/ASSETS.md) for a one-liner per preset.

## Notes

- All presets use `working-directory = home` by default — change to an absolute path if you want a different start dir.
- Keybind paths in `agentic.config` are commented examples; uncomment and edit to taste.
- Ghostty reloads config only on new windows — `super+r` forces reload.
- `scrollback-limit` takes bytes; `scrollback-limit = 50000` ≈ 50KB of history.
