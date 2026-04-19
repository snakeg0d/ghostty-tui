# Ghostty TUI

A beautiful, keyboard-driven terminal interface for managing [Ghostty](https://ghostty.org) terminal configuration. Navigate through 11 comprehensive tabs to customize themes, fonts, cursors, shaders, and more - all with instant preview in new windows.

## Features

- **11 Comprehensive Configuration Tabs** - Complete control over all Ghostty settings
- **Live Configuration Updates** - Changes apply immediately to new windows
- **200+ Built-in Themes** - Browse and apply themes with live preview
- **Shader Effects** - Apply cursor and screen shaders for visual flair
- **Keyboard-Driven Navigation** - Efficient, mouse-free workflow
- **Cross-Platform** - Works on macOS, Linux, and Windows
- **Configuration Preview** - View your current config file in-app
- **Smart Path Detection** - Automatically finds your Ghostty config

## Installation

### From Source

```bash
git clone https://github.com/snakeg0d/ghostty-tui
cd ghostty-tui
cargo build --release
./target/release/ghostty-tui
```

### Via Cargo (once published to crates.io)

```bash
cargo install ghostty-tui
ghostty-tui
```

## Requirements

- [Ghostty](https://ghostty.org) terminal emulator installed
- Rust 1.75+ (for building from source)
- `ghostty-manager.sh` script (bundled with the project)

## Quick Start

Launch the TUI:
```bash
ghostty-tui
```

**Navigation:**
- `Tab`/`Shift+Tab` - Switch between tabs
- `↑`/`↓` or `j`/`k` - Navigate options
- `Enter` - Select option (applies immediately)
- `r` - Reload current configuration
- `q` or `Esc` - Quit

**Note:** Changes apply to new windows/tabs only. Open a new Ghostty window to see your updates.

## Configuration Tabs

1. **Theme** - Browse and apply 200+ built-in themes
2. **Font** - Font family, size, weight, style, ligatures
3. **Cursor** - Cursor style, color, blink rate, opacity
4. **CFX (Cursor Effects)** - Shader effects for cursor (glow, trail, pulse)
5. **SFX (Screen Effects)** - Full-screen shaders (CRT, bloom, glitch, etc.)
6. **Look** - Window opacity, blur, padding, title bar
7. **Mouse** - Mouse behavior, scroll speed, copy/paste settings
8. **Clip** - Clipboard permissions and integration
9. **Win** - Window management, size, decorations, tabs
10. **Shell** - Shell integration, working directory, scrollback
11. **Cfg** - View current configuration file

## Screenshots

*Coming soon - screenshots and demo GIFs will be added*

## Environment Variables

- `GHOSTTY_MANAGER_PATH` - Override path to ghostty-manager.sh script
- `GHOSTTY_CONFIG_PATH` - Override default Ghostty config location

## Platform Support

- **macOS** - Full support (primary development platform)
- **Linux** - Full support (XDG Base Directory spec respected)
- **Windows** - Implemented (community testing welcome)

## Development

### Building
```bash
cargo build
```

### Running Tests
```bash
cargo test
```

### Code Quality
```bash
cargo clippy -- -D warnings
cargo fmt --check
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history and release notes.

## License

This project is licensed under the MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- Built for [Ghostty](https://ghostty.org) by Mitchell Hashimoto
- Uses [Ratatui](https://github.com/ratatui-org/ratatui) for the terminal UI
- Inspired by the need for a keyboard-driven configuration experience

## Related Projects

- [Ghostty](https://ghostty.org) - The Ghostty terminal emulator
- [Spectre](https://spectre-ghostty-config.vercel.app/) - Visual Ghostty configuration editor

---

Built with care for the terminal-native workflow.
