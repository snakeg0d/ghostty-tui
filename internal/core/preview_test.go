package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// previewFixture builds a nested include graph. Every level sets values the
// specimen has to beat: the root sets a command and a window policy, the child
// and the grandchild each set opacity and a font family.
func previewFixture(t *testing.T) (*testFixture, map[string]string) {
	t.Helper()
	fx := newFixture(t)
	dir := filepath.Dir(fx.config)
	child := filepath.Join(dir, "child.conf")
	grand := filepath.Join(dir, "nested", "grand.conf")
	if err := os.MkdirAll(filepath.Dir(grand), 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	fx.writeConfig(strings.Join([]string{
		"# original root",
		"background-opacity = 0.5",
		"font-family = Menlo",
		"font-family = Iosevka",
		"font-feature = -calt",
		"command = /bin/zsh",
		"window-save-state = always",
		"config-file = child.conf",
	}, "\n") + "\n")
	if err := os.WriteFile(child, []byte("# original child\nbackground-opacity = 0.8\nfont-family = Courier New\nconfig-file = nested/grand.conf\n"), 0o644); err != nil {
		t.Fatalf("write child: %v", err)
	}
	if err := os.WriteFile(grand, []byte("# original grandchild\nbackground-opacity = 0.9\nfont-family = Grand Font\nfont-feature = +liga\n"), 0o644); err != nil {
		t.Fatalf("write grandchild: %v", err)
	}
	return fx, map[string]string{"child": child, "grand": grand}
}

// snapshotFiles reads the given files for byte comparison.
func snapshotFiles(t *testing.T, paths ...string) map[string]string {
	t.Helper()
	out := make(map[string]string, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		out[path] = string(data)
	}
	return out
}

// assertFilesUnchanged verifies that a snapshot is untouched.
func assertFilesUnchanged(t *testing.T, snapshot map[string]string) {
	t.Helper()
	for path, want := range snapshot {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if string(data) != want {
			t.Errorf("%s was modified by the preview:\ngot  %q\nwant %q", path, data, want)
		}
	}
}

// harnessPathFromOpen finds the --config-file argument the preview launched.
func harnessPathFromOpen(t *testing.T, fx *testFixture) string {
	t.Helper()
	var harness string
	for _, call := range fx.runner.commands() {
		if filepath.Base(call.Path) != "open" {
			continue
		}
		for _, arg := range call.Args {
			if strings.HasPrefix(arg, "--config-file=") {
				harness = strings.TrimPrefix(arg, "--config-file=")
			}
		}
	}
	if harness == "" {
		t.Fatal("no --config-file argument was passed to the launcher")
	}
	return harness
}

// TestPreviewHarnessResolvesToTheStagedSpecimen is the regression for the
// preview bug: including every resolved graph file made Ghostty load nested
// includes twice and let the original command, opacity and fonts win, because a
// file's includes are applied after its own values.
func TestPreviewHarnessResolvesToTheStagedSpecimen(t *testing.T) {
	fx, files := previewFixture(t)
	e := fx.openEngine()
	original := snapshotFiles(t, fx.config, files["child"], files["grand"])

	// A spaced installation path exercises the quoting path.
	e.exePath = "/Applications/My Tools/ghostty-tui"

	result, err := e.Preview(Change{
		Revision: e.State().Revision,
		Label:    "Preview",
		Values: Values{
			"background-opacity": {"0.95"},
			"font-family":        {"", "Maple Mono"},
		},
	})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if fx.runner.openCommandCount() != 1 {
		t.Fatalf("expected exactly one launch, got %d", fx.runner.openCommandCount())
	}
	if !strings.Contains(result.Message, "specimen") || !strings.Contains(result.Message, "not modified") {
		t.Errorf("the result must describe the specimen and its isolation: %q", result.Message)
	}

	// The originals must be untouched: the preview only reads them.
	assertFilesUnchanged(t, original)

	harness := harnessPathFromOpen(t, fx)
	if !strings.HasPrefix(harness, filepath.Join(fx.state, "preview")) {
		t.Errorf("the harness must live in the state preview directory, got %s", harness)
	}
	staged := filepath.Join(filepath.Dir(harness), stagedFileName)
	if _, err := os.Stat(staged); err != nil {
		t.Fatalf("the staged include must be kept for inspection: %v", err)
	}

	// Resolve the generated harness exactly as Ghostty does.
	graph, err := resolveGraph([]string{harness})
	if err != nil {
		t.Fatalf("resolving the harness: %v", err)
	}
	order := graph.paths()
	if len(order) != 5 {
		t.Fatalf("harness graph = %v, want harness, root, child, grandchild, staged", order)
	}
	if !samePath(order[0], harness) || !samePath(order[len(order)-1], staged) {
		t.Errorf("the staged file must load last: %v", order)
	}
	loads := map[string]int{}
	for _, path := range order {
		loads[canonicalKey(path)]++
	}
	for path, n := range loads {
		if n != 1 {
			t.Errorf("%s is loaded %d times; nested includes must not be doubled", path, n)
		}
	}

	effective := evaluate(e.defaults, graph.files, themeIndex(e.themes))
	if got := effective.Last("background-opacity"); got != "0.95" {
		t.Errorf("staged opacity must beat every original level, got %q", got)
	}
	// The staged font choice resets the accumulated chain, so the specimen's own
	// family is the only one left: nothing from the user's configuration leaks
	// into the specimen and nothing is duplicated.
	if got := effective["font-family"]; !equalStrings(got, []string{"Maple Mono"}) {
		t.Errorf("font chain = %v, want the staged choice alone", got)
	}
	// A repeated key the specimen does not stage shows each level applied
	// exactly once, which is what a doubly loaded nested include would break.
	if got := effective["font-feature"]; !equalStrings(got, []string{"-calt", "+liga"}) {
		t.Errorf("repeated originals must each appear once, got %v", got)
	}
	if got := effective.Last("command"); got != `shell:'/Applications/My Tools/ghostty-tui' specimen` {
		t.Errorf("the specimen command must win over the user's command, got %q", got)
	}
	if got := effective.Last("window-save-state"); got != "never" {
		t.Errorf("the disposable-window override must win, got %q", got)
	}
	// The original configuration keeps its own font chain: the reset exists only
	// inside the specimen.
	if got := e.State().Current["font-family"]; !equalStrings(got, []string{"Menlo", "Iosevka", "Courier New", "Grand Font"}) {
		t.Errorf("the user's own font chain must be untouched, got %v", got)
	}
	// The harness must not disturb the studio's own managed include.
	if strings.Contains(string(mustRead(t, harness)), managedIncludeLine(managedPath(fx.config))) {
		t.Error("the harness must not point at the managed include")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(fx.config), managedFileName)); !os.IsNotExist(err) {
		t.Error("previewing must not create the managed include")
	}
}

// TestPreviewIncludesConfiguredRootsOnly proves the harness follows the real
// load order: the roots are included individually, so an include is never
// applied twice, and a second root is still honoured.
func TestPreviewIncludesConfiguredRootsOnly(t *testing.T) {
	fx := newFixture(t)
	nativeDir := filepath.Join(fx.home, "Library", "Application Support", "com.mitchellh.ghostty")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	child := filepath.Join(filepath.Dir(fx.config), "child.conf")
	if err := os.WriteFile(child, []byte("font-size = 20\n"), 0o644); err != nil {
		t.Fatalf("write child: %v", err)
	}
	fx.writeConfig("font-size = 11\nconfig-file = child.conf\n")
	second := filepath.Join(nativeDir, "config")
	if err := os.WriteFile(second, []byte("background-opacity = 0.6\n"), 0o644); err != nil {
		t.Fatalf("write second root: %v", err)
	}
	e, err := fx.openEngineWithEnv(Options{StateDir: fx.state, GhosttyPath: fx.bin}, nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if len(e.State().Sources) != 3 {
		t.Fatalf("expected three sources, got %v", e.State().Sources)
	}
	original := snapshotFiles(t, fx.config, second, child)

	if _, err := e.Preview(Change{Revision: e.State().Revision, Label: "Preview", Values: Values{"font-size": {"17"}}}); err != nil {
		t.Fatalf("preview: %v", err)
	}
	assertFilesUnchanged(t, original)

	harness := harnessPathFromOpen(t, fx)
	graph, err := resolveGraph([]string{harness})
	if err != nil {
		t.Fatalf("resolving the harness: %v", err)
	}
	// harness, root one, its child, root two, staged
	if len(graph.files) != 5 {
		t.Fatalf("harness graph = %v, want five files", graph.paths())
	}
	loads := map[string]int{}
	for _, path := range graph.paths() {
		loads[canonicalKey(path)]++
	}
	for path, n := range loads {
		if n != 1 {
			t.Errorf("%s loaded %d times", path, n)
		}
	}
	effective := evaluate(e.defaults, graph.files, themeIndex(e.themes))
	if got := effective.Last("font-size"); got != "17" {
		t.Errorf("staged font size must win, got %q", got)
	}
	if got := effective.Last("background-opacity"); got != "0.6" {
		t.Errorf("the second root must still contribute, got %q", got)
	}
}

// TestPreviewRefusesToLaunchAHarnessGhosttyRejects guards the window: a rejected
// harness would leave the specimen running the user's real configuration.
func TestPreviewRefusesToLaunchAHarnessGhosttyRejects(t *testing.T) {
	fx, _ := previewFixture(t)
	e := fx.openEngine()
	var validated []string
	fx.runner.validate = func(path string) ([]string, error) {
		validated = append(validated, path)
		return []string{path + ":3:command: invalid value"}, nil
	}
	e.exePath = "/usr/local/bin/ghostty-tui"

	_, err := e.Preview(Change{Revision: e.State().Revision, Label: "Preview", Values: Values{"background-opacity": {"0.95"}}})
	if err == nil {
		t.Fatal("a harness Ghostty rejects must not be launched")
	}
	if !strings.Contains(err.Error(), "no window was opened") {
		t.Errorf("the refusal must say that nothing was opened: %v", err)
	}
	if fx.runner.openCommandCount() != 0 {
		t.Error("no launcher process may run for a rejected harness")
	}
	// The harness must have been validated before any launch was attempted, and
	// kept on disk so the user can inspect it.
	if len(validated) != 1 {
		t.Fatalf("expected exactly one harness validation, got %v", validated)
	}
	if _, statErr := os.Stat(validated[0]); statErr != nil {
		t.Errorf("the rejected harness must be kept for inspection: %v", statErr)
	}
	if !strings.HasPrefix(validated[0], filepath.Join(fx.state, "preview")) {
		t.Errorf("the validated file must be the preview harness, got %s", validated[0])
	}
}

func TestCommandDirectiveQuotesUnsafePathsWithoutShellExpansion(t *testing.T) {
	cases := []struct{ path, want string }{
		{"/Users/daniel/.local/bin/ghostty-tui", "direct:/Users/daniel/.local/bin/ghostty-tui"},
		{"/opt/homebrew/bin/ghostty-tui", "direct:/opt/homebrew/bin/ghostty-tui"},
		{"/Applications/My Tools/ghostty-tui", "shell:'/Applications/My Tools/ghostty-tui'"},
		{"/tmp/it's here/ghostty-tui", `shell:'/tmp/it'\''s here/ghostty-tui'`},
		{"/tmp/$(rm -rf x)/ghostty-tui", "shell:'/tmp/$(rm -rf x)/ghostty-tui'"},
		{"/tmp/`id`/ghostty-tui", "shell:'/tmp/`id`/ghostty-tui'"},
	}
	for _, c := range cases {
		got := commandDirective(c.path)
		if got != c.want {
			t.Errorf("commandDirective(%q) = %q, want %q", c.path, got, c.want)
		}
		if strings.ContainsAny(got, "\n\r") {
			t.Errorf("a command directive must stay on one line: %q", got)
		}
		if strings.HasPrefix(got, "direct:") && strings.ContainsAny(got, shellMetacharacters) {
			t.Errorf("direct mode must only be used for paths that need no quoting: %q", got)
		}
	}
	fx := newFixture(t)
	e := fx.openEngine()
	e.exePath = "/Applications/My Tools/ghostty-tui"
	rendered := string(e.previewStagedConfig(Values{"background-opacity": {"0.95"}}))
	if !strings.Contains(rendered, "command = shell:'/Applications/My Tools/ghostty-tui' specimen\n") {
		t.Errorf("staged config must render the specimen command:\n%s", rendered)
	}
	if !strings.Contains(rendered, "window-save-state = never\n") {
		t.Errorf("staged config must disable window state:\n%s", rendered)
	}
}

// TestPreviewCommandFormsAreAcceptedByRealGhostty validates the generated
// specimen command forms with the installed build. It only reads the binary and
// writes to a temporary directory.
func TestPreviewCommandFormsAreAcceptedByRealGhostty(t *testing.T) {
	bin := "/Applications/Ghostty.app/Contents/MacOS/ghostty"
	if _, err := os.Stat(bin); err != nil {
		t.Skip("no installed ghostty to validate against")
	}
	n := &native{bin: bin, runner: osRunner{}}
	for _, exePath := range []string{"/usr/local/bin/ghostty-tui", "/Applications/My Tools/ghostty-tui", "/tmp/it's here/ghostty-tui"} {
		dir := t.TempDir()
		harness := filepath.Join(dir, "specimen.ghostty")
		content := "# harness\n" +
			"config-file = ?" + filepath.Join(dir, "staged.ghostty") + "\n"
		staged := "# staged\n" +
			"background-opacity = 0.95\n" +
			"palette = 0=#101010\n" +
			"window-save-state = never\n" +
			"command = " + commandDirective(exePath) + " specimen\n"
		if err := os.WriteFile(filepath.Join(dir, "staged.ghostty"), []byte(staged), 0o600); err != nil {
			t.Fatalf("write staged: %v", err)
		}
		if err := os.WriteFile(harness, []byte(content), 0o600); err != nil {
			t.Fatalf("write harness: %v", err)
		}
		diagnostics, err := n.validate(context.Background(), harness)
		if err != nil {
			t.Fatalf("ghostty could not validate %q: %v", exePath, err)
		}
		if len(diagnostics) > 0 {
			t.Errorf("ghostty rejected the harness for %q: %v", exePath, diagnostics)
		}
	}
}

// mustRead reads a file or fails the test.
func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}
