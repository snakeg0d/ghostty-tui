//! Ghostty TUI - Terminal UI for managing Ghostty configuration
//!
//! A beautiful, keyboard-driven interface for configuring the Ghostty terminal.
//! Supports themes, fonts, cursor settings, shaders, and more.
//!
//! # Features
//! - 11 comprehensive configuration tabs
//! - Live config updates (apply on select)
//! - 200+ built-in themes
//! - Cursor and screen shader effects
//! - Cross-platform support (macOS, Linux, Windows)
//!
//! # Usage
//! ```
//! ghostty-tui              # Launch interactive TUI
//! ghostty-tui --help       # Show help
//! ghostty-tui --version    # Show version
//! ```

use crossterm::{
    event::{self, Event, KeyCode, KeyEventKind},
    execute,
    terminal::{disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen},
};
use ratatui::{
    backend::CrosstermBackend,
    layout::{Alignment, Constraint, Direction, Layout, Rect},
    style::{Color, Style, Stylize},
    text::{Line, Span, Text},
    widgets::{Block, Borders, List, ListItem, ListState, Paragraph, Wrap},
    Frame, Terminal,
};
use std::{env, io, path::PathBuf, process::Command};

// ============================================================================
// Constants
// ============================================================================

/// Application version from Cargo.toml
const VERSION: &str = env!("CARGO_PKG_VERSION");

/// TokyoNight color palette
const BLUE: Color = Color::Rgb(122, 162, 247);      // Primary borders/headers
const MAGENTA: Color = Color::Rgb(187, 154, 247);   // Highlights
const YELLOW: Color = Color::Rgb(224, 175, 104);    // Status messages
const BG: Color = Color::Rgb(26, 27, 38);           // Background
const GRAY: Color = Color::Rgb(86, 95, 137);        // Inactive elements

// ============================================================================
// Platform-Specific Configuration
// ============================================================================

/// Get the platform-specific Ghostty config path
fn get_ghostty_config_path() -> Result<PathBuf, String> {
    #[cfg(target_os = "macos")]
    {
        let home = env::var("HOME").map_err(|_| "HOME environment variable not set")?;
        Ok(PathBuf::from(home).join("Library/Application Support/com.mitchellh.ghostty/config"))
    }

    #[cfg(target_os = "linux")]
    {
        if let Ok(xdg_config) = env::var("XDG_CONFIG_HOME") {
            Ok(PathBuf::from(xdg_config).join("ghostty/config"))
        } else {
            let home = env::var("HOME").map_err(|_| "HOME environment variable not set")?;
            Ok(PathBuf::from(home).join(".config/ghostty/config"))
        }
    }

    #[cfg(target_os = "windows")]
    {
        let appdata = env::var("APPDATA").map_err(|_| "APPDATA environment variable not set")?;
        Ok(PathBuf::from(appdata).join("ghostty/config"))
    }

    #[cfg(not(any(target_os = "macos", target_os = "linux", target_os = "windows")))]
    {
        Err("Unsupported platform".to_string())
    }
}

/// Get the ghostty-manager.sh script path
/// Priority: 1) GHOSTTY_MANAGER_PATH env var, 2) Same directory as config, 3) Bundled with binary
fn get_manager_script_path() -> Result<PathBuf, String> {
    // First check environment variable
    if let Ok(path) = env::var("GHOSTTY_MANAGER_PATH") {
        let script_path = PathBuf::from(path);
        if script_path.exists() {
            return Ok(script_path);
        }
    }

    // Try adjacent to the binary (shipped-alongside layout: <bin-dir>/../share/ghostty-tui/ghostty-manager.sh)
    if let Ok(exe) = env::current_exe() {
        if let Some(bin_dir) = exe.parent() {
            for candidate in [
                bin_dir.join("ghostty-manager.sh"),
                bin_dir.join("../share/ghostty-tui/ghostty-manager.sh"),
                bin_dir.join("../scripts/ghostty-manager.sh"),
            ] {
                if candidate.exists() {
                    return Ok(candidate);
                }
            }
        }
    }

    // Try in the same directory as the ghostty config (user-local setup)
    let config_path = get_ghostty_config_path()?;
    if let Some(config_dir) = config_path.parent() {
        if let Some(parent) = config_dir.parent() {
            let script_in_config = parent.join("ghostty-manager.sh");
            if script_in_config.exists() {
                return Ok(script_in_config);
            }
        }
    }

    Err("Could not find ghostty-manager.sh. Set GHOSTTY_MANAGER_PATH environment variable.".to_string())
}

// ============================================================================
// Asset discovery
// ============================================================================

/// Built-in cursor shader names, used when no on-disk asset dir is found.
const FALLBACK_CURSOR_SHADERS: &[&str] = &[
    "ripple_cursor", "sonic_boom_cursor", "cursor_tail", "cursor_sweep",
    "glow", "tesla_coil", "cursor_warp", "rectangle_boom_cursor",
    "ripple_rectangle_cursor",
];

/// Built-in screen shader names, used when no on-disk asset dir is found.
const FALLBACK_SCREEN_SHADERS: &[&str] = &[
    "amber-glow", "bettercrt", "bloom", "drunkard", "fireworks", "glitchy",
    "inside-the-matrix", "matrix-hallway", "retro-terminal", "starfield",
    "tft", "water",
];

/// Resolve the asset root directory.
///
/// Priority:
///   1. `$GHOSTTY_TUI_ASSETS`                           (explicit override)
///   2. `<binary-dir>/../assets/`                       (in-repo layout)
///   3. `<binary-dir>/../share/ghostty-tui/assets/`     (installed layout)
///   4. `$XDG_DATA_HOME/ghostty-tui/assets/`            (user-local fallback)
fn resolve_assets_dir() -> Option<PathBuf> {
    if let Ok(p) = env::var("GHOSTTY_TUI_ASSETS") {
        let path = PathBuf::from(p);
        if path.is_dir() {
            return Some(path);
        }
    }
    if let Ok(exe) = env::current_exe() {
        if let Some(bin_dir) = exe.parent() {
            for candidate in [
                bin_dir.join("../assets"),
                bin_dir.join("../share/ghostty-tui/assets"),
            ] {
                if candidate.is_dir() {
                    return Some(candidate);
                }
            }
        }
    }
    let xdg = env::var("XDG_DATA_HOME")
        .unwrap_or_else(|_| format!("{}/.local/share", env::var("HOME").unwrap_or_default()));
    let path = PathBuf::from(xdg).join("ghostty-tui/assets");
    if path.is_dir() {
        Some(path)
    } else {
        None
    }
}

/// Walk a directory for `*.glsl` files, return names (without extension), sorted.
fn walk_shaders(dir: &std::path::Path) -> Vec<String> {
    match std::fs::read_dir(dir) {
        Ok(entries) => {
            let mut names: Vec<String> = entries
                .filter_map(|e| e.ok())
                .filter_map(|e| {
                    let path = e.path();
                    if path.extension().and_then(|s| s.to_str()) == Some("glsl") {
                        path.file_stem().and_then(|s| s.to_str()).map(str::to_string)
                    } else {
                        None
                    }
                })
                .collect();
            names.sort();
            names
        }
        Err(_) => Vec::new(),
    }
}

/// Discover cursor shaders. Falls back to the built-in list if no asset dir is found.
fn discover_cursor_shaders() -> Vec<String> {
    let mut shaders = vec!["None".to_string()];
    if let Some(assets) = resolve_assets_dir() {
        let found = walk_shaders(&assets.join("shaders").join("cursors"));
        if !found.is_empty() {
            shaders.extend(found);
            return shaders;
        }
    }
    // Fallback to hardcoded list (keeps the UI functional without on-disk assets)
    for s in FALLBACK_CURSOR_SHADERS {
        shaders.push((*s).to_string());
    }
    shaders
}

/// Discover screen shaders. Falls back to the built-in list if no asset dir is found.
fn discover_screen_shaders() -> Vec<String> {
    if let Some(assets) = resolve_assets_dir() {
        let found = walk_shaders(&assets.join("shaders").join("screen"));
        if !found.is_empty() {
            return found;
        }
    }
    FALLBACK_SCREEN_SHADERS.iter().map(|s| s.to_string()).collect()
}

/// Install bundled assets to the user's ghostty config dir.
///
/// Copies `<assets>/themes/*` -> `~/.config/ghostty/themes/`
///        `<assets>/shaders/cursors/*.glsl` -> `~/.config/ghostty/shaders/cursors/`
///        `<assets>/shaders/screen/*.glsl` -> `~/.config/ghostty/shaders/screen/`
///
/// Does NOT overwrite existing files. Does NOT install config presets (those are
/// always opt-in per-file, since a config preset replaces the whole config).
fn install_assets() -> Result<(usize, usize), String> {
    let assets = resolve_assets_dir()
        .ok_or_else(|| "Could not locate bundled assets — set GHOSTTY_TUI_ASSETS or run from a checkout.".to_string())?;

    let home = env::var("HOME").map_err(|_| "HOME not set".to_string())?;
    let user_base = PathBuf::from(home).join(".config").join("ghostty");

    let mappings = [
        (assets.join("themes"), user_base.join("themes"), None),
        (assets.join("shaders").join("cursors"), user_base.join("shaders").join("cursors"), Some("glsl")),
        (assets.join("shaders").join("screen"),  user_base.join("shaders").join("screen"),  Some("glsl")),
    ];

    let mut copied = 0usize;
    let mut skipped = 0usize;

    for (src, dst, ext_filter) in &mappings {
        if !src.is_dir() {
            continue;
        }
        std::fs::create_dir_all(dst).map_err(|e| format!("mkdir {:?}: {}", dst, e))?;
        for entry in std::fs::read_dir(src).map_err(|e| format!("read {:?}: {}", src, e))? {
            let entry = entry.map_err(|e| e.to_string())?;
            let path = entry.path();
            if !path.is_file() {
                continue;
            }
            if let Some(ext) = ext_filter {
                if path.extension().and_then(|s| s.to_str()) != Some(ext) {
                    continue;
                }
            }
            // Skip README.md in theme dir
            if path.file_name().and_then(|s| s.to_str()) == Some("README.md") {
                continue;
            }
            let target = dst.join(path.file_name().unwrap());
            if target.exists() {
                skipped += 1;
                continue;
            }
            std::fs::copy(&path, &target).map_err(|e| format!("copy {:?} -> {:?}: {}", path, target, e))?;
            copied += 1;
        }
    }
    Ok((copied, skipped))
}

// ============================================================================
// Theme discovery
// ============================================================================

/// Curated "Popular" list shown at the top of the themes tab.
const POPULAR_THEMES: &[&str] = &[
    "Dracula",
    "TokyoNight",
    "Gruvbox Dark",
    "Catppuccin Mocha",
    "Nord",
    "Solarized Dark",
    "One Dark",
    "Monokai",
];

/// Fallback list shown only when `ghostty +list-themes` is unavailable.
const FALLBACK_THEMES: &[&str] = &[
    "Dracula",
    "TokyoNight",
    "Gruvbox Dark",
    "Catppuccin Mocha",
    "Nord",
    "Solarized Dark",
    "One Dark",
    "Monokai",
    "tokyonight-night",
    "tokyonight-storm",
    "tokyonight-day",
    "onenord",
    "onenord-light",
    "Tango",
    "Xterm",
    "Linux",
];

/// Discover themes at runtime.
///
/// Runs `ghostty +list-themes` and parses its output. Each line is of the
/// form `<name> (<source>)` — we strip the trailing parenthesised source tag.
///
/// Returns the assembled theme list (with the curated POPULAR header prepended)
/// and a boolean indicating whether discovery succeeded. If ghostty isn't in
/// `$PATH` or the command fails, returns the FALLBACK_THEMES wrapped in the
/// same header scheme so the UI still renders.
fn discover_themes() -> (Vec<String>, bool) {
    let mut out: Vec<String> = Vec::new();
    out.push("-- POPULAR --".to_string());
    for p in POPULAR_THEMES {
        out.push((*p).to_string());
    }
    out.push(String::new());

    match Command::new("ghostty").arg("+list-themes").output() {
        Ok(output) if output.status.success() => {
            out.push("-- ALL THEMES --".to_string());
            let stdout = String::from_utf8_lossy(&output.stdout);
            let mut names: Vec<String> = stdout
                .lines()
                .map(|line| {
                    // Strip trailing " (source)" tag if present.
                    match line.rfind(" (") {
                        Some(idx) if line.trim_end().ends_with(')') => line[..idx].to_string(),
                        _ => line.to_string(),
                    }
                })
                .filter(|s| !s.is_empty())
                .collect();
            names.sort();
            names.dedup();
            out.extend(names);
            (out, true)
        }
        _ => {
            out.push("-- FALLBACK (ghostty not in PATH) --".to_string());
            for t in FALLBACK_THEMES {
                out.push((*t).to_string());
            }
            (out, false)
        }
    }
}

// ============================================================================
// Application Data Structures
// ============================================================================

/// Available configuration tabs
#[derive(Debug, Clone, PartialEq)]
enum Tab {
    Themes,
    Font,
    Cursor,
    CursorShaders,
    ScreenShaders,
    Appearance,
    Mouse,
    Clipboard,
    Window,
    Shell,
    Config,
}

/// Main application state
#[derive(Debug)]
struct App {
    tab: Tab,
    theme_state: ListState,
    font_state: ListState,
    cursor_state: ListState,
    cursor_shader_state: ListState,
    screen_shader_state: ListState,
    appearance_state: ListState,
    mouse_state: ListState,
    clipboard_state: ListState,
    window_state: ListState,
    shell_state: ListState,

    themes: Vec<String>,
    font_options: Vec<String>,
    cursor_options: Vec<String>,
    cursor_shaders: Vec<String>,
    screen_shaders: Vec<String>,
    screen_shader_combos: Vec<(String, String)>, // (name, description)
    appearance_options: Vec<String>,
    mouse_options: Vec<String>,
    clipboard_options: Vec<String>,
    window_options: Vec<String>,
    shell_options: Vec<String>,

    current_config: String,
    status_message: String,
    manager_script_path: PathBuf,
}

// ============================================================================
// Application Implementation
// ============================================================================

impl App {
    /// Initialize the application with default state
    fn new() -> Result<Self, String> {
        // Get manager script path early to fail fast if not found
        let manager_script_path = get_manager_script_path()?;

        let (themes, themes_discovered) = discover_themes();
        let theme_status = if themes_discovered {
            format!("Discovered {} themes via `ghostty +list-themes`.", themes.len().saturating_sub(11))
        } else {
            "⚠ ghostty not in PATH — using fallback theme list.".to_string()
        };

        let mut app = Self {
            tab: Tab::Themes,
            theme_state: ListState::default(),
            font_state: ListState::default(),
            cursor_state: ListState::default(),
            cursor_shader_state: ListState::default(),
            screen_shader_state: ListState::default(),
            appearance_state: ListState::default(),
            mouse_state: ListState::default(),
            clipboard_state: ListState::default(),
            window_state: ListState::default(),
            shell_state: ListState::default(),

            themes,

            font_options: vec![
                "-- FONT FAMILY --".to_string(),
                "JetBrains Mono".to_string(),
                "Fira Code".to_string(),
                "Cascadia Code".to_string(),
                "SF Mono".to_string(),
                "Menlo".to_string(),
                "Monaco".to_string(),
                "Hack".to_string(),
                "Source Code Pro".to_string(),
                "IBM Plex Mono".to_string(),
                "Inconsolata".to_string(),
                "".to_string(),
                "-- FONT SIZE --".to_string(),
                "Size: 10pt".to_string(),
                "Size: 11pt".to_string(),
                "Size: 12pt".to_string(),
                "Size: 13pt".to_string(),
                "Size: 14pt".to_string(),
                "Size: 16pt".to_string(),
                "Size: 18pt".to_string(),
                "Size: 20pt".to_string(),
                "".to_string(),
                "-- FONT STYLE --".to_string(),
                "Weight: Normal".to_string(),
                "Weight: Bold".to_string(),
                "".to_string(),
                "-- LIGATURES --".to_string(),
                "Ligatures: On".to_string(),
                "Ligatures: Off".to_string(),
            ],
            cursor_options: vec![
                "-- CURSOR STYLE --".to_string(),
                "Style: Block".to_string(),
                "Style: Bar".to_string(),
                "Style: Underline".to_string(),
                "Style: Block Hollow".to_string(),
                "".to_string(),
                "-- CURSOR COLOR --".to_string(),
                "Color: Match Theme".to_string(),
                "Color: #7aa2f7 (Blue)".to_string(),
                "Color: #bb9af7 (Magenta)".to_string(),
                "Color: #e0af68 (Yellow)".to_string(),
                "Color: #9ece6a (Green)".to_string(),
                "Color: #f7768e (Red)".to_string(),
                "Color: #ffffff (White)".to_string(),
                "".to_string(),
                "-- CURSOR BLINK --".to_string(),
                "Blink: On".to_string(),
                "Blink: Off".to_string(),
                "".to_string(),
                "-- CURSOR OPACITY --".to_string(),
                "Opacity: 1.0 (Solid)".to_string(),
                "Opacity: 0.9".to_string(),
                "Opacity: 0.8".to_string(),
                "Opacity: 0.7".to_string(),
                "Opacity: 0.5 (Faded)".to_string(),
            ],

            cursor_shaders: discover_cursor_shaders(),

            screen_shaders: discover_screen_shaders(),

            screen_shader_combos: vec![
                ("None".to_string(), "Disable all screen shaders".to_string()),
                ("crt-classic".to_string(), "tft + bettercrt".to_string()),
                ("crt-bloom".to_string(), "tft + retro-terminal + bloom".to_string()),
                ("drunk-retro".to_string(), "drunkard + retro-terminal + bloom".to_string()),
                ("glitch-water".to_string(), "glitchy + bettercrt + water + bloom".to_string()),
            ],

            appearance_options: vec!["── PRESETS ──","Default (Liquid Glass)","Solid (No transparency)","── OPACITY ──","Opacity: 0.7 (Light)","Opacity: 0.8","Opacity: 0.9","Opacity: 0.95 (Current)","Opacity: 1.0 (Solid)","── BLUR ──","Blur: 0 (Off)","Blur: 10 (Subtle)","Blur: 20 (Medium)","Blur: 40 (Strong)","── PADDING X ──","Padding X: 0","Padding X: 5","Padding X: 10","Padding X: 20","── PADDING Y ──","Padding Y: 0","Padding Y: 5","Padding Y: 10","Padding Y: 20"].iter().map(|s| s.to_string()).collect(),

            mouse_options: vec![
                "-- MOUSE BEHAVIOR --".to_string(),
                "Hide While Typing: On".to_string(),
                "Hide While Typing: Off".to_string(),
                "".to_string(),
                "-- SCROLL SPEED --".to_string(),
                "Scroll Multiplier: 1.0 (Normal)".to_string(),
                "Scroll Multiplier: 2.0".to_string(),
                "Scroll Multiplier: 3.0".to_string(),
                "Scroll Multiplier: 5.0 (Fast)".to_string(),
                "".to_string(),
                "-- COPY BEHAVIOR --".to_string(),
                "Copy on Select: Off".to_string(),
                "Copy on Select: On (Selection)".to_string(),
                "Copy on Select: Clipboard".to_string(),
                "".to_string(),
                "-- FOCUS --".to_string(),
                "Focus Follows Mouse: Off".to_string(),
                "Focus Follows Mouse: On".to_string(),
            ],
            clipboard_options: vec!["-- CLIPBOARD READ --","Read: Ask (Prompt)","Read: Allow","Read: Deny","","-- CLIPBOARD WRITE --","Write: Ask (Prompt)","Write: Allow","Write: Deny","","-- CLIPBOARD OPTIONS --","Trim Trailing Spaces: On","Trim Trailing Spaces: Off","","-- SECURITY --","Paste Protection: On","Paste Protection: Off"].iter().map(|s| s.to_string()).collect(),

            window_options: vec!["-- SIZE PRESETS --".to_string(),"Size: Small (80x24)".to_string(),"Size: Medium (100x30)".to_string(),"Size: Large (120x40)".to_string(),"Size: Ultrawide (200x50)".to_string(),"Size: Gostty (100x50)".to_string(),"".to_string(),"-- WINDOW DECORATIONS --".to_string(),"Decorations: Auto".to_string(),"Decorations: None".to_string(),"Decorations: Client".to_string(),"Decorations: Server".to_string(),"".to_string(),"-- TAB BAR --".to_string(),"Tab Bar: Auto".to_string(),"Tab Bar: Always".to_string(),"Tab Bar: Never".to_string(),"".to_string(),"-- DISPLAY --".to_string(),"VSync: On".to_string(),"VSync: Off".to_string(),"Fullscreen: On".to_string(),"Fullscreen: Off".to_string(),"".to_string(),"-- STATE --".to_string(),"Save State: Default".to_string(),"Save State: Always".to_string(),"Save State: Never".to_string()],

            shell_options: vec!["-- SHELL INTEGRATION --","Integration: Detect (Auto)","Integration: Bash","Integration: Fish","Integration: Zsh","Integration: Elvish","Integration: None","","-- WORKING DIRECTORY --","Start In: Home","Start In: Inherit","","-- CLOSE BEHAVIOR --","Confirm Close: On","Confirm Close: Off","Confirm Close: Always","","-- SCROLLBACK --","Scrollback: 1,000 lines","Scrollback: 5,000 lines","Scrollback: 10,000 lines","Scrollback: 50,000 lines","Scrollback: 100,000 lines"].iter().map(|s| s.to_string()).collect(),

            current_config: String::new(),
            status_message: theme_status,
            manager_script_path,
        };

        app.theme_state.select(Some(1)); // Start at first theme (skip header)
        app.font_state.select(Some(0));
        app.cursor_state.select(Some(0));
        app.cursor_shader_state.select(Some(0));
        app.screen_shader_state.select(Some(1)); // Start at first combo (skip header)
        app.appearance_state.select(Some(1)); // Start at first preset
        app.mouse_state.select(Some(1)); // Start at first option
        app.clipboard_state.select(Some(1)); // Start at first option
        app.window_state.select(Some(0));
        app.shell_state.select(Some(1)); // Start at first option
        app.load_current_config();

        Ok(app)
    }

    fn load_current_config(&mut self) {
        let output = Command::new(&self.manager_script_path)
            .arg("current")
            .output();

        match output {
            Ok(output) if output.status.success() => {
                self.current_config = String::from_utf8_lossy(&output.stdout).to_string();
            }
            Ok(output) => {
                let error = String::from_utf8_lossy(&output.stderr);
                self.status_message = format!("Failed to load config: {}", error);
                self.current_config = "Error loading configuration".to_string();
            }
            Err(e) => {
                self.status_message = format!("Failed to execute manager script: {}", e);
                self.current_config = "Error loading configuration".to_string();
            }
        }
    }

    fn next_tab(&mut self) {
        self.tab = match self.tab {
            Tab::Themes => Tab::Font,
            Tab::Font => Tab::Cursor,
            Tab::Cursor => Tab::CursorShaders,
            Tab::CursorShaders => Tab::ScreenShaders,
            Tab::ScreenShaders => Tab::Appearance,
            Tab::Appearance => Tab::Mouse,
            Tab::Mouse => Tab::Clipboard,
            Tab::Clipboard => Tab::Window,
            Tab::Window => Tab::Shell,
            Tab::Shell => Tab::Config,
            Tab::Config => Tab::Themes,
        };
    }

    fn previous_tab(&mut self) {
        self.tab = match self.tab {
            Tab::Themes => Tab::Config,
            Tab::Font => Tab::Themes,
            Tab::Cursor => Tab::Font,
            Tab::CursorShaders => Tab::Cursor,
            Tab::ScreenShaders => Tab::CursorShaders,
            Tab::Appearance => Tab::ScreenShaders,
            Tab::Mouse => Tab::Appearance,
            Tab::Clipboard => Tab::Mouse,
            Tab::Window => Tab::Clipboard,
            Tab::Shell => Tab::Window,
            Tab::Config => Tab::Shell,
        };
    }

    fn next_item(&mut self) {
        match self.tab {
            Tab::Themes => {
                let len = self.themes.len();
                let i = match self.theme_state.selected() {
                    Some(i) => {
                        let mut next = if i >= len - 1 { 0 } else { i + 1 };
                        // Skip separators and empty lines
                        while self.themes[next].starts_with("--") || self.themes[next].is_empty() {
                            next = if next >= len - 1 { 0 } else { next + 1 };
                        }
                        next
                    }
                    None => 1, // Start at first theme (skip header)
                };
                self.theme_state.select(Some(i));
            }
            Tab::Font => {
                let len = self.font_options.len();
                if len > 0 {
                    let i = match self.font_state.selected() {
                        Some(i) => if i >= len - 1 { 0 } else { i + 1 },
                        None => 0,
                    };
                    self.font_state.select(Some(i));
                }
            }
            Tab::Cursor => {
                let len = self.cursor_options.len();
                if len > 0 {
                    let i = match self.cursor_state.selected() {
                        Some(i) => if i >= len - 1 { 0 } else { i + 1 },
                        None => 0,
                    };
                    self.cursor_state.select(Some(i));
                }
            }
            Tab::CursorShaders => {
                let len = self.cursor_shaders.len();
                let i = match self.cursor_shader_state.selected() {
                    Some(i) => if i >= len - 1 { 0 } else { i + 1 },
                    None => 0,
                };
                self.cursor_shader_state.select(Some(i));
            }
            Tab::ScreenShaders => {
                // Total items including separators: 1 (header) + combos + 1 (empty) + 1 (header) + shaders
                let total_len = 1 + self.screen_shader_combos.len() + 1 + 1 + self.screen_shaders.len();
                let i = match self.screen_shader_state.selected() {
                    Some(i) => {
                        let mut next = if i >= total_len - 1 { 0 } else { i + 1 };
                        // Skip separators at positions: 0 (first header), combos_len+1 (empty), combos_len+2 (second header)
                        loop {
                            if next == 0 || next == self.screen_shader_combos.len() + 1 || next == self.screen_shader_combos.len() + 2 {
                                next = if next >= total_len - 1 { 0 } else { next + 1 };
                            } else {
                                break;
                            }
                        }
                        next
                    }
                    None => 1, // Start at first combo (skip header)
                };
                self.screen_shader_state.select(Some(i));
            }
            Tab::Appearance => {
                let len = self.appearance_options.len();
                let i = match self.appearance_state.selected() {
                    Some(i) => {
                        let mut next = if i >= len - 1 { 0 } else { i + 1 };
                        // Skip separators
                        while self.appearance_options[next].starts_with("──") {
                            next = if next >= len - 1 { 0 } else { next + 1 };
                        }
                        next
                    }
                    None => 1,
                };
                self.appearance_state.select(Some(i));
            }
            Tab::Mouse => {
                let len = self.mouse_options.len();
                if len > 0 {
                    let i = match self.mouse_state.selected() {
                        Some(i) => if i >= len - 1 { 0 } else { i + 1 },
                        None => 0,
                    };
                    self.mouse_state.select(Some(i));
                }
            }
            Tab::Clipboard => {
                let len = self.clipboard_options.len();
                if len > 0 { self.clipboard_state.select(Some(match self.clipboard_state.selected() { Some(i) => if i >= len - 1 { 0 } else { i + 1 }, None => 0 })); }
            }
            Tab::Window => {
                let len = self.window_options.len();
                let i = match self.window_state.selected() {
                    Some(i) => if i >= len - 1 { 0 } else { i + 1 },
                    None => 0,
                };
                self.window_state.select(Some(i));
            }
            Tab::Shell => {
                let len = self.shell_options.len();
                if len > 0 {
                    let i = match self.shell_state.selected() {
                        Some(i) => if i >= len - 1 { 0 } else { i + 1 },
                        None => 0,
                    };
                    self.shell_state.select(Some(i));
                }
            }
            Tab::Config => {}
        }
    }

    fn previous_item(&mut self) {
        match self.tab {
            Tab::Themes => {
                let len = self.themes.len();
                let i = match self.theme_state.selected() {
                    Some(i) => {
                        let mut prev = if i == 0 { len - 1 } else { i - 1 };
                        // Skip separators and empty lines
                        while self.themes[prev].starts_with("--") || self.themes[prev].is_empty() {
                            prev = if prev == 0 { len - 1 } else { prev - 1 };
                        }
                        prev
                    }
                    None => 1,
                };
                self.theme_state.select(Some(i));
            }
            Tab::Font => {
                let len = self.font_options.len();
                if len > 0 {
                    let i = match self.font_state.selected() {
                        Some(i) => if i == 0 { len - 1 } else { i - 1 },
                        None => 0,
                    };
                    self.font_state.select(Some(i));
                }
            }
            Tab::Cursor => {
                let len = self.cursor_options.len();
                if len > 0 {
                    let i = match self.cursor_state.selected() {
                        Some(i) => if i == 0 { len - 1 } else { i - 1 },
                        None => 0,
                    };
                    self.cursor_state.select(Some(i));
                }
            }
            Tab::CursorShaders => {
                let len = self.cursor_shaders.len();
                let i = match self.cursor_shader_state.selected() {
                    Some(i) => if i == 0 { len - 1 } else { i - 1 },
                    None => 0,
                };
                self.cursor_shader_state.select(Some(i));
            }
            Tab::ScreenShaders => {
                // Total items including separators
                let total_len = 1 + self.screen_shader_combos.len() + 1 + 1 + self.screen_shaders.len();
                let i = match self.screen_shader_state.selected() {
                    Some(i) => {
                        let mut prev = if i == 0 { total_len - 1 } else { i - 1 };
                        // Skip separators
                        loop {
                            if prev == 0 || prev == self.screen_shader_combos.len() + 1 || prev == self.screen_shader_combos.len() + 2 {
                                prev = if prev == 0 { total_len - 1 } else { prev - 1 };
                            } else {
                                break;
                            }
                        }
                        prev
                    }
                    None => 1,
                };
                self.screen_shader_state.select(Some(i));
            }
            Tab::Appearance => {
                let len = self.appearance_options.len();
                let i = match self.appearance_state.selected() {
                    Some(i) => {
                        let mut prev = if i == 0 { len - 1 } else { i - 1 };
                        // Skip separators
                        while self.appearance_options[prev].starts_with("──") {
                            prev = if prev == 0 { len - 1 } else { prev - 1 };
                        }
                        prev
                    }
                    None => 1,
                };
                self.appearance_state.select(Some(i));
            }
            Tab::Mouse => {
                let len = self.mouse_options.len();
                if len > 0 {
                    let i = match self.mouse_state.selected() {
                        Some(i) => if i == 0 { len - 1 } else { i - 1 },
                        None => 0,
                    };
                    self.mouse_state.select(Some(i));
                }
            }
            Tab::Clipboard => {
                let len = self.clipboard_options.len();
                if len > 0 { self.clipboard_state.select(Some(match self.clipboard_state.selected() { Some(i) => if i == 0 { len - 1 } else { i - 1 }, None => 0 })); }
            }
            Tab::Window => {
                let len = self.window_options.len();
                let i = match self.window_state.selected() {
                    Some(i) => if i == 0 { len - 1 } else { i - 1 },
                    None => 0,
                };
                self.window_state.select(Some(i));
            }
            Tab::Shell => {
                let len = self.shell_options.len();
                if len > 0 {
                    let i = match self.shell_state.selected() {
                        Some(i) => if i == 0 { len - 1 } else { i - 1 },
                        None => 0,
                    };
                    self.shell_state.select(Some(i));
                }
            }
            Tab::Config => {}
        }
    }

    fn select_item(&mut self) {
        match self.tab {
            Tab::Themes => {
                if let Some(i) = self.theme_state.selected() {
                    let theme = &self.themes[i];
                    if theme.starts_with("--") || theme.is_empty() { return; }
                    self.execute_command(&format!("theme \"{}\"", theme));
                }
            }
            Tab::Font => {
                if let Some(i) = self.font_state.selected() {
                    let option = &self.font_options[i];
                    if option.starts_with("--") || option.is_empty() {
                        return; // Skip separators
                    }

                    if option.starts_with("Size:") {
                        let size = option.split(": ").nth(1).unwrap_or("14").replace("pt", "");
                        self.execute_command(&format!("font-size {}", size));
                    } else if option.starts_with("Weight:") {
                        let weight = if option.contains("Bold") { "bold" } else { "normal" };
                        self.execute_command(&format!("font-weight {}", weight));
                    } else if option.starts_with("Ligatures:") {
                        let state = if option.contains("On") { "on" } else { "off" };
                        self.execute_command(&format!("ligatures {}", state));
                    } else {
                        // Font family
                        self.execute_command(&format!("font-family \"{}\"", option));
                    }
                }
            }
            Tab::Cursor => {
                if let Some(i) = self.cursor_state.selected() {
                    let option = &self.cursor_options[i];
                    if option.starts_with("--") || option.is_empty() {
                        return; // Skip separators
                    }

                    if option.starts_with("Style:") {
                        let style = if option.contains("Block Hollow") { "block_hollow" }
                            else if option.contains("Block") { "block" }
                            else if option.contains("Bar") { "bar" }
                            else { "underline" };
                        self.execute_command(&format!("cursor-style {}", style));
                    } else if option.starts_with("Color:") {
                        if option.contains("Match Theme") {
                            self.execute_command("cursor-color theme");
                        } else if option.contains("#") {
                            let color = option.split(" ").nth(1).unwrap_or("#ffffff");
                            self.execute_command(&format!("cursor-color {}", color));
                        }
                    } else if option.starts_with("Blink:") {
                        let state = if option.contains("On") { "true" } else { "false" };
                        self.execute_command(&format!("cursor-blink {}", state));
                    } else if option.starts_with("Opacity:") {
                        let opacity = option.split(": ").nth(1).unwrap_or("1.0").split(" ").next().unwrap_or("1.0");
                        self.execute_command(&format!("cursor-opacity {}", opacity));
                    }
                }
            }
            Tab::CursorShaders => {
                if let Some(i) = self.cursor_shader_state.selected() {
                    let shader = &self.cursor_shaders[i];
                    if shader == "None" {
                        self.execute_command("shader-off");
                    } else {
                        self.execute_command(&format!("shader \"{}\"", shader));
                    }
                }
            }
            Tab::ScreenShaders => {
                if let Some(i) = self.screen_shader_state.selected() {
                    // Account for separators: 1 (header) + combos + 1 (empty) + 1 (header) + shaders
                    let combos_end = 1 + self.screen_shader_combos.len();
                    let shaders_start = combos_end + 2; // Skip empty line and "INDIVIDUAL" header

                    if i >= 1 && i < combos_end {
                        // It's a combo (adjust for header)
                        let combo_idx = i - 1;
                        let combo = &self.screen_shader_combos[combo_idx].0;
                        if combo == "None" {
                            self.execute_command("screen-shader-off");
                        } else {
                            self.execute_command(&format!("screen-shader \"{}\"", combo));
                        }
                    } else if i >= shaders_start {
                        // It's an individual shader
                        let shader_idx = i - shaders_start;
                        if shader_idx < self.screen_shaders.len() {
                            let shader = &self.screen_shaders[shader_idx];
                            self.execute_command(&format!("screen-shader \"{}\"", shader));
                        }
                    }
                }
            }
            Tab::Appearance => {
                if let Some(i) = self.appearance_state.selected() {
                    let option = &self.appearance_options[i];
                    if option.starts_with("──") { return; }
                    if option == "Default (Liquid Glass)" { self.execute_command("appearance default"); }
                    else if option == "Solid (No transparency)" { self.execute_command("appearance solid"); }
                    else if option.starts_with("Opacity:") {
                        let opacity = if option.contains("0.7") { "0.7" } else if option.contains("0.8") { "0.8" } else if option.contains("0.9") { "0.9" } else if option.contains("0.95") { "0.95" } else { "1.0" };
                        self.execute_command(&format!("opacity \"{}\"", opacity));
                    } else if option.starts_with("Blur:") {
                        let blur = if option.contains("0 ") { "0" } else if option.contains("10") { "10" } else if option.contains("20") { "20" } else { "40" };
                        self.execute_command(&format!("blur \"{}\"", blur));
                    } else if option.starts_with("Padding X:") {
                        let padding = option.split(": ").nth(1).unwrap_or("0");
                        self.execute_command(&format!("padding-x {}", padding));
                    } else if option.starts_with("Padding Y:") {
                        let padding = option.split(": ").nth(1).unwrap_or("0");
                        self.execute_command(&format!("padding-y {}", padding));
                    }
                }
            }
            Tab::Mouse => {
                if let Some(i) = self.mouse_state.selected() {
                    let option = &self.mouse_options[i];
                    if option.starts_with("--") || option.is_empty() {
                        return; // Skip separators
                    }

                    if option.starts_with("Hide While Typing:") {
                        let state = if option.contains("On") { "true" } else { "false" };
                        self.execute_command(&format!("mouse-hide {}", state));
                    } else if option.starts_with("Scroll Multiplier:") {
                        let mult = option.split(": ").nth(1).unwrap_or("1.0").split(" ").next().unwrap_or("1.0");
                        self.execute_command(&format!("scroll-multiplier {}", mult));
                    } else if option.starts_with("Copy on Select:") {
                        let mode = if option.contains("Off") { "false" }
                            else if option.contains("Clipboard") { "clipboard" }
                            else { "true" };
                        self.execute_command(&format!("copy-on-select {}", mode));
                    } else if option.starts_with("Focus Follows Mouse:") {
                        let state = if option.contains("On") { "true" } else { "false" };
                        self.execute_command(&format!("focus-follows-mouse {}", state));
                    }
                }
            }
            Tab::Clipboard => {
                if let Some(i) = self.clipboard_state.selected() {
                    let o = &self.clipboard_options[i];
                    if o.starts_with("--") || o.is_empty() { return; }
                    if o.starts_with("Read:") { self.execute_command(&format!("clipboard-read {}", if o.contains("Ask") { "ask" } else if o.contains("Allow") { "allow" } else { "deny" })); }
                    else if o.starts_with("Write:") { self.execute_command(&format!("clipboard-write {}", if o.contains("Ask") { "ask" } else if o.contains("Allow") { "allow" } else { "deny" })); }
                    else if o.starts_with("Trim") { self.execute_command(&format!("clipboard-trim {}", if o.contains("On") { "true" } else { "false" })); }
                    else if o.starts_with("Paste") { self.execute_command(&format!("paste-protection {}", if o.contains("On") { "true" } else { "false" })); }
                }
            }
            Tab::Window => {
                if let Some(i) = self.window_state.selected() {
                    let o = &self.window_options[i];
                    if o.starts_with("Size:") { let s = if o.contains("Small") { "small" } else if o.contains("Medium") { "medium" } else if o.contains("Large") { "large" } else if o.contains("Ultrawide") { "ultrawide" } else { "gostty" }; self.execute_command(&format!("size \"{}\"", s)); }
                    else if o.starts_with("Decorations:") { let d = if o.contains("None") { "none" } else if o.contains("Client") { "client" } else if o.contains("Server") { "server" } else { "auto" }; self.execute_command(&format!("window-decoration {}", d)); }
                    else if o.starts_with("Tab Bar:") { let t = if o.contains("Always") { "top" } else if o.contains("Never") { "hidden" } else { "auto" }; self.execute_command(&format!("tab-bar {}", t)); }
                    else if o.starts_with("VSync:") { self.execute_command(&format!("vsync {}", if o.contains("On") { "true" } else { "false" })); }
                    else if o.starts_with("Fullscreen:") { self.execute_command(&format!("fullscreen {}", if o.contains("On") { "true" } else { "false" })); }
                    else if o.starts_with("Save State:") { let s = if o.contains("Always") { "always" } else if o.contains("Never") { "never" } else { "default" }; self.execute_command(&format!("save-state {}", s)); }
                }
            }
            Tab::Shell => {
                if let Some(i) = self.shell_state.selected() {
                    let o = &self.shell_options[i];
                    if o.starts_with("--") || o.is_empty() { return; }
                    if o.starts_with("Integration:") { let m = if o.contains("Detect") { "detect" } else if o.contains("Bash") { "bash" } else if o.contains("Fish") { "fish" } else if o.contains("Zsh") { "zsh" } else if o.contains("Elvish") { "elvish" } else { "none" }; self.execute_command(&format!("shell-integration {}", m)); }
                    else if o.starts_with("Start In:") { let d = if o.contains("Home") { "home" } else { "inherit" }; self.execute_command(&format!("working-directory {}", d)); }
                    else if o.starts_with("Confirm Close:") { let c = if o.contains("Always") { "always" } else if o.contains("Off") { "false" } else { "true" }; self.execute_command(&format!("confirm-close {}", c)); }
                    else if o.starts_with("Scrollback:") { let lines = if o.contains("1,000") { "1000" } else if o.contains("5,000") { "5000" } else if o.contains("10,000") { "10000" } else if o.contains("50,000") { "50000" } else { "100000" }; self.execute_command(&format!("scrollback {}", lines)); }
                }
            }
            Tab::Config => {}
        }
    }

    fn execute_command(&mut self, cmd: &str) {
        let args = match shell_words::split(cmd) {
            Ok(parts) if !parts.is_empty() => parts,
            Ok(_) => {
                self.status_message = "✗ Empty command".to_string();
                return;
            }
            Err(e) => {
                self.status_message = format!("✗ Malformed command: {}", e);
                return;
            }
        };

        let output = Command::new(&self.manager_script_path)
            .args(&args)
            .output();

        match output {
            Ok(output) if output.status.success() => {
                self.status_message = format!("✓ Applied: {}", cmd);
                self.load_current_config();
            }
            Ok(output) => {
                let error = String::from_utf8_lossy(&output.stderr);
                self.status_message = format!("✗ Failed: {} - {}", cmd, error);
            }
            Err(e) => {
                self.status_message = format!("✗ Error executing command: {}", e);
            }
        }
    }
}

// ============================================================================
// Main Entry Point & CLI
// ============================================================================

fn main() -> Result<(), io::Error> {
    // Parse CLI arguments
    let args: Vec<String> = env::args().collect();

    // Handle --help flag
    if args.len() > 1 {
        match args[1].as_str() {
            "--help" | "-h" => {
                print_help();
                return Ok(());
            }
            "--version" | "-v" => {
                println!("ghostty-tui v{}", VERSION);
                return Ok(());
            }
            "--config-path" => {
                match get_ghostty_config_path() {
                    Ok(path) => {
                        println!("{}", path.display());
                        return Ok(());
                    }
                    Err(e) => {
                        eprintln!("Error: {}", e);
                        std::process::exit(1);
                    }
                }
            }
            "--install-assets" => {
                match install_assets() {
                    Ok((copied, skipped)) => {
                        println!("Installed {} files. Skipped {} that already existed.", copied, skipped);
                        println!("Install dir: ~/.config/ghostty/{{themes,shaders/{{cursors,screen}}}}");
                        return Ok(());
                    }
                    Err(e) => {
                        eprintln!("install-assets failed: {}", e);
                        std::process::exit(1);
                    }
                }
            }
            "--assets-path" => {
                match resolve_assets_dir() {
                    Some(p) => {
                        println!("{}", p.display());
                        return Ok(());
                    }
                    None => {
                        eprintln!("No assets directory resolved. Set GHOSTTY_TUI_ASSETS.");
                        std::process::exit(1);
                    }
                }
            }
            _ => {
                eprintln!("Unknown argument: {}", args[1]);
                eprintln!("Use --help for usage information");
                std::process::exit(1);
            }
        }
    }

    // Create app (fail fast if setup is invalid)
    let mut app = match App::new() {
        Ok(app) => app,
        Err(e) => {
            eprintln!("Failed to initialize: {}", e);
            eprintln!("\nTroubleshooting:");
            eprintln!("  1. Ensure ghostty-manager.sh is accessible");
            eprintln!("  2. Set GHOSTTY_MANAGER_PATH environment variable to script location");
            eprintln!("  3. Run with --config-path to see expected config location");
            std::process::exit(1);
        }
    };

    // Setup terminal
    enable_raw_mode()?;
    let mut stdout = io::stdout();
    execute!(stdout, EnterAlternateScreen)?;
    let backend = CrosstermBackend::new(stdout);
    let mut terminal = Terminal::new(backend)?;

    // Run app
    let res = run_app(&mut terminal, &mut app);

    // Restore terminal
    disable_raw_mode()?;
    execute!(terminal.backend_mut(), LeaveAlternateScreen)?;
    terminal.show_cursor()?;

    if let Err(err) = res {
        println!("{:?}", err)
    }

    Ok(())
}

fn print_help() {
    println!("ghostty-tui v{}", VERSION);
    println!();
    println!("USAGE:");
    println!("    ghostty-tui [OPTIONS]");
    println!();
    println!("OPTIONS:");
    println!("    -h, --help           Print this help message");
    println!("    -v, --version        Print version information");
    println!("    --config-path        Print the Ghostty config file path for this platform");
    println!("    --assets-path        Print the resolved ghostty-tui assets directory");
    println!("    --install-assets     Copy bundled themes + shaders into ~/.config/ghostty/");
    println!();
    println!("ENVIRONMENT VARIABLES:");
    println!("    GHOSTTY_MANAGER_PATH    Path to ghostty-manager.sh script");
    println!("    GHOSTTY_TUI_ASSETS      Path to the ghostty-tui assets dir (themes/shaders/configs)");
    println!("    GHOSTTY_THEMES_DIR      Override user themes directory (default ~/.config/ghostty/themes)");
    println!("    GHOSTTY_CONFIG_PATH     Override active Ghostty config path");
    println!();
    println!("NAVIGATION:");
    println!("    Tab/Shift+Tab        Switch between tabs");
    println!("    ↑/↓ or j/k           Navigate options");
    println!("    Enter                Select option");
    println!("    r                    Reload current config");
    println!("    q or Esc             Quit");
    println!();
    println!("For more information, visit: https://github.com/snakeg0d/ghostty-tui");
}

// ============================================================================
// Event Loop
// ============================================================================

fn run_app<B: ratatui::backend::Backend>(
    terminal: &mut Terminal<B>,
    app: &mut App,
) -> io::Result<()> {
    loop {
        terminal.draw(|f| ui(f, app))?;

        if let Event::Key(key) = event::read()? {
            if key.kind == KeyEventKind::Press {
                match key.code {
                    KeyCode::Char('q') | KeyCode::Esc => return Ok(()),
                    KeyCode::Tab => app.next_tab(),
                    KeyCode::BackTab => app.previous_tab(),
                    KeyCode::Down | KeyCode::Char('j') => app.next_item(),
                    KeyCode::Up | KeyCode::Char('k') => app.previous_item(),
                    KeyCode::Enter => app.select_item(),
                    KeyCode::Char('r') => app.load_current_config(),
                    _ => {}
                }
            }
        }
    }
}

// ============================================================================
// UI Rendering
// ============================================================================

fn ui(f: &mut Frame, app: &mut App) {
    let chunks = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(3),  // Header
            Constraint::Length(3),  // Tabs
            Constraint::Min(0),     // Content
            Constraint::Length(3),  // Status
        ])
        .split(f.area());

    // Header
    let header = Paragraph::new("╔═══ GHOSTTY CONTROL PANEL ═══╗")
        .style(Style::default().fg(BLUE).bold())
        .alignment(Alignment::Center);
    f.render_widget(header, chunks[0]);

    // Tabs
    let tabs_text = vec![
        Span::styled("1", Style::default().fg(if app.tab == Tab::Themes { MAGENTA } else { GRAY })),
        Span::raw(" Theme  "),
        Span::styled("2", Style::default().fg(if app.tab == Tab::Font { MAGENTA } else { GRAY })),
        Span::raw(" Font  "),
        Span::styled("3", Style::default().fg(if app.tab == Tab::Cursor { MAGENTA } else { GRAY })),
        Span::raw(" Cursor  "),
        Span::styled("4", Style::default().fg(if app.tab == Tab::CursorShaders { MAGENTA } else { GRAY })),
        Span::raw(" CFX  "),
        Span::styled("5", Style::default().fg(if app.tab == Tab::ScreenShaders { MAGENTA } else { GRAY })),
        Span::raw(" SFX  "),
        Span::styled("6", Style::default().fg(if app.tab == Tab::Appearance { MAGENTA } else { GRAY })),
        Span::raw(" Look  "),
        Span::styled("7", Style::default().fg(if app.tab == Tab::Mouse { MAGENTA } else { GRAY })),
        Span::raw(" Mouse  "),
        Span::styled("8", Style::default().fg(if app.tab == Tab::Clipboard { MAGENTA } else { GRAY })),
        Span::raw(" Clip  "),
        Span::styled("9", Style::default().fg(if app.tab == Tab::Window { MAGENTA } else { GRAY })),
        Span::raw(" Win  "),
        Span::styled("10", Style::default().fg(if app.tab == Tab::Shell { MAGENTA } else { GRAY })),
        Span::raw(" Shell  "),
        Span::styled("11", Style::default().fg(if app.tab == Tab::Config { MAGENTA } else { GRAY })),
        Span::raw(" Cfg"),
    ];

    let tabs = Paragraph::new(Line::from(tabs_text))
        .alignment(Alignment::Center)
        .block(Block::default().borders(Borders::BOTTOM).border_style(Style::default().fg(BLUE)));
    f.render_widget(tabs, chunks[1]);

    // Content
    match app.tab {
        Tab::Themes => render_themes(f, app, chunks[2]),
        Tab::Font => render_font(f, app, chunks[2]),
        Tab::Cursor => render_cursor(f, app, chunks[2]),
        Tab::CursorShaders => render_cursor_shaders(f, app, chunks[2]),
        Tab::ScreenShaders => render_screen_shaders(f, app, chunks[2]),
        Tab::Appearance => render_appearance(f, app, chunks[2]),
        Tab::Mouse => render_mouse(f, app, chunks[2]),
        Tab::Clipboard => render_clipboard(f, app, chunks[2]),
        Tab::Window => render_window(f, app, chunks[2]),
        Tab::Shell => render_shell(f, app, chunks[2]),
        Tab::Config => render_config(f, app, chunks[2]),
    }

    // Status bar
    let status = Paragraph::new(app.status_message.as_str())
        .style(Style::default().fg(YELLOW))
        .block(Block::default().borders(Borders::TOP).border_style(Style::default().fg(BLUE)));
    f.render_widget(status, chunks[3]);
}

fn render_themes(f: &mut Frame, app: &mut App, area: Rect) {
    let items: Vec<ListItem> = app.themes.iter().map(|t| ListItem::new(t.as_str()).style(if t.starts_with("--") { Style::default().fg(YELLOW).bold() } else if t.is_empty() { Style::default() } else { Style::default().fg(Color::White) })).collect();
    f.render_stateful_widget(List::new(items).block(Block::default().title(" 🎨 THEMES ").title_style(Style::default().fg(MAGENTA).bold()).borders(Borders::ALL).border_style(Style::default().fg(BLUE))).highlight_style(Style::default().bg(MAGENTA).fg(BG).bold()).highlight_symbol("▶ "), area, &mut app.theme_state);
}

fn render_font(f: &mut Frame, app: &mut App, area: Rect) {
    let items: Vec<ListItem> = app.font_options
        .iter()
        .map(|option| {
            let style = if option.starts_with("--") {
                Style::default().fg(YELLOW).bold()
            } else if option.is_empty() {
                Style::default()
            } else {
                Style::default().fg(Color::White)
            };
            ListItem::new(option.as_str()).style(style)
        })
        .collect();

    let list = List::new(items)
        .block(Block::default()
            .title(" FONT ")
            .title_style(Style::default().fg(MAGENTA).bold())
            .borders(Borders::ALL)
            .border_style(Style::default().fg(BLUE)))
        .highlight_style(Style::default().bg(BLUE).fg(BG).bold())
        .highlight_symbol("> ");

    f.render_stateful_widget(list, area, &mut app.font_state);
}

fn render_cursor(f: &mut Frame, app: &mut App, area: Rect) {
    let items: Vec<ListItem> = app.cursor_options
        .iter()
        .map(|option| {
            let style = if option.starts_with("--") {
                Style::default().fg(YELLOW).bold()
            } else if option.is_empty() {
                Style::default()
            } else {
                Style::default().fg(Color::White)
            };
            ListItem::new(option.as_str()).style(style)
        })
        .collect();

    let list = List::new(items)
        .block(Block::default()
            .title(" CURSOR ")
            .title_style(Style::default().fg(MAGENTA).bold())
            .borders(Borders::ALL)
            .border_style(Style::default().fg(BLUE)))
        .highlight_style(Style::default().bg(BLUE).fg(BG).bold())
        .highlight_symbol("> ");

    f.render_stateful_widget(list, area, &mut app.cursor_state);
}

fn render_cursor_shaders(f: &mut Frame, app: &mut App, area: Rect) {
    let items: Vec<ListItem> = app.cursor_shaders
        .iter()
        .map(|shader| ListItem::new(shader.as_str()))
        .collect();

    let list = List::new(items)
        .block(Block::default()
            .title(" ⚡ CURSOR EFFECTS ")
            .title_style(Style::default().fg(MAGENTA).bold())
            .borders(Borders::ALL)
            .border_style(Style::default().fg(BLUE)))
        .highlight_style(Style::default().bg(BLUE).fg(BG).bold())
        .highlight_symbol("▶ ");

    f.render_stateful_widget(list, area, &mut app.cursor_shader_state);
}

fn render_screen_shaders(f: &mut Frame, app: &mut App, area: Rect) {
    let mut all_items = Vec::new();

    // Add combos first (separator + combos)
    all_items.push("── COMBOS ──".to_string());
    for (name, desc) in &app.screen_shader_combos {
        all_items.push(format!("{} - {}", name, desc));
    }

    // Add individual shaders (separator + shaders)
    all_items.push("".to_string());
    all_items.push("── INDIVIDUAL ──".to_string());
    for shader in &app.screen_shaders {
        all_items.push(shader.clone());
    }

    // Build the list items with proper styling
    let items: Vec<ListItem> = all_items
        .iter()
        .map(|item| {
            if item.starts_with("──") {
                // Separator styling
                ListItem::new(item.as_str()).style(Style::default().fg(YELLOW).bold())
            } else if item.is_empty() {
                // Empty line
                ListItem::new("")
            } else {
                // Regular item
                ListItem::new(item.as_str())
            }
        })
        .collect();

    let list = List::new(items)
        .block(Block::default()
            .title(" 🌊 SCREEN EFFECTS ")
            .title_style(Style::default().fg(MAGENTA).bold())
            .borders(Borders::ALL)
            .border_style(Style::default().fg(BLUE)))
        .highlight_style(Style::default().bg(MAGENTA).fg(BG).bold())
        .highlight_symbol("▶ ");

    f.render_stateful_widget(list, area, &mut app.screen_shader_state);
}

fn render_window(f: &mut Frame, app: &mut App, area: Rect) {
    let items: Vec<ListItem> = app.window_options.iter().map(|o| ListItem::new(o.as_str()).style(if o.starts_with("--") { Style::default().fg(YELLOW).bold() } else if o.is_empty() { Style::default() } else { Style::default().fg(Color::White) })).collect();
    f.render_stateful_widget(List::new(items).block(Block::default().title(" WINDOW ").title_style(Style::default().fg(MAGENTA).bold()).borders(Borders::ALL).border_style(Style::default().fg(BLUE))).highlight_style(Style::default().bg(BLUE).fg(BG).bold()).highlight_symbol("> "), area, &mut app.window_state);
}

fn render_shell(f: &mut Frame, app: &mut App, area: Rect) {
    let items: Vec<ListItem> = app.shell_options.iter().map(|o| ListItem::new(o.as_str()).style(if o.starts_with("--") { Style::default().fg(YELLOW).bold() } else if o.is_empty() { Style::default() } else { Style::default().fg(Color::White) })).collect();
    f.render_stateful_widget(List::new(items).block(Block::default().title(" SHELL ").title_style(Style::default().fg(MAGENTA).bold()).borders(Borders::ALL).border_style(Style::default().fg(BLUE))).highlight_style(Style::default().bg(BLUE).fg(BG).bold()).highlight_symbol("> "), area, &mut app.shell_state);
}

fn render_appearance(f: &mut Frame, app: &mut App, area: Rect) {
    let items: Vec<ListItem> = app.appearance_options
        .iter()
        .map(|option| {
            let style = if option.starts_with("──") {
                Style::default().fg(YELLOW).bold()
            } else {
                Style::default().fg(Color::White)
            };
            ListItem::new(option.as_str()).style(style)
        })
        .collect();

    let list = List::new(items)
        .block(Block::default()
            .title(" 🎨 APPEARANCE ")
            .title_style(Style::default().fg(MAGENTA).bold())
            .borders(Borders::ALL)
            .border_style(Style::default().fg(BLUE)))
        .highlight_style(Style::default().bg(BLUE).fg(BG).bold())
        .highlight_symbol("▶ ");

    f.render_stateful_widget(list, area, &mut app.appearance_state);
}

fn render_mouse(f: &mut Frame, app: &mut App, area: Rect) {
    let items: Vec<ListItem> = app.mouse_options
        .iter()
        .map(|option| {
            let style = if option.starts_with("--") {
                Style::default().fg(YELLOW).bold()
            } else if option.is_empty() {
                Style::default()
            } else {
                Style::default().fg(Color::White)
            };
            ListItem::new(option.as_str()).style(style)
        })
        .collect();

    let list = List::new(items)
        .block(Block::default()
            .title(" MOUSE ")
            .title_style(Style::default().fg(MAGENTA).bold())
            .borders(Borders::ALL)
            .border_style(Style::default().fg(BLUE)))
        .highlight_style(Style::default().bg(BLUE).fg(BG).bold())
        .highlight_symbol("> ");

    f.render_stateful_widget(list, area, &mut app.mouse_state);
}

fn render_clipboard(f: &mut Frame, app: &mut App, area: Rect) {
    let items: Vec<ListItem> = app.clipboard_options.iter().map(|o| ListItem::new(o.as_str()).style(if o.starts_with("--") { Style::default().fg(YELLOW).bold() } else if o.is_empty() { Style::default() } else { Style::default().fg(Color::White) })).collect();
    f.render_stateful_widget(List::new(items).block(Block::default().title(" CLIPBOARD ").title_style(Style::default().fg(MAGENTA).bold()).borders(Borders::ALL).border_style(Style::default().fg(BLUE))).highlight_style(Style::default().bg(BLUE).fg(BG).bold()).highlight_symbol("> "), area, &mut app.clipboard_state);
}

fn render_config(f: &mut Frame, app: &mut App, area: Rect) {
    let config_text = Text::from(app.current_config.as_str());

    let paragraph = Paragraph::new(config_text)
        .block(Block::default()
            .title(" ⚙️  CURRENT CONFIG ")
            .title_style(Style::default().fg(MAGENTA).bold())
            .borders(Borders::ALL)
            .border_style(Style::default().fg(BLUE)))
        .wrap(Wrap { trim: false });

    f.render_widget(paragraph, area);
}
