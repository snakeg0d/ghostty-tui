# Contributing to Ghostty TUI

Thank you for your interest in contributing to Ghostty TUI! This document provides guidelines for contributing to the project.

## Code of Conduct

Be respectful, constructive, and kind to other contributors. We want this to be a welcoming community for everyone.

## How to Contribute

### Reporting Bugs

Before creating a bug report:
1. Check existing issues to avoid duplicates
2. Use the bug report template
3. Include your environment details (OS, Ghostty version, etc.)
4. Provide clear reproduction steps

### Suggesting Features

We welcome feature suggestions! Please:
1. Check if the feature has already been requested
2. Describe the use case and benefit
3. Consider implementation complexity
4. Be open to discussion and alternatives

### Contributing Code

1. **Fork the repository**
   ```bash
   git clone https://github.com/snakeg0d/ghostty-tui
   cd ghostty-tui
   ```

2. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes**
   - Write clear, concise code
   - Follow existing code style
   - Add comments for complex logic
   - Update documentation if needed

4. **Test your changes**
   ```bash
   cargo build
   cargo test
   cargo clippy -- -D warnings
   cargo fmt --check
   ```

5. **Commit your changes**
   ```bash
   git add .
   git commit -m "Brief description of changes"
   ```

6. **Push to your fork**
   ```bash
   git push origin feature/your-feature-name
   ```

7. **Submit a Pull Request**
   - Provide a clear description of changes
   - Reference related issues
   - Explain testing performed

## Code Style

- **Formatting**: Run `cargo fmt` before committing
- **Linting**: Fix all `cargo clippy` warnings
- **Naming**: Use descriptive variable and function names
- **Comments**: Add comments for non-obvious code
- **Error Handling**: Use `Result` types, avoid `unwrap()` in production code

## Testing Requirements

- Add unit tests for new functionality
- Verify changes work on your platform
- Note any platform-specific behavior
- Test both success and error cases

## Documentation

- Update README.md if adding user-facing features
- Add doc comments (`///`) for public APIs
- Update CHANGELOG.md for notable changes
- Keep inline comments current

## Platform Support

Ghostty TUI targets:
- **macOS** (primary platform)
- **Linux** (full support)
- **Windows** (implemented, testing welcome)

Please test on your platform and note any platform-specific issues.

## Development Setup

### Prerequisites
- Rust 1.75+ (install via [rustup](https://rustup.rs/))
- Ghostty terminal emulator (1.3.x tested)
- `ghostty-manager.sh` script — bundled under `scripts/`

### Building
```bash
cargo build --release
./target/release/ghostty-tui
```

### Testing
```bash
# Run all tests
cargo test

# Run specific test
cargo test test_name

# Run with verbose output
cargo test -- --nocapture
```

### Debugging
```bash
# Enable debug logging
RUST_LOG=debug cargo run
```

## Project Structure

```
ghostty-tui/
├── src/
│   ├── main.rs           # Entry point
│   ├── config.rs         # Config file handling
│   ├── backend.rs        # Backend script execution
│   └── ui/               # UI components
├── tests/                # Integration tests
├── Cargo.toml            # Dependencies
└── README.md             # User documentation
```

## Questions?

If you have questions about contributing:
- Open a GitHub Discussion
- Ask in an issue
- Check existing documentation

We appreciate all contributions, from bug reports to code improvements!
