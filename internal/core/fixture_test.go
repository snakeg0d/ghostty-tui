package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakeGhostty models the parts of the Ghostty CLI the engine depends on. It is
// deterministic, never touches the network and never launches a window.
type fakeGhostty struct {
	version  string
	defaults map[string][]string
	// options maps a key to the values its documentation enumerates.
	options map[string][]string
	// restartKeys are documented as requiring a full restart.
	restartKeys map[string]bool
	fonts       string
	face        string
	// themeList is the native theme listing; it fails like the real build does.
	themeList     string
	themeListFail bool
	validateCalls int
	// changesOnly overrides the `--changes-only` output; when empty the default
	// dump is returned, which means "nothing differs from the defaults".
	changesOnly string
	// familyProbes answers targeted `+list-fonts --family=NAME` probes. A name
	// that is absent answers with empty output, exactly like the installed
	// build does for an unknown family.
	familyProbes map[string]string
	// brewInstalls maps a cask token to the family a real install would make
	// visible, so tests can exercise the post-install verification.
	brewInstalls map[string]string
}

// defaultsText renders `ghostty +show-config --default`.
func (f *fakeGhostty) defaultsText() string {
	keys := make([]string, 0, len(f.defaults))
	for key := range f.defaults {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, key := range keys {
		for _, value := range f.defaults[key] {
			fmt.Fprintf(&b, "%s = %s\n", key, value)
		}
	}
	return b.String()
}

// docsText renders `ghostty +show-config --default --docs`.
func (f *fakeGhostty) docsText() string {
	keys := make([]string, 0, len(f.defaults))
	for key := range f.defaults {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, key := range keys {
		fmt.Fprintf(&b, "# Documentation for %s.\n", key)
		if len(f.options[key]) > 0 {
			b.WriteString("# \n# Valid values are:\n# \n")
			for _, option := range f.options[key] {
				fmt.Fprintf(&b, "#   * `%s`\n", option)
			}
		}
		if f.restartKeys[key] {
			b.WriteString("# \n# Warning: This setting cannot be reloaded at runtime. To change the\n# setting you must fully restart Ghostty.\n")
		}
		values := f.defaults[key]
		if len(values) == 0 {
			values = []string{""}
		}
		for _, value := range values {
			fmt.Fprintf(&b, "%s = %s\n", key, value)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// defaultValidate is the fake validator: unknown keys and the INVALID marker are
// reported the way the real validator reports them.
func (f *fakeGhostty) defaultValidate(path string, known map[string]bool) ([]string, error) {
	var diagnostics []string
	seen := map[string]bool{}
	var walk func(string) error
	walk = func(file string) error {
		canonical := canonicalKey(file)
		if seen[canonical] {
			return nil
		}
		seen[canonical] = true
		content, err := os.ReadFile(file)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		for _, entry := range parseEntries(content) {
			if entry.key == "config-file" {
				ref := strings.TrimPrefix(entry.value, "?")
				if !filepath.IsAbs(ref) {
					ref = filepath.Join(filepath.Dir(file), ref)
				}
				if err := walk(ref); err != nil {
					return err
				}
				continue
			}
			if !known[entry.key] {
				diagnostics = append(diagnostics, fmt.Sprintf("%s:%d:%s: unknown field", file, entry.line, entry.key))
				continue
			}
			if strings.Contains(entry.value, "INVALID") {
				diagnostics = append(diagnostics, fmt.Sprintf("%s:%d:%s: invalid value %q", file, entry.line, entry.key, entry.value))
				continue
			}
			if reason := f.valueTypeError(entry.key, entry.value); reason != "" {
				diagnostics = append(diagnostics, fmt.Sprintf("%s:%d:%s: invalid value %q (%s)", file, entry.line, entry.key, entry.value, reason))
			}
		}
		return nil
	}
	if err := walk(path); err != nil {
		return nil, err
	}
	return diagnostics, nil
}

// valueTypeError mirrors the real validator's type checking for the boolean and
// numeric keys this fixture defines.
func (f *fakeGhostty) valueTypeError(key, value string) string {
	if value == "" {
		return ""
	}
	var scalar string
	for _, candidate := range f.defaults[key] {
		if candidate != "" {
			scalar = candidate
			break
		}
	}
	if scalar == "" {
		return ""
	}
	switch scalar {
	case "true", "false":
		if value != "true" && value != "false" {
			return "expected a boolean"
		}
		return ""
	}
	if _, err := strconv.ParseFloat(scalar, 64); err != nil {
		return ""
	}
	if _, err := strconv.ParseFloat(value, 64); err != nil {
		return "expected a number"
	}
	return ""
}

// fakeRunner is the injectable Runner used by every test.
type fakeRunner struct {
	mu      sync.Mutex
	calls   []Command
	ghostty *fakeGhostty
	known   map[string]bool
	// validate overrides the built-in fake validator when set.
	validate func(path string) ([]string, error)

	brewCalls int
	brewOut   string
	brewErr   error

	openCalls int
	openErr   error
}

func (r *fakeRunner) Run(_ context.Context, cmd Command) ([]byte, []byte, error) {
	// The engine probes font families concurrently, so the fake serialises its
	// whole dispatch: the recorded calls and the simulated font database stay
	// consistent without the caller needing to synchronise.
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, cmd)

	switch filepath.Base(cmd.Path) {
	case "ghostty", "ghostty-fake":
		// A targeted family probe is answered before the plain listing, because
		// both invocations start with "+list-fonts".
		for _, arg := range cmd.Args {
			if strings.HasPrefix(arg, "--family=") {
				return []byte(r.ghostty.familyProbes[strings.TrimPrefix(arg, "--family=")]), nil, nil
			}
		}
		for _, arg := range cmd.Args {
			switch {
			case arg == "--version":
				return []byte("Ghostty " + r.ghostty.version + "\n\nVersion\n  - version: " + r.ghostty.version + "\n"), nil, nil
			case arg == "+list-fonts":
				return []byte(r.ghostty.fonts), nil, nil
			case arg == "+list-themes":
				if r.ghostty.themeListFail {
					return []byte(r.ghostty.themeList), nil, &execExitError{code: 1}
				}
				return []byte(r.ghostty.themeList), nil, nil
			case strings.HasPrefix(arg, "+show-face"):
				return []byte(r.ghostty.face), nil, nil
			case arg == "+show-config":
				continue
			case strings.HasPrefix(arg, "+validate-config"):
				path := strings.TrimPrefix(arg, "+validate-config")
				if arg == "+validate-config" {
					for _, candidate := range cmd.Args {
						if strings.HasPrefix(candidate, "--config-file=") {
							path = strings.TrimPrefix(candidate, "--config-file=")
						}
					}
				} else {
					path = strings.TrimPrefix(path, "=")
				}
				r.ghostty.validateCalls++
				validate := r.validate
				if validate == nil {
					validate = func(p string) ([]string, error) { return r.ghostty.defaultValidate(p, r.known) }
				}
				diagnostics, err := validate(path)
				if err != nil {
					return nil, nil, err
				}
				if len(diagnostics) > 0 {
					return []byte(strings.Join(diagnostics, "\n") + "\n"), nil, &execExitError{code: 1}
				}
				return nil, nil, nil
			}
		}
		// show-config requests are distinguished by their flags.
		args := strings.Join(cmd.Args, " ")
		switch {
		case strings.Contains(args, "--docs"):
			return []byte(r.ghostty.docsText()), nil, nil
		case strings.Contains(args, "--default"):
			return []byte(r.ghostty.defaultsText()), nil, nil
		case strings.Contains(args, "--changes-only"):
			if r.ghostty.changesOnly != "" {
				return []byte(r.ghostty.changesOnly), nil, nil
			}
			return []byte(r.ghostty.defaultsText()), nil, nil
		}
		return nil, nil, fmt.Errorf("fake ghostty: unsupported invocation %v", cmd.Args)
	case "brew":
		r.brewCalls++
		if r.brewErr == nil {
			for _, arg := range cmd.Args {
				if family, ok := r.ghostty.brewInstalls[arg]; ok {
					if r.ghostty.familyProbes == nil {
						r.ghostty.familyProbes = map[string]string{}
					}
					r.ghostty.familyProbes[family] = family + "\n  " + family + " Regular\n"
				}
			}
		}
		return []byte(r.brewOut), nil, r.brewErr
	case "open":
		r.openCalls++
		return nil, nil, r.openErr
	}
	return nil, nil, fmt.Errorf("fake runner: unexpected command %s", cmd.Path)
}

func (r *fakeRunner) commands() []Command {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]Command(nil), r.calls...)
}

// brewCommandCount reports how often Homebrew was invoked.
func (r *fakeRunner) brewCommandCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.brewCalls
}

func (r *fakeRunner) openCommandCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.openCalls
}

// execExitError stands in for exec.ExitError.
type execExitError struct{ code int }

func (e *execExitError) Error() string { return fmt.Sprintf("exit status %d", e.code) }

// ExitCode exposes the code the way exec.ExitError does.
func (e *execExitError) ExitCode() int { return e.code }

// testFixture assembles a temporary machine: a fake Ghostty bundle, a state
// directory and an optional configuration file.
type testFixture struct {
	t         *testing.T
	root      string
	home      string
	bundle    string
	bin       string
	resources string
	config    string
	state     string
	runner    *fakeRunner
	engine    *Engine
}

// fakeDefaults covers the curated schema plus the keys the tests stage.
func fakeDefaults() map[string][]string {
	defaults := make(map[string][]string, len(builtinDefaults))
	for key, value := range builtinDefaults {
		defaults[key] = []string{value}
	}
	// Repeated keys have empty defaults in real Ghostty output.
	for _, key := range []string{"font-family", "font-feature", "font-variation", "palette", "custom-shader", "theme"} {
		defaults[key] = []string{""}
	}
	defaults["working-directory"] = []string{""}
	defaults["command"] = []string{""}
	defaults["title"] = []string{""}
	defaults["window-show-tab-bar"] = []string{"auto"}
	defaults["language"] = []string{""}
	return defaults
}

// newFixture creates the temporary machine.
func newFixture(t *testing.T) *testFixture {
	t.Helper()
	root := t.TempDir()
	fx := &testFixture{t: t, root: root}

	fx.home = filepath.Join(root, "home")
	fx.bundle = filepath.Join(root, "Ghostty.app")
	fx.bin = filepath.Join(fx.bundle, "Contents", "MacOS", "ghostty")
	fx.resources = filepath.Join(fx.bundle, "Contents", "Resources", "ghostty")
	for _, dir := range []string{
		filepath.Dir(fx.bin),
		filepath.Join(fx.resources, "themes"),
		filepath.Join(fx.resources, "doc"),
		filepath.Join(fx.home, ".config", "ghostty"),
		filepath.Join(fx.home, ".config", "ghostty", "themes"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(fx.bin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fx.resources, "doc", "ghostty.5.md"), []byte("# ghostty\n"), 0o644); err != nil {
		t.Fatalf("write doc: %v", err)
	}
	fx.config = filepath.Join(fx.home, ".config", "ghostty", "config")
	fx.state = filepath.Join(root, "state")

	fx.runner = &fakeRunner{
		ghostty: &fakeGhostty{
			version:     "1.3.1",
			defaults:    fakeDefaults(),
			options:     map[string][]string{},
			restartKeys: map[string]bool{},
			fonts:       "Iosevka\n  Iosevka\n\nJetBrains Mono\n  JetBrains Mono\n  JetBrains Mono Bold\n\nMenlo\n  Menlo\n",
			themeList:   "Nord (resources)\nCatppuccin Mocha (resources)\n",
		},
	}
	fx.runner.known = knownKeys(fx.runner.ghostty.defaults)
	return fx
}

// writeBundleTheme writes a theme file into the fake application bundle.
func (fx *testFixture) writeBundleTheme(name, content string) string {
	fx.t.Helper()
	dir := filepath.Join(fx.resources, "themes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fx.t.Fatalf("mkdir bundle themes: %v", err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fx.t.Fatalf("write bundle theme: %v", err)
	}
	return path
}

// knownKeys builds the validator's key set.
func knownKeys(defaults map[string][]string) map[string]bool {
	out := make(map[string]bool, len(defaults))
	for key := range defaults {
		out[key] = true
	}
	out["config-file"] = true
	return out
}

// writeConfig writes the fixture configuration file.
func (fx *testFixture) writeConfig(content string) {
	fx.t.Helper()
	if err := os.MkdirAll(filepath.Dir(fx.config), 0o755); err != nil {
		fx.t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(fx.config, []byte(content), 0o644); err != nil {
		fx.t.Fatalf("write config: %v", err)
	}
}

// readConfig returns the fixture configuration file.
func (fx *testFixture) readConfig() string {
	fx.t.Helper()
	data, err := os.ReadFile(fx.config)
	if err != nil {
		fx.t.Fatalf("read config: %v", err)
	}
	return string(data)
}

// openEngine starts an engine against the fixture with an explicit config path.
func (fx *testFixture) openEngine() *Engine {
	fx.t.Helper()
	env := map[string]string{}
	e, err := open(
		Options{ConfigPath: fx.config, StateDir: fx.state, GhosttyPath: fx.bin},
		fx.runner,
		func(key string) string { return env[key] },
		fx.home,
		func(string) (string, error) { return fx.bin, nil },
	)
	if err != nil {
		fx.t.Fatalf("open engine: %v", err)
	}
	fx.engine = e
	return e
}

// openEngineWithEnv starts an engine with environment overrides, used for the
// discovery tests.
func (fx *testFixture) openEngineWithEnv(opts Options, env map[string]string) (*Engine, error) {
	fx.t.Helper()
	return open(
		opts,
		fx.runner,
		func(key string) string { return env[key] },
		fx.home,
		func(string) (string, error) { return fx.bin, nil },
	)
}

// writeTheme writes a theme file into the user theme directory.
func (fx *testFixture) writeTheme(name, background, foreground string) {
	fx.t.Helper()
	dir := filepath.Join(fx.home, ".config", "ghostty", "themes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fx.t.Fatalf("mkdir themes: %v", err)
	}
	content := "background = " + background + "\nforeground = " + foreground + "\n" +
		"palette = 0=#101010\npalette = 1=#ff0000\npalette = 15=#ffffff\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		fx.t.Fatalf("write theme: %v", err)
	}
}

// writeShader writes a shader asset under the user shader directory.
func (fx *testFixture) writeShader(rel string) string {
	fx.t.Helper()
	path := filepath.Join(fx.home, ".config", "ghostty", "shaders", rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fx.t.Fatalf("mkdir shaders: %v", err)
	}
	if err := os.WriteFile(path, []byte("void mainImage(out vec4 c, in vec2 p) { c = vec4(0.0); }\n"), 0o644); err != nil {
		fx.t.Fatalf("write shader: %v", err)
	}
	return path
}
