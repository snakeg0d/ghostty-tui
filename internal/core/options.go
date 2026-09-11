package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Environment variable names honoured by the engine. The override variables
// exist so the studio and its tests can operate on fixtures without touching
// the user's live configuration.
const (
	envConfigPath      = "GHOSTTY_CONFIG_PATH"
	envStateDir        = "GHOSTTY_TUI_STATE_DIR"
	envGhosttyBin      = "GHOSTTY_BIN"
	envGhosttyBinAlias = "GHOSTTY_TUI_GHOSTTY_PATH"
	envAssets          = "GHOSTTY_TUI_ASSETS"
	envResources       = "GHOSTTY_RESOURCES_DIR"
	envXDGConfigHome   = "XDG_CONFIG_HOME"
	envXDGStateHome    = "XDG_STATE_HOME"

	ghosttyBundleBinary  = "/Applications/Ghostty.app/Contents/MacOS/ghostty"
	ghosttyBundleResRoot = "/Applications/Ghostty.app/Contents/Resources/ghostty"
	managedDirName       = "ghostty-studio"
	// managedFileName is the include file the studio writes and owns.
	managedFileName = "studio.ghostty"
)

// configFileNames lists the configuration file names Ghostty reads in a
// configuration directory, most specific first.
var configFileNames = []string{"config.ghostty", "config"}

// paths is the resolved filesystem surface of one engine instance.
type paths struct {
	// configOverride is an explicitly requested configuration file. When set,
	// the engine manages exactly this file instead of the platform defaults.
	configOverride string
	configExplicit bool

	stateDir   string
	ghosttyBin string
	assetDir   string
	// resourcesDir is the installed Ghostty resources directory (holding
	// themes/, doc/ and shell-integration/).
	resourcesDir string

	home      string
	xdgConfig string
	lookPath  func(string) (string, error)
}

// resolvePaths turns Options plus the process environment into concrete paths.
// It performs no writes.
func resolvePaths(opts Options, getenv func(string) string, home string, lookPath func(string) (string, error)) paths {
	p := paths{home: home, lookPath: lookPath}
	xdgConfig := getenv(envXDGConfigHome)
	if xdgConfig == "" && home != "" {
		xdgConfig = filepath.Join(home, ".config")
	}
	p.xdgConfig = xdgConfig

	if opts.ConfigPath != "" {
		p.configOverride = expandHome(opts.ConfigPath, home)
		p.configExplicit = true
	} else if env := getenv(envConfigPath); env != "" {
		p.configOverride = expandHome(env, home)
		p.configExplicit = true
	}

	switch {
	case opts.StateDir != "":
		p.stateDir = expandHome(opts.StateDir, home)
	case getenv(envStateDir) != "":
		p.stateDir = expandHome(getenv(envStateDir), home)
	case getenv(envXDGStateHome) != "":
		p.stateDir = filepath.Join(expandHome(getenv(envXDGStateHome), home), managedDirName)
	case xdgConfig != "":
		p.stateDir = filepath.Join(xdgConfig, managedDirName)
	}

	switch {
	case opts.GhosttyPath != "":
		p.ghosttyBin = expandHome(opts.GhosttyPath, home)
	case getenv(envGhosttyBin) != "":
		p.ghosttyBin = expandHome(getenv(envGhosttyBin), home)
	case getenv(envGhosttyBinAlias) != "":
		p.ghosttyBin = expandHome(getenv(envGhosttyBinAlias), home)
	}

	if opts.AssetDir != "" {
		p.assetDir = expandHome(opts.AssetDir, home)
	} else if env := getenv(envAssets); env != "" {
		p.assetDir = expandHome(env, home)
	}

	p.resourcesDir = discoverResourcesDir(getenv(envResources), p.ghosttyBin)
	return p
}

// discoverResourcesDir locates the installed Ghostty resources directory.
func discoverResourcesDir(override, binary string) string {
	if override != "" {
		if info, err := os.Stat(override); err == nil && info.IsDir() {
			return override
		}
	}
	if binary != "" && strings.HasSuffix(binary, ".app/Contents/MacOS/ghostty") {
		// <bundle>/Contents/MacOS/ghostty -> <bundle>/Contents/Resources/ghostty
		candidate := filepath.Join(filepath.Dir(filepath.Dir(binary)), "Resources", "ghostty")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	if info, err := os.Stat(bundleResources); err == nil && info.IsDir() {
		return bundleResources
	}
	return ""
}

// configDirs lists the directories Ghostty reads configuration from, in load
// order. On macOS the XDG directory is read first and the Application Support
// directory last, so the last file wins for scalar keys and repeated keys
// append.
func (p paths) configDirs() []string {
	var dirs []string
	if p.xdgConfig != "" {
		dirs = append(dirs, filepath.Join(p.xdgConfig, "ghostty"))
	}
	if p.home != "" {
		dirs = append(dirs, filepath.Join(p.home, "Library", "Application Support", "com.mitchellh.ghostty"))
	}
	return dirs
}

// existingConfigFiles returns the configuration files Ghostty reads, in load
// order, plus the files that exist but are shadowed by a more specific name in
// the same directory.
func (p paths) existingConfigFiles() (files, shadowed []string) {
	for _, dir := range p.configDirs() {
		var found bool
		for _, name := range configFileNames {
			path := filepath.Join(dir, name)
			info, err := os.Stat(path)
			if err != nil || !info.Mode().IsRegular() {
				continue
			}
			if !found {
				files = append(files, path)
				found = true
				continue
			}
			shadowed = append(shadowed, path)
		}
	}
	return files, shadowed
}

// defaultConfigPath is the file created when no configuration exists yet. A
// macOS application bundle install keeps everything in Application Support; any
// other installation uses the documented XDG location.
func (p paths) defaultConfigPath() string {
	dirs := p.configDirs()
	if len(dirs) == 0 {
		return ""
	}
	if strings.Contains(p.ghosttyBin, ".app/Contents/MacOS/") && len(dirs) > 1 {
		return filepath.Join(dirs[len(dirs)-1], "config.ghostty")
	}
	return filepath.Join(dirs[0], "config.ghostty")
}

// managedPath returns the include file the studio owns, located next to the
// configuration file that receives the include directive so that relative
// includes elsewhere in the graph keep resolving.
func managedPath(configPath string) string {
	if configPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(configPath), managedFileName)
}

// themeDirs lists the directories searched for theme files: the user theme
// directories first, then the installed Ghostty themes.
func (p paths) themeDirs() []themeDir {
	var dirs []themeDir
	if p.xdgConfig != "" {
		dirs = append(dirs, themeDir{
			path:   filepath.Join(p.xdgConfig, "ghostty", "themes"),
			source: "User theme",
		})
	}
	if p.home != "" {
		dirs = append(dirs, themeDir{
			path:   filepath.Join(p.home, "Library", "Application Support", "com.mitchellh.ghostty", "themes"),
			source: "User theme",
		})
	}
	if p.resourcesDir != "" {
		dirs = append(dirs, themeDir{
			path:   filepath.Join(p.resourcesDir, "themes"),
			source: "Ghostty bundle",
		})
	}
	return dirs
}

// shaderDirs lists the directories searched for custom shader assets.
func (p paths) shaderDirs() []themeDir {
	var dirs []themeDir
	if p.assetDir != "" {
		dirs = append(dirs, themeDir{path: filepath.Join(p.assetDir, "shaders"), source: "Bundled effect"})
	}
	dirs = append(dirs, themeDir{path: filepath.Join(p.stateDir, "shaders"), source: "User shader"})
	if p.xdgConfig != "" {
		dirs = append(dirs, themeDir{path: filepath.Join(p.xdgConfig, "ghostty", "shaders"), source: "User shader"})
	}
	if p.home != "" {
		dirs = append(dirs, themeDir{path: filepath.Join(p.home, "Library", "Application Support", "com.mitchellh.ghostty", "shaders"), source: "User shader"})
	}
	return dirs
}

// docPath returns the installed Ghostty configuration reference manpage.
func (p paths) docPath() string {
	if p.resourcesDir == "" {
		return ""
	}
	return filepath.Join(p.resourcesDir, "doc", "ghostty.5.md")
}

// themeDir pairs a directory with the label used for catalog sources.
type themeDir struct {
	path   string
	source string
}

// themeFileDirs is the subset of dirs that exist on disk.
func themeFileDirs(dirs []themeDir) []themeDir {
	var out []themeDir
	for _, dir := range dirs {
		if info, err := os.Stat(dir.path); err == nil && info.IsDir() {
			out = append(out, dir)
		}
	}
	return out
}

// expandHome expands a leading ~ or ~/ using home.
func expandHome(path, home string) string {
	if home == "" {
		return path
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

// requireExecutable verifies an explicitly configured binary path.
func requireExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("ghostty executable %s: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("ghostty executable %s is a directory", path)
	}
	if info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("ghostty executable %s is not executable", path)
	}
	return nil
}
