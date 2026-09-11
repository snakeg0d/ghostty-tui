# Verified references

Checked 11 September 2026 against the installed software and primary sources.

## Charm

- [Bubble Tea](https://github.com/charmbracelet/bubbletea): `charm.land/bubbletea/v2` **2.0.9**, verified with `go list -m -json ...@latest`. The GitHub webpage release badge lagged the module registry.
- [Bubbles](https://github.com/charmbracelet/bubbles): `charm.land/bubbles/v2` **2.2.1**.
- [Lip Gloss](https://github.com/charmbracelet/lipgloss): `charm.land/lipgloss/v2` **2.0.6**.
- [Freeze](https://github.com/charmbracelet/freeze): **0.2.2**, development-only capture of actual terminal output. It is not a runtime dependency or a substitute for testing input.

## Ghostty

Installed binary: `/Applications/Ghostty.app/Contents/MacOS/ghostty`, **1.3.1**, CoreText/Metal. Bundled manual: `/Applications/Ghostty.app/Contents/Resources/ghostty/doc/ghostty.5.md` and `ghostty.1.md`.

- [Configuration](https://ghostty.org/docs/config)
- [Configuration reference](https://ghostty.org/docs/config/reference)
- [Themes](https://ghostty.org/docs/features/theme)
- [Shell integration](https://ghostty.org/docs/features/shell-integration)
- [1.3 release notes](https://ghostty.org/docs/install/release-notes/1-3-0)
- [1.2 release notes](https://ghostty.org/docs/install/release-notes/1-2-0)

Native experiments verified that `config.ghostty` wins over `config` in the same directory; included files load after parent settings; and the macOS config overrides XDG settings. Both config locations exist on Daniel's machine. Experiments used temporary XDG directories and read-only native commands.

The diagnostic CLI has its own argument parser. `+validate-config --config-file=PATH` works. Adding `--config-default-files=false` to that action is rejected. `+show-config` accepts `--default`, `--changes-only`, `--docs`; it rejects `--config-file`. These flags must not be confused with GUI launcher configuration flags.

There are **463 bundled theme files** on the test machine. Its `+list-themes --plain` can exit 1 after partial output; filesystem catalog discovery is necessary. `+list-fonts` completed successfully. GLSL compilation is a renderer operation; config validation alone does not verify it.

## Font sources

Homebrew's native JSON catalog confirmed these casks exist and were not installed at the start of this build. Installations retain Homebrew ownership and upstream licenses.

| Font | Verified cask | Version at audit | Upstream |
|---|---|---|---|
| Monaspace | [font-monaspace](https://formulae.brew.sh/cask/font-monaspace) | 1.400 | [GitHub Next](https://github.com/githubnext/monaspace) |
| Maple Mono | [font-maple-mono](https://formulae.brew.sh/cask/font-maple-mono) | 7.9 | [Maple Mono](https://github.com/subframe7536/maple-font) |
| Commit Mono | [font-commit-mono](https://formulae.brew.sh/cask/font-commit-mono) | 1.143 | [Commit Mono](https://github.com/eigilnikolajsen/commit-mono) |
| Iosevka | [font-iosevka](https://formulae.brew.sh/cask/font-iosevka) | managed by Homebrew | [Iosevka](https://github.com/be5invis/Iosevka) |
| JetBrains Mono | [font-jetbrains-mono](https://formulae.brew.sh/cask/font-jetbrains-mono) | managed by Homebrew | [JetBrains](https://github.com/JetBrains/JetBrainsMono) |

## Recovered project

Original app: `~/dev/_archive/dev-root-gade-era-20260707/dotfiles/ghostty/ghostty-tui`. Original manager and shader files are in its parent directory. The archived implementation and both personal Ghostty config files were hash-checked before and during the build. All 21 recovered GLSL files were copied byte for byte into `assets/legacy-shaders` with their source comments retained.
