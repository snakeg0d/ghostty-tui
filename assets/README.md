# Assets and provenance

`shaders/` contains three original MIT-licensed Ghostty Studio effects. They are deliberately static and subtle. Ghostty's config validator checks paths and syntax of configuration; it does not prove GLSL compilation or GPU cost. Use native preview to judge rendering before applying effects to working terminals.

`legacy-shaders/` preserves Daniel's 21 recovered shaders byte for byte for personal continuity. Individual files carry their original source and license comments; these assets are not covered by the new shader license. They are not automatically enabled or included in the curated effect shelf. The archived original remains at `~/dev/_archive/dev-root-gade-era-20260707/dotfiles/ghostty/shaders/`.

Themes come from the user's installed Ghostty and local theme directories. Ghostty ships the iTerm2 Color Schemes catalog; the application does not need to download duplicate copies. Font installs use selected Homebrew casks so Homebrew retains package ownership and upstream notices.
