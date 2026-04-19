# Ghostty TUI - Quick Start Guide

## Installation

### From Source
```bash
git clone https://github.com/yourusername/ghostty-tui
cd ghostty-tui
cargo build --release
./target/release/ghostty-tui
```

### Via Cargo (once published)
```bash
cargo install ghostty-tui
ghostty-tui
```

## Requirements

1. **Ghostty terminal** installed and configured
2. **ghostty-manager.sh** script accessible
   - Set `GHOSTTY_MANAGER_PATH` environment variable to script location
   - Or place in same directory as your Ghostty config

## First Run

```bash
# Check your config path
ghostty-tui --config-path

# Verify manager script can be found
ghostty-tui

# If it fails, set the manager script path:
export GHOSTTY_MANAGER_PATH="/path/to/ghostty-manager.sh"
ghostty-tui
```

## Basic Usage

### Launch the TUI
```bash
ghostty-tui
```

### Navigation
- **Tab/Shift+Tab**: Switch between tabs
- **↑↓ or j/k**: Navigate options
- **Enter**: Select option (applies immediately)
- **r**: Reload current config
- **q or Esc**: Quit

### Available Tabs

1. **Theme** - 200+ built-in themes
2. **Font** - Font family, size, weight, ligatures
3. **Cursor** - Style, color, blink, opacity
4. **CFX** - Cursor effects/shaders
5. **SFX** - Screen effects (CRT, bloom, glitch, etc.)
6. **Look** - Opacity, blur, padding
7. **Mouse** - Hide while typing, scroll speed, copy behavior
8. **Clip** - Clipboard read/write permissions
9. **Win** - Window size, decorations, tab bar
10. **Shell** - Integration, working directory, scrollback
11. **Cfg** - View current configuration

## Environment Variables

### TUI Application
- `GHOSTTY_MANAGER_PATH` - Path to ghostty-manager.sh script

### Manager Script
- `GHOSTTY_CONFIG_PATH` - Override default config location

## Platform Support

- **macOS**: Full support (primary platform)
- **Linux**: Full support (XDG paths respected)
- **Windows**: Implemented (not yet tested)

## Common Issues

### "Could not find ghostty-manager.sh"
```bash
# Solution: Set the environment variable
export GHOSTTY_MANAGER_PATH="/path/to/ghostty-manager.sh"

# Or add to ~/.zshrc or ~/.bashrc:
echo 'export GHOSTTY_MANAGER_PATH="$HOME/dev/dotfiles/ghostty/ghostty-manager.sh"' >> ~/.zshrc
```

### Changes don't appear
Changes apply to **new windows/tabs only**. Open a new Ghostty window to see updates.

### Want to see what changed?
Switch to the **Cfg** tab (11) to view the current configuration file.

## Tips

1. **Experiment freely** - Changes apply immediately but only to new windows
2. **Use the Cfg tab** - See exactly what's in your config file
3. **Press 'r' to reload** - If you manually edit the config file
4. **Check version** - `ghostty-tui --version` to see current version
5. **Get help** - `ghostty-tui --help` for all options

## Getting Help

- GitHub Issues: https://github.com/yourusername/ghostty-tui/issues
- Documentation: https://github.com/yourusername/ghostty-tui/wiki

## Next Steps

- Explore all 11 tabs
- Try different themes
- Experiment with shaders
- Customize your perfect terminal setup!
