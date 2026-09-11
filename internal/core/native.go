package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Command is one external command invocation using an argument array. The
// engine never builds a shell command line, so configuration values and file
// names cannot be interpreted as shell syntax.
type Command struct {
	Path string
	Args []string
	Dir  string
	// Env holds additions on top of the inherited environment, in KEY=VALUE
	// form. Values are never interpolated into a shell.
	Env []string
}

// Runner executes commands. It is injectable so tests can exercise the engine
// without a Ghostty installation, without network access and without opening
// any window.
type Runner interface {
	Run(ctx context.Context, cmd Command) (stdout, stderr []byte, err error)
}

// osRunner is the production Runner.
type osRunner struct{}

func (osRunner) Run(ctx context.Context, cmd Command) ([]byte, []byte, error) {
	c := exec.CommandContext(ctx, cmd.Path, cmd.Args...)
	c.Dir = cmd.Dir
	if len(cmd.Env) > 0 {
		c.Env = append(os.Environ(), cmd.Env...)
	}
	var stdout, stderr strings.Builder
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	return []byte(stdout.String()), []byte(stderr.String()), err
}

// Command timeouts. Native listings are fast; a font install or a window launch
// is allowed to take longer.
const (
	probeTimeout   = 30 * time.Second
	installTimeout = 15 * time.Minute
	launchTimeout  = 30 * time.Second
)

// bundleBinary and bundleResources are the last-resort locations of an
// installed macOS application bundle. They are variables so tests can run
// against a temporary fixture machine instead of the real installation.
var (
	bundleBinary    = ghosttyBundleBinary
	bundleResources = ghosttyBundleResRoot
)

// exitCoder is implemented by *exec.ExitError and by the test runner's error.
type exitCoder interface{ ExitCode() int }

// exitCode extracts a process exit code, returning -1 for non-exit failures.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var coder exitCoder
	if errors.As(err, &coder) {
		return coder.ExitCode()
	}
	return -1
}

// nativeError is returned when a native probe cannot be trusted.
type nativeError struct {
	probe  string
	code   int
	stderr string
	stdout string
	err    error
}

func (e *nativeError) Error() string {
	detail := strings.TrimSpace(e.stderr)
	if detail == "" {
		detail = strings.TrimSpace(e.stdout)
	}
	if detail == "" && e.err != nil {
		detail = e.err.Error()
	}
	if detail == "" {
		detail = "no output"
	}
	if e.code >= 0 {
		return fmt.Sprintf("ghostty %s failed (exit %d): %s", e.probe, e.code, firstLine(detail))
	}
	return fmt.Sprintf("ghostty %s failed: %s", e.probe, firstLine(detail))
}

func (e *nativeError) Unwrap() error { return e.err }

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// errNoGhostty reports that no usable Ghostty executable was discovered.
var errNoGhostty = errors.New("ghostty executable not found")

// native wraps the installed Ghostty executable. A zero-value native with an
// empty bin is unusable but safe: every probe reports unavailable instead of
// inventing data.
type native struct {
	bin      string
	runner   Runner
	version  string
	warnings []string
}

// available reports whether a usable Ghostty executable was discovered.
func (n *native) available() bool { return n != nil && n.bin != "" }

// run executes a ghostty probe with the standard timeout.
func (n *native) run(ctx context.Context, probe string, args ...string) ([]byte, []byte, error) {
	if !n.available() {
		return nil, nil, errNoGhostty
	}
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	stdout, stderr, err := n.runner.Run(ctx, Command{Path: n.bin, Args: args})
	if err != nil {
		return stdout, stderr, &nativeError{probe: probe, code: exitCode(err), stderr: string(stderr), stdout: string(stdout), err: err}
	}
	return stdout, stderr, nil
}

// detectVersion runs `ghostty --version` and extracts the version string.
func (n *native) detectVersion(ctx context.Context) string {
	if !n.available() {
		return ""
	}
	stdout, _, err := n.run(ctx, "--version", "--version")
	if err != nil {
		n.warn(fmt.Sprintf("could not read the Ghostty version: %v", err))
		return ""
	}
	return parseVersion(string(stdout))
}

var versionPattern = regexp.MustCompile(`(?m)^\s*-\s*version:\s*(\S+)\s*$`)

// parseVersion reads the version from `ghostty --version` output.
func parseVersion(out string) string {
	if m := versionPattern.FindStringSubmatch(out); m != nil {
		return m[1]
	}
	if strings.HasPrefix(out, "Ghostty ") {
		return strings.TrimSpace(strings.TrimPrefix(firstLine(out), "Ghostty "))
	}
	return ""
}

func (n *native) warn(message string) {
	n.warnings = append(n.warnings, message)
}

// rest returns the accumulated warnings and clears them.
func (n *native) takeWarnings() []string {
	out := n.warnings
	n.warnings = nil
	return out
}

// defaults reads the complete default configuration from Ghostty. This is the
// authoritative key list and default value set for the installed build.
func (n *native) defaults(ctx context.Context) (Values, error) {
	stdout, _, err := n.run(ctx, "+show-config --default", "+show-config", "--default")
	if err != nil {
		return nil, err
	}
	return valuesFromConfig(stdout), nil
}

// changes reads the effective configuration that differs from the defaults.
// Ghostty 1.3.1 accepts no --config-file argument here, so this reflects the
// default configuration locations only.
func (n *native) changes(ctx context.Context) (Values, error) {
	stdout, _, err := n.run(ctx, "+show-config --changes-only", "+show-config", "--changes-only")
	if err != nil {
		return nil, err
	}
	return valuesFromConfig(stdout), nil
}

// docs reads the default configuration with documentation comments.
func (n *native) docs(ctx context.Context) ([]byte, error) {
	stdout, _, err := n.run(ctx, "+show-config --default --docs", "+show-config", "--default", "--docs")
	if err != nil {
		return nil, err
	}
	return stdout, nil
}

// validate runs Ghostty's native validator over one configuration file. The
// validator follows config-file directives, so the caller passes a wrapper that
// reproduces the final include graph. Diagnostics are returned as
// "path:line:key: message" strings.
func (n *native) validate(ctx context.Context, path string) ([]string, error) {
	if !n.available() {
		return nil, errNoGhostty
	}
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	// NOTE: Ghostty 1.3.1 actions accept only --config-file here.
	// --config-default-files is a GUI/general flag and makes the action fail.
	stdout, stderr, err := n.runner.Run(ctx, Command{
		Path: n.bin,
		Args: []string{"+validate-config", "--config-file=" + path},
	})
	_ = stderr
	out := strings.TrimRight(string(stdout), "\n")
	if err == nil {
		return nil, nil
	}
	var diagnostics []string
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			diagnostics = append(diagnostics, line)
		}
	}
	if len(diagnostics) == 0 {
		return nil, &nativeError{probe: "+validate-config", code: exitCode(err), stdout: string(stdout), err: err}
	}
	return diagnostics, nil
}

// valuesFromConfig parses show-config output. Repeated keys keep their order,
// which is what makes font-family and palette lists meaningful.
func valuesFromConfig(content []byte) Values {
	out := make(Values)
	for _, e := range parseEntries(content) {
		out[e.key] = append(out[e.key], e.value)
	}
	return out
}

// fontFamily is one family reported by `ghostty +list-fonts`.
type fontFamily struct {
	Name   string
	Styles []string
}

// listFonts parses `ghostty +list-fonts`. Family names are unindented, styles
// are indented beneath them and blank lines separate families.
func (n *native) listFonts(ctx context.Context) ([]fontFamily, error) {
	stdout, _, err := n.run(ctx, "+list-fonts", "+list-fonts")
	if err != nil {
		return nil, err
	}
	return parseFontList(string(stdout)), nil
}

func parseFontList(out string) []fontFamily {
	var families []fontFamily
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			if len(families) == 0 {
				continue
			}
			style := strings.TrimSpace(line)
			if style != "" {
				families[len(families)-1].Styles = append(families[len(families)-1].Styles, style)
			}
			continue
		}
		families = append(families, fontFamily{Name: strings.TrimSpace(line)})
	}
	return families
}

// fontProbeConcurrency bounds how many targeted family probes run at once, so
// supplementing the font catalog stays fast without spawning a process per
// family all at once.
const fontProbeConcurrency = 4

// listFontsFamily runs a targeted `ghostty +list-fonts --family=NAME` probe.
// The installed build answers an unknown family with empty output and a zero
// exit status, so an empty result means "not installed" rather than an error.
// This matters because the unfiltered listing omits some installed families.
func (n *native) listFontsFamily(ctx context.Context, family string) ([]fontFamily, error) {
	if !n.available() {
		return nil, errNoGhostty
	}
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	stdout, _, err := n.runner.Run(ctx, Command{Path: n.bin, Args: []string{"+list-fonts", "--family=" + family}})
	if err != nil {
		return nil, err
	}
	return parseFontList(string(stdout)), nil
}

// probeFamilies probes several family names concurrently and returns the
// families that were found plus the names whose probe failed. It deliberately
// bypasses the warning collector: probes run in parallel.
func (n *native) probeFamilies(ctx context.Context, names []string) ([]fontFamily, []string) {
	if !n.available() || len(names) == 0 {
		return nil, nil
	}
	found := make([][]fontFamily, len(names))
	failed := make([]bool, len(names))
	sem := make(chan struct{}, fontProbeConcurrency)
	var wg sync.WaitGroup
	for i, name := range names {
		wg.Add(1)
		go func(i int, name string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			families, err := n.listFontsFamily(ctx, name)
			if err != nil {
				failed[i] = true
				return
			}
			found[i] = families
		}(i, name)
	}
	wg.Wait()
	var out []fontFamily
	var failures []string
	for i, name := range names {
		if failed[i] {
			failures = append(failures, name)
			continue
		}
		out = append(out, found[i]...)
	}
	return out, failures
}

// themeListing is the result of `ghostty +list-themes`.
type themeListing struct {
	Names []string
	// Degraded is set when the command failed part way through, which is the
	// observed behaviour of Ghostty 1.3.1 on macOS: exit status 1 with a
	// truncated list. Callers must fall back to the filesystem inventory.
	Degraded bool
	Err      error
}

// listThemes collects the native theme listing without depending on it.
func (n *native) listThemes(ctx context.Context) themeListing {
	stdout, _, err := n.run(ctx, "+list-themes", "+list-themes", "--plain")
	names := parseThemeList(string(stdout))
	if err != nil {
		return themeListing{Names: names, Degraded: true, Err: err}
	}
	return themeListing{Names: names}
}

// parseThemeList strips the " (resources)" style source suffix Ghostty appends.
func parseThemeList(out string) []string {
	var names []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if i := strings.LastIndex(line, " ("); i > 0 && strings.HasSuffix(line, ")") {
			line = strings.TrimSpace(line[:i])
		}
		names = append(names, line)
	}
	return names
}

// showFace resolves codepoints to the installed face Ghostty would use for the
// current effective configuration.
func (n *native) showFace(ctx context.Context, text string) (string, error) {
	stdout, _, err := n.run(ctx, "+show-face", "+show-face", "--string="+text)
	if err != nil {
		return "", err
	}
	return string(stdout), nil
}

// discoverGhostty resolves the executable. An explicit path must exist;
// otherwise discovery is best effort and an empty result simply means the
// native features are unavailable.
func discoverGhostty(p paths) (string, error) {
	if p.ghosttyBin != "" {
		if err := requireExecutable(p.ghosttyBin); err != nil {
			return "", err
		}
		return p.ghosttyBin, nil
	}
	if p.lookPath != nil {
		if found, err := p.lookPath("ghostty"); err == nil && found != "" {
			return found, nil
		}
	}
	if err := requireExecutable(bundleBinary); err == nil {
		return bundleBinary, nil
	}
	return "", nil
}

// discoverBrew resolves the Homebrew executable used for font installs.
func discoverBrew(p paths) string {
	candidates := []string{"/opt/homebrew/bin/brew", "/usr/local/bin/brew"}
	for _, candidate := range candidates {
		if err := requireExecutable(candidate); err == nil {
			return candidate
		}
	}
	if p.lookPath != nil {
		if found, err := p.lookPath("brew"); err == nil && found != "" {
			return found
		}
	}
	return ""
}

// discoverAssetDir resolves the bundled asset directory.
func discoverAssetDir(p paths, exePath string) string {
	if p.assetDir != "" {
		return p.assetDir
	}
	var candidates []string
	if exePath != "" {
		dir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(dir, "..", "share", "ghostty-studio", "assets"),
			filepath.Join(dir, "..", "assets"),
			filepath.Join(dir, "assets"),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "assets"))
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			if abs, err := filepath.Abs(candidate); err == nil {
				return abs
			}
			return candidate
		}
	}
	return ""
}

// openBinary is the macOS launcher used for native previews.
const openBinary = "/usr/bin/open"

// sortedNames returns sorted unique names.
func sortedNames(names []string) []string {
	seen := make(map[string]bool, len(names))
	var out []string
	for _, name := range names {
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// formatBytes renders a byte count for diagnostics.
func formatBytes(n int64) string {
	switch {
	case n < 1024:
		return strconv.FormatInt(n, 10) + " B"
	case n < 1024*1024:
		return strconv.FormatFloat(float64(n)/1024, 'f', 1, 64) + " KiB"
	default:
		return strconv.FormatFloat(float64(n)/(1024*1024), 'f', 1, 64) + " MiB"
	}
}
