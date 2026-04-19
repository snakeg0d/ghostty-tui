#!/bin/bash
# SPDX-License-Identifier: MIT
# Part of ghostty-tui — https://github.com/snakeg0d/ghostty-tui
#
# Ghostty Manager — write-through helper for themes, shaders, configs,
# and other Ghostty settings. Invoked by the ghostty-tui binary; can
# also be run standalone (e.g. `ghostty-manager.sh theme Dracula`).
#
# Usage: ghostty-manager.sh <command> [args]

set -e

# ---- Platform detection --------------------------------------------------
OS="$(uname -s)"
case "$OS" in
    Darwin)
        GHOSTTY_CONFIG="${GHOSTTY_CONFIG_PATH:-$HOME/Library/Application Support/com.mitchellh.ghostty/config}"
        SED_INPLACE="sed -i ''"
        ;;
    Linux)
        if [ -n "$XDG_CONFIG_HOME" ]; then
            GHOSTTY_CONFIG="${GHOSTTY_CONFIG_PATH:-$XDG_CONFIG_HOME/ghostty/config}"
        else
            GHOSTTY_CONFIG="${GHOSTTY_CONFIG_PATH:-$HOME/.config/ghostty/config}"
        fi
        SED_INPLACE="sed -i"
        ;;
    *)
        echo "Unsupported platform: $OS"
        exit 1
        ;;
esac

# ---- Asset-root resolution ------------------------------------------------
# Priority:
#   1. GHOSTTY_TUI_ASSETS                       (explicit env override)
#   2. <script-dir>/../assets                   (in-repo layout)
#   3. <script-dir>/../share/ghostty-tui/assets (installed layout)
#   4. $XDG_DATA_HOME/ghostty-tui/assets        (user-local fallback)
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
if [ -n "${GHOSTTY_TUI_ASSETS:-}" ] && [ -d "$GHOSTTY_TUI_ASSETS" ]; then
    ASSETS_DIR="$GHOSTTY_TUI_ASSETS"
elif [ -d "$SCRIPT_DIR/../assets" ]; then
    ASSETS_DIR="$(cd "$SCRIPT_DIR/../assets" && pwd)"
elif [ -d "$SCRIPT_DIR/../share/ghostty-tui/assets" ]; then
    ASSETS_DIR="$(cd "$SCRIPT_DIR/../share/ghostty-tui/assets" && pwd)"
else
    ASSETS_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/ghostty-tui/assets"
fi

SHADERS_DIR="$ASSETS_DIR/shaders/cursors"
SCREEN_SHADERS_DIR="$ASSETS_DIR/shaders/screen"
CONFIGS_DIR="$ASSETS_DIR/configs"
THEMES_DIR="${GHOSTTY_THEMES_DIR:-$HOME/.config/ghostty/themes}"

# Legacy alias kept so downstream command blocks still referencing
# $DOTFILES_DIR continue to resolve correctly.
DOTFILES_DIR="$ASSETS_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# ============================================
# THEME MANAGEMENT
# ============================================

list_themes() {
    echo -e "${CYAN}═══ Built-in Themes ═══${NC}"
    ghostty +list-themes | head -20
    echo "..."
    echo -e "\n${CYAN}═══ Custom Themes ═══${NC}"
    ls -1 "$THEMES_DIR" | grep -v README.md || echo "(none)"
}

set_theme() {
    # Use "$*" to combine all arguments into a single theme name (handles spaces)
    local theme="$*"
    if [ -z "$theme" ]; then
        echo -e "${RED}Error: No theme specified${NC}"
        echo "Usage: ghostty-manager.sh theme <theme-name>"
        return 1
    fi

    # Update config file
    if grep -q "^theme = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^theme = .*/theme = $theme/" "$GHOSTTY_CONFIG"
    else
        echo "theme = $theme" >> "$GHOSTTY_CONFIG"
    fi

    echo -e "${GREEN}✓ Theme set to: $theme${NC}"
    echo -e "${YELLOW}Note: Restart Ghostty or open new window to see changes${NC}"
}

# ============================================
# SHADER MANAGEMENT
# ============================================

list_shaders() {
    echo -e "${CYAN}═══ Available Cursor Shaders ═══${NC}"
    local i=1
    for shader in "$SHADERS_DIR"/*.glsl; do
        local name=$(basename "$shader" .glsl)
        echo -e "${BLUE}$i.${NC} $name"
        ((i++))
    done
}

enable_shader() {
    # Use "$*" to handle shader names with spaces
    local shader_name="$*"
    if [ -z "$shader_name" ]; then
        echo -e "${RED}Error: No shader specified${NC}"
        list_shaders
        return 1
    fi

    local shader_path="$SHADERS_DIR/${shader_name}.glsl"
    if [ ! -f "$shader_path" ]; then
        echo -e "${RED}Error: Shader not found: $shader_name${NC}"
        list_shaders
        return 1
    fi

    # Remove existing cursor shader config
    $SED_INPLACE '/^custom-shader.*cursors/d' "$GHOSTTY_CONFIG"
    $SED_INPLACE '/^custom-shader-animation = /d' "$GHOSTTY_CONFIG"

    # Add new cursor shader
    echo "custom-shader = $shader_path" >> "$GHOSTTY_CONFIG"
    echo "custom-shader-animation = true" >> "$GHOSTTY_CONFIG"

    echo -e "${GREEN}✓ Cursor shader enabled: $shader_name${NC}"
    echo -e "${YELLOW}Note: Restart Ghostty or open new window to see changes${NC}"
}

disable_shader() {
    # Remove cursor shaders only (keep screen shaders)
    $SED_INPLACE '/^custom-shader.*cursors/d' "$GHOSTTY_CONFIG"
    # Only remove animation if no shaders remain
    if ! grep -q "^custom-shader = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE '/^custom-shader-animation = /d' "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}✓ Cursor shader disabled${NC}"
    echo -e "${YELLOW}Note: Restart Ghostty or open new window to see changes${NC}"
}

# ============================================
# SCREEN SHADER MANAGEMENT (CRT, GLITCH, etc.)
# ============================================

list_screen_shaders() {
    echo -e "${CYAN}═══ Available Screen Shaders ═══${NC}"
    echo -e "${YELLOW}Popular Combos (from catskull.net):${NC}"
    echo -e "  ${MAGENTA}crt-classic${NC}    - tft + bettercrt"
    echo -e "  ${MAGENTA}crt-bloom${NC}      - tft + retro-terminal + bloom"
    echo -e "  ${MAGENTA}drunk-retro${NC}    - drunkard + retro-terminal + bloom"
    echo -e "  ${MAGENTA}glitch-water${NC}   - glitchy + bettercrt + water + bloom"
    echo ""
    echo -e "${YELLOW}Individual Shaders:${NC}"
    local i=1
    for shader in "$SCREEN_SHADERS_DIR"/*.glsl; do
        local name=$(basename "$shader" .glsl)
        echo -e "${BLUE}$i.${NC} $name"
        ((i++))
    done
}

enable_screen_shader() {
    local shader_names=("$@")

    if [ ${#shader_names[@]} -eq 0 ]; then
        echo -e "${RED}Error: No shader specified${NC}"
        list_screen_shaders
        return 1
    fi

    # Handle preset combos
    case "${shader_names[0]}" in
        crt-classic)
            shader_names=("tft" "bettercrt")
            ;;
        crt-bloom)
            shader_names=("tft" "retro-terminal" "bloom")
            ;;
        drunk-retro)
            shader_names=("drunkard" "retro-terminal" "bloom")
            ;;
        glitch-water)
            shader_names=("glitchy" "bettercrt" "water" "bloom")
            ;;
    esac

    # Remove existing screen shaders (but keep cursor shaders)
    $SED_INPLACE '/^custom-shader.*screen/d' "$GHOSTTY_CONFIG"
    $SED_INPLACE '/^custom-shader-animation = /d' "$GHOSTTY_CONFIG"

    # Add new screen shaders
    local added_count=0
    for shader_name in "${shader_names[@]}"; do
        local shader_path="$SCREEN_SHADERS_DIR/${shader_name}.glsl"
        if [ ! -f "$shader_path" ]; then
            echo -e "${YELLOW}Warning: Shader not found: $shader_name${NC}"
            continue
        fi
        echo "custom-shader = $shader_path" >> "$GHOSTTY_CONFIG"
        ((added_count++))
    done

    if [ $added_count -eq 0 ]; then
        echo -e "${RED}Error: No valid shaders found${NC}"
        return 1
    fi

    # Enable animation for screen shaders
    echo "custom-shader-animation = true" >> "$GHOSTTY_CONFIG"

    echo -e "${GREEN}✓ Screen shader(s) enabled: ${shader_names[*]}${NC}"
    echo -e "${YELLOW}Note: Restart Ghostty or open new window to see changes${NC}"
}

disable_screen_shader() {
    # Remove screen shaders only (keep cursor shaders)
    $SED_INPLACE '/^custom-shader.*screen/d' "$GHOSTTY_CONFIG"
    # Only remove animation if no shaders remain
    if ! grep -q "^custom-shader = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE '/^custom-shader-animation = /d' "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}✓ Screen shaders disabled${NC}"
    echo -e "${YELLOW}Note: Restart Ghostty or open new window to see changes${NC}"
}

# ============================================
# PRESET CONFIG MANAGEMENT
# ============================================

list_configs() {
    echo -e "${CYAN}═══ Available Preset Configs ═══${NC}"
    local i=1
    for config in "$CONFIGS_DIR"/*.config; do
        local name=$(basename "$config" .config)
        # Get first comment line as description
        local desc=$(grep "^#" "$config" | head -1 | sed 's/^# *//')
        echo -e "${BLUE}$i.${NC} ${GREEN}$name${NC}"
        if [ -n "$desc" ]; then
            echo -e "    $desc"
        fi
        ((i++))
    done
}

load_config() {
    # Use "$*" to handle config names with spaces
    local config_name="$*"
    if [ -z "$config_name" ]; then
        echo -e "${RED}Error: No config specified${NC}"
        list_configs
        return 1
    fi

    local config_path="$CONFIGS_DIR/${config_name}.config"
    if [ ! -f "$config_path" ]; then
        config_path="$CONFIGS_DIR/$config_name"
    fi

    if [ ! -f "$config_path" ]; then
        echo -e "${RED}Error: Config not found: $config_name${NC}"
        list_configs
        return 1
    fi

    # Backup current config
    cp "$GHOSTTY_CONFIG" "$GHOSTTY_CONFIG.backup.$(date +%Y%m%d-%H%M%S)"

    # Copy preset config
    cp "$config_path" "$GHOSTTY_CONFIG"

    echo -e "${GREEN}✓ Config loaded: $config_name${NC}"
    echo -e "${YELLOW}Note: Previous config backed up${NC}"
    echo -e "${YELLOW}Note: Restart Ghostty or open new window to see changes${NC}"
}

# ============================================
# APPEARANCE MANAGEMENT
# ============================================

set_opacity() {
    local opacity="$*"
    if [ -z "$opacity" ]; then
        echo -e "${RED}Error: No opacity value specified${NC}"
        echo "Usage: ghostty-manager.sh opacity <0.0-1.0>"
        return 1
    fi

    # Update config
    if grep -q "^background-opacity = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^background-opacity = .*/background-opacity = $opacity/" "$GHOSTTY_CONFIG"
    else
        echo "background-opacity = $opacity" >> "$GHOSTTY_CONFIG"
    fi

    echo -e "${GREEN}✓ Opacity set to: $opacity${NC}"
    echo -e "${YELLOW}Note: Restart Ghostty or open new window to see changes${NC}"
}

set_blur() {
    local blur="$*"
    if [ -z "$blur" ]; then
        echo -e "${RED}Error: No blur value specified${NC}"
        echo "Usage: ghostty-manager.sh blur <0-100>"
        return 1
    fi

    # Update config
    if grep -q "^background-blur = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^background-blur = .*/background-blur = $blur/" "$GHOSTTY_CONFIG"
    else
        echo "background-blur = $blur" >> "$GHOSTTY_CONFIG"
    fi

    echo -e "${GREEN}✓ Blur set to: $blur${NC}"
    echo -e "${YELLOW}Note: Restart Ghostty or open new window to see changes${NC}"
}

apply_appearance_preset() {
    local preset="$*"

    case "$preset" in
        default)
            # Liquid glass: TokyoNight with opacity and blur
            set_theme "TokyoNight"
            set_opacity "0.8"
            set_blur "20"
            disable_screen_shader
            disable_shader
            echo -e "${GREEN}✓ Applied Default (Liquid Glass) preset${NC}"
            ;;
        solid)
            # Solid: no transparency
            set_opacity "1.0"
            set_blur "0"
            echo -e "${GREEN}✓ Applied Solid preset${NC}"
            ;;
        *)
            echo -e "${RED}Error: Unknown preset: $preset${NC}"
            echo "Available presets: default, solid"
            return 1
            ;;
    esac
}

# ============================================
# PADDING MANAGEMENT
# ============================================

set_padding_x() {
    local padding="$1"
    if grep -q "^window-padding-x = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^window-padding-x = .*/window-padding-x = $padding/" "$GHOSTTY_CONFIG"
    else
        echo "window-padding-x = $padding" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}Padding X: $padding${NC}"
}

set_padding_y() {
    local padding="$1"
    if grep -q "^window-padding-y = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^window-padding-y = .*/window-padding-y = $padding/" "$GHOSTTY_CONFIG"
    else
        echo "window-padding-y = $padding" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}Padding Y: $padding${NC}"
}

# ============================================
# FONT MANAGEMENT
# ============================================

set_font_family() {
    local font="$*"
    if [ -z "$font" ]; then
        echo -e "${RED}Error: No font specified${NC}"
        return 1
    fi

    if grep -q "^font-family = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^font-family = .*/font-family = $font/" "$GHOSTTY_CONFIG"
    else
        echo "font-family = $font" >> "$GHOSTTY_CONFIG"
    fi

    echo -e "${GREEN}Font family set to: $font${NC}"
}

set_font_size() {
    local size="$1"
    if grep -q "^font-size = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^font-size = .*/font-size = $size/" "$GHOSTTY_CONFIG"
    else
        echo "font-size = $size" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}Font size set to: $size${NC}"
}

set_font_weight() {
    local weight="$1"
    # For bold, we use font-style; for normal, we remove the setting
    if [ "$weight" = "bold" ]; then
        if grep -q "^font-style = " "$GHOSTTY_CONFIG"; then
            $SED_INPLACE "s/^font-style = .*/font-style = Bold/" "$GHOSTTY_CONFIG"
        else
            echo "font-style = Bold" >> "$GHOSTTY_CONFIG"
        fi
    else
        $SED_INPLACE '/^font-style = /d' "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}Font weight set to: $weight${NC}"
}

toggle_ligatures() {
    local state="$1"
    if [ "$state" = "on" ]; then
        $SED_INPLACE '/^font-feature = -calt/d' "$GHOSTTY_CONFIG"
    else
        if ! grep -q "^font-feature = -calt" "$GHOSTTY_CONFIG"; then
            echo "font-feature = -calt" >> "$GHOSTTY_CONFIG"
        fi
    fi
    echo -e "${GREEN}Ligatures: $state${NC}"
}

# ============================================
# CURSOR MANAGEMENT
# ============================================

set_cursor_style() {
    local style="$1"
    # Convert display name to config value
    case "$style" in
        block) style="block" ;;
        bar) style="bar" ;;
        underline) style="underline" ;;
        block_hollow) style="block_hollow" ;;
    esac

    if grep -q "^cursor-style = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^cursor-style = .*/cursor-style = $style/" "$GHOSTTY_CONFIG"
    else
        echo "cursor-style = $style" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}Cursor style set to: $style${NC}"
}

set_cursor_color() {
    local color="$1"
    if [ "$color" = "theme" ]; then
        $SED_INPLACE '/^cursor-color = /d' "$GHOSTTY_CONFIG"
        echo -e "${GREEN}Cursor color: match theme${NC}"
    else
        if grep -q "^cursor-color = " "$GHOSTTY_CONFIG"; then
            $SED_INPLACE "s/^cursor-color = .*/cursor-color = $color/" "$GHOSTTY_CONFIG"
        else
            echo "cursor-color = $color" >> "$GHOSTTY_CONFIG"
        fi
        echo -e "${GREEN}Cursor color set to: $color${NC}"
    fi
}

set_cursor_blink() {
    local state="$1"
    if grep -q "^cursor-style-blink = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^cursor-style-blink = .*/cursor-style-blink = $state/" "$GHOSTTY_CONFIG"
    else
        echo "cursor-style-blink = $state" >> "$GHOSTTY_CONFIG"
    fi

    # Also update shell-integration-features if turning off
    if [ "$state" = "false" ]; then
        if grep -q "^shell-integration-features = " "$GHOSTTY_CONFIG"; then
            $SED_INPLACE "s/^shell-integration-features = .*/shell-integration-features = no-cursor/" "$GHOSTTY_CONFIG"
        else
            echo "shell-integration-features = no-cursor" >> "$GHOSTTY_CONFIG"
        fi
    fi
    echo -e "${GREEN}Cursor blink: $state${NC}"
}

set_cursor_opacity() {
    local opacity="$1"
    if grep -q "^cursor-opacity = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^cursor-opacity = .*/cursor-opacity = $opacity/" "$GHOSTTY_CONFIG"
    else
        echo "cursor-opacity = $opacity" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}Cursor opacity: $opacity${NC}"
}

# ============================================
# MOUSE MANAGEMENT
# ============================================

set_mouse_hide() {
    local state="$1"
    if grep -q "^mouse-hide-while-typing = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^mouse-hide-while-typing = .*/mouse-hide-while-typing = $state/" "$GHOSTTY_CONFIG"
    else
        echo "mouse-hide-while-typing = $state" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}Mouse hide while typing: $state${NC}"
}

set_scroll_multiplier() {
    local mult="$1"
    if grep -q "^mouse-scroll-multiplier = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^mouse-scroll-multiplier = .*/mouse-scroll-multiplier = $mult/" "$GHOSTTY_CONFIG"
    else
        echo "mouse-scroll-multiplier = $mult" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}Scroll multiplier: $mult${NC}"
}

set_copy_on_select() {
    local mode="$1"
    if grep -q "^copy-on-select = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^copy-on-select = .*/copy-on-select = $mode/" "$GHOSTTY_CONFIG"
    else
        echo "copy-on-select = $mode" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}Copy on select: $mode${NC}"
}

set_focus_follows_mouse() {
    local state="$1"
    if grep -q "^focus-follows-mouse = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^focus-follows-mouse = .*/focus-follows-mouse = $state/" "$GHOSTTY_CONFIG"
    else
        echo "focus-follows-mouse = $state" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}Focus follows mouse: $state${NC}"
}

# ============================================
# CLIPBOARD MANAGEMENT
# ============================================

set_clipboard_read() {
    local mode="$1"
    if grep -q "^clipboard-read = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^clipboard-read = .*/clipboard-read = $mode/" "$GHOSTTY_CONFIG"
    else
        echo "clipboard-read = $mode" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}Clipboard read: $mode${NC}"
}

set_clipboard_write() {
    local mode="$1"
    if grep -q "^clipboard-write = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^clipboard-write = .*/clipboard-write = $mode/" "$GHOSTTY_CONFIG"
    else
        echo "clipboard-write = $mode" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}Clipboard write: $mode${NC}"
}

set_clipboard_trim() {
    local state="$1"
    if grep -q "^clipboard-trim-trailing-spaces = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^clipboard-trim-trailing-spaces = .*/clipboard-trim-trailing-spaces = $state/" "$GHOSTTY_CONFIG"
    else
        echo "clipboard-trim-trailing-spaces = $state" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}Clipboard trim: $state${NC}"
}

set_paste_protection() {
    local state="$1"
    if grep -q "^clipboard-paste-protection = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^clipboard-paste-protection = .*/clipboard-paste-protection = $state/" "$GHOSTTY_CONFIG"
    else
        echo "clipboard-paste-protection = $state" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}Paste protection: $state${NC}"
}

# ============================================
# WINDOW SIZE PRESETS
# ============================================

set_window_size() {
    local preset="$1"

    case "$preset" in
        small)
            local width=80
            local height=24
            ;;
        medium)
            local width=100
            local height=30
            ;;
        large)
            local width=120
            local height=40
            ;;
        ultrawide)
            local width=200
            local height=50
            ;;
        gostty)
            # Minimum for gostty animation
            local width=100
            local height=50
            ;;
        *)
            echo -e "${RED}Error: Unknown preset: $preset${NC}"
            echo "Available presets: small, medium, large, ultrawide, gostty"
            return 1
            ;;
    esac

    # Update config
    $SED_INPLACE "s/^window-width = .*/window-width = $width/" "$GHOSTTY_CONFIG"
    $SED_INPLACE "s/^window-height = .*/window-height = $height/" "$GHOSTTY_CONFIG"

    # Add if doesn't exist
    if ! grep -q "^window-width = " "$GHOSTTY_CONFIG"; then
        echo "window-width = $width" >> "$GHOSTTY_CONFIG"
    fi
    if ! grep -q "^window-height = " "$GHOSTTY_CONFIG"; then
        echo "window-height = $height" >> "$GHOSTTY_CONFIG"
    fi

    echo -e "${GREEN}✓ Window size set to $preset: ${width}x${height}${NC}"
    echo -e "${YELLOW}Note: Applies to new windows${NC}"
}

set_window_decoration() {
    local mode="$1"
    if grep -q "^window-decoration = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^window-decoration = .*/window-decoration = $mode/" "$GHOSTTY_CONFIG"
    else
        echo "window-decoration = $mode" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}✓ Window decoration: $mode${NC}"
}

set_tab_bar() {
    local mode="$1"
    if grep -q "^gtk-tabs-location = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^gtk-tabs-location = .*/gtk-tabs-location = $mode/" "$GHOSTTY_CONFIG"
    else
        echo "gtk-tabs-location = $mode" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}✓ Tab bar: $mode${NC}"
}

set_vsync() {
    local state="$1"
    if grep -q "^window-vsync = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^window-vsync = .*/window-vsync = $state/" "$GHOSTTY_CONFIG"
    else
        echo "window-vsync = $state" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}✓ VSync: $state${NC}"
}

set_fullscreen() {
    local state="$1"
    if grep -q "^fullscreen = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^fullscreen = .*/fullscreen = $state/" "$GHOSTTY_CONFIG"
    else
        echo "fullscreen = $state" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}✓ Fullscreen: $state${NC}"
}

set_save_state() {
    local mode="$1"
    if grep -q "^window-save-state = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^window-save-state = .*/window-save-state = $mode/" "$GHOSTTY_CONFIG"
    else
        echo "window-save-state = $mode" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}✓ Window save state: $mode${NC}"
}

# ============================================
# SHELL MANAGEMENT
# ============================================

set_shell_integration() {
    local mode="$1"
    if grep -q "^shell-integration = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^shell-integration = .*/shell-integration = $mode/" "$GHOSTTY_CONFIG"
    else
        echo "shell-integration = $mode" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}✓ Shell integration: $mode${NC}"
}

set_working_directory() {
    local mode="$1"
    if grep -q "^working-directory = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^working-directory = .*/working-directory = $mode/" "$GHOSTTY_CONFIG"
    else
        echo "working-directory = $mode" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}✓ Working directory: $mode${NC}"
}

set_confirm_close() {
    local mode="$1"
    if grep -q "^confirm-close-surface = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^confirm-close-surface = .*/confirm-close-surface = $mode/" "$GHOSTTY_CONFIG"
    else
        echo "confirm-close-surface = $mode" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}✓ Confirm close: $mode${NC}"
}

set_scrollback() {
    local lines="$1"
    local bytes=$((lines * 100))
    if grep -q "^scrollback-limit = " "$GHOSTTY_CONFIG"; then
        $SED_INPLACE "s/^scrollback-limit = .*/scrollback-limit = $bytes/" "$GHOSTTY_CONFIG"
    else
        echo "scrollback-limit = $bytes" >> "$GHOSTTY_CONFIG"
    fi
    echo -e "${GREEN}✓ Scrollback: ~$lines lines ($bytes bytes)${NC}"
}

# ============================================
# SPLASH SCREEN MANAGEMENT
# ============================================

toggle_splash() {
    local zshrc="$HOME/.zshrc"

    # Check if splash is currently enabled
    if grep -q "ghostty +boo" "$zshrc" && ! grep -q "^#.*ghostty +boo" "$zshrc"; then
        # Disable by commenting out
        $SED_INPLACE '/if.*ghostty.*boo/,/^fi/s/^/# /' "$zshrc"
        echo -e "${GREEN}✓ Splash screen disabled${NC}"
        echo -e "${YELLOW}Note: Open new terminal to see changes${NC}"
    else
        # Enable by uncommenting
        $SED_INPLACE '/# if.*ghostty.*boo/,/# fi/s/^# //' "$zshrc"
        echo -e "${GREEN}✓ Splash screen enabled${NC}"
        echo -e "${YELLOW}Note: Open new terminal to see changes${NC}"
    fi
}

# ============================================
# UTILITY FUNCTIONS
# ============================================

show_current() {
    echo -e "${CYAN}═══ Current Ghostty Configuration ═══${NC}\n"

    # Theme
    local theme=$(grep "^theme = " "$GHOSTTY_CONFIG" | sed 's/theme = //')
    echo -e "${GREEN}Theme:${NC} $theme"

    # Cursor Shader
    local shader=$(grep "^cursor-style-shader = " "$GHOSTTY_CONFIG" | sed 's/cursor-style-shader = //' | xargs basename 2>/dev/null || echo "(none)")
    echo -e "${GREEN}Cursor Shader:${NC} ${shader%.glsl}"

    # Screen Shaders
    local screen_shaders=$(grep "^custom-shader = " "$GHOSTTY_CONFIG" | sed 's/custom-shader = //' | xargs -I {} basename {} .glsl 2>/dev/null || echo "")
    if [ -n "$screen_shaders" ]; then
        echo -e "${GREEN}Screen Shaders:${NC} $screen_shaders"
    else
        echo -e "${GREEN}Screen Shaders:${NC} (none)"
    fi

    # Window size
    local width=$(grep "^window-width = " "$GHOSTTY_CONFIG" | sed 's/window-width = //')
    local height=$(grep "^window-height = " "$GHOSTTY_CONFIG" | sed 's/window-height = //')
    echo -e "${GREEN}Window Size:${NC} ${width}x${height}"

    # Font
    local font=$(grep "^font-family = " "$GHOSTTY_CONFIG" | head -1 | sed 's/font-family = //')
    echo -e "${GREEN}Font:${NC} $font"

    # Splash screen status
    if grep -q "ghostty +boo" "$HOME/.zshrc" && ! grep -q "^#.*ghostty +boo" "$HOME/.zshrc"; then
        echo -e "${GREEN}Splash Screen:${NC} enabled"
    else
        echo -e "${GREEN}Splash Screen:${NC} disabled"
    fi
}

show_help() {
    cat << 'EOF'
╔═══════════════════════════════════════════════════════════════╗
║                     GHOSTTY MANAGER                           ║
║              Unified Terminal Customization Tool              ║
╚═══════════════════════════════════════════════════════════════╝

USAGE:
    ghostty-manager.sh <command> [args]

COMMANDS:

  Theme Management:
    list-themes              List all available themes
    theme <name>             Switch to a specific theme

  Cursor Shader Management:
    list-shaders             List all available cursor shaders
    shader <name>            Enable a cursor shader
    shader-off               Disable cursor shader

  Screen Shader Management (CRT, Glitch, etc.):
    list-screen-shaders      List all available screen shaders
    screen-shader <name...>  Enable screen shader(s) - can stack multiple
    screen-shader-off        Disable all screen shaders

  Preset Configs:
    list-configs             List all preset configurations
    load-config <name>       Load a preset configuration

  Window Size:
    size <preset>            Set window size preset
                             (small|medium|large|ultrawide|gostty)

  Splash Screen:
    splash                   Toggle boot splash screen on/off

  Info:
    current                  Show current configuration
    help                     Show this help message

EXAMPLES:
    ghostty-manager.sh theme tokyonight-night
    ghostty-manager.sh shader ripple_cursor
    ghostty-manager.sh screen-shader crt-classic
    ghostty-manager.sh screen-shader bettercrt bloom
    ghostty-manager.sh load-config aecsar-wide
    ghostty-manager.sh size ultrawide
    ghostty-manager.sh splash

EOF
}

# ============================================
# MAIN DISPATCHER
# ============================================

main() {
    local command="$1"
    shift || true

    case "$command" in
        list-themes)
            list_themes
            ;;
        theme)
            set_theme "$@"
            ;;
        list-shaders)
            list_shaders
            ;;
        shader)
            enable_shader "$@"
            ;;
        shader-off)
            disable_shader
            ;;
        list-screen-shaders)
            list_screen_shaders
            ;;
        screen-shader)
            enable_screen_shader "$@"
            ;;
        screen-shader-off)
            disable_screen_shader
            ;;
        opacity)
            set_opacity "$@"
            ;;
        blur)
            set_blur "$@"
            ;;
        appearance)
            apply_appearance_preset "$@"
            ;;
        padding-x)
            set_padding_x "$@"
            ;;
        padding-y)
            set_padding_y "$@"
            ;;
        list-configs)
            list_configs
            ;;
        load-config)
            load_config "$@"
            ;;
        font-family)
            set_font_family "$@"
            ;;
        font-size)
            set_font_size "$@"
            ;;
        font-weight)
            set_font_weight "$@"
            ;;
        ligatures)
            toggle_ligatures "$@"
            ;;
        cursor-style)
            set_cursor_style "$@"
            ;;
        cursor-color)
            set_cursor_color "$@"
            ;;
        cursor-blink)
            set_cursor_blink "$@"
            ;;
        cursor-opacity)
            set_cursor_opacity "$@"
            ;;
        mouse-hide)
            set_mouse_hide "$@"
            ;;
        scroll-multiplier)
            set_scroll_multiplier "$@"
            ;;
        copy-on-select)
            set_copy_on_select "$@"
            ;;
        focus-follows-mouse)
            set_focus_follows_mouse "$@"
            ;;
        clipboard-read)
            set_clipboard_read "$@"
            ;;
        clipboard-write)
            set_clipboard_write "$@"
            ;;
        clipboard-trim)
            set_clipboard_trim "$@"
            ;;
        paste-protection)
            set_paste_protection "$@"
            ;;
        size)
            set_window_size "$@"
            ;;
        window-decoration)
            set_window_decoration "$@"
            ;;
        tab-bar)
            set_tab_bar "$@"
            ;;
        vsync)
            set_vsync "$@"
            ;;
        fullscreen)
            set_fullscreen "$@"
            ;;
        save-state)
            set_save_state "$@"
            ;;
        shell-integration)
            set_shell_integration "$@"
            ;;
        working-directory)
            set_working_directory "$@"
            ;;
        confirm-close)
            set_confirm_close "$@"
            ;;
        scrollback)
            set_scrollback "$@"
            ;;
        splash)
            toggle_splash
            ;;
        current)
            show_current
            ;;
        help|--help|-h|"")
            show_help
            ;;
        *)
            echo -e "${RED}Error: Unknown command: $command${NC}"
            echo "Run 'ghostty-manager.sh help' for usage information"
            exit 1
            ;;
    esac
}

main "$@"
