# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Cross-platform support for macOS, Linux, and Windows
- Platform-specific config path detection
- `--help` / `-h` CLI flag for usage information
- `--version` / `-v` CLI flag for version display
- `--config-path` CLI flag to show platform-specific config location
- `GHOSTTY_MANAGER_PATH` environment variable support for custom script location
- Comprehensive error handling with user-friendly messages
- Module-level documentation and code organization comments
- Detailed error messages for initialization failures
- Graceful handling of missing config files
- Troubleshooting guide in error output

### Changed
- `App::new()` now returns `Result<Self, String>` for better error handling
- Config path detection is now platform-aware (not hardcoded to macOS)
- Manager script path uses fallback chain instead of hardcoded path
- Error messages now show in status bar instead of panicking
- Improved code organization with section separators

### Fixed
- Removed all hardcoded file paths
- Replaced panic-prone `.unwrap()` calls with proper error handling
- Command execution failures now display helpful error messages

## [0.1.0] - 2025-12-30

### Added
- Initial release
- 11 comprehensive configuration tabs
- 200+ built-in theme support
- Cursor and screen shader effects
- Live config updates
- Keyboard-driven navigation
- TokyoNight color scheme
