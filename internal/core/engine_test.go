package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenRefreshAndState(t *testing.T) {
	fx := newFixture(t)
	fx.runner.ghostty.themeListFail = true
	fx.writeConfig("# Keep this comment.\nfont-family = Menlo\nfont-family = Iosevka\nwindow-padding-x = 5\n")
	e := fx.openEngine()

	state := e.State()
	if state.Version != "1.3.1" {
		t.Errorf("version = %q, want 1.3.1", state.Version)
	}
	if state.GhosttyPath != fx.bin {
		t.Errorf("ghostty path = %q, want %q", state.GhosttyPath, fx.bin)
	}
	if state.ConfigPath != fx.config {
		t.Errorf("config path = %q, want %q", state.ConfigPath, fx.config)
	}
	if state.Revision == "" {
		t.Error("revision must not be empty")
	}
	list := state.Current["font-family"]
	if len(list) != 2 || list[0] != "Menlo" || list[1] != "Iosevka" {
		t.Errorf("repeated font-family values = %v, want [Menlo Iosevka]", list)
	}
	if got := state.Current.Last("window-padding-x"); got != "5" {
		t.Errorf("window-padding-x = %q, want 5", got)
	}
	var degraded bool
	for _, warning := range state.Warnings {
		if strings.Contains(warning, "+list-themes") {
			degraded = true
		}
	}
	if !degraded {
		t.Errorf("the degraded native theme listing must be reported: %v", state.Warnings)
	}

	// Refresh must be idempotent and must not create state.
	if err := e.Refresh(); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if _, err := os.Stat(fx.state); !os.IsNotExist(err) {
		t.Errorf("startup must not create the state directory, stat error = %v", err)
	}
	if again := e.State(); again.Revision != state.Revision {
		t.Error("revision must not change without a configuration change")
	}
}

func TestApplyWritesManagedIncludeAndReportsHonestly(t *testing.T) {
	fx := newFixture(t)
	original := "# Keep this exact comment.\nfont-family = Menlo\nfont-family = Iosevka\nbackground-opacity = 0.8\n"
	fx.writeConfig(original)
	e := fx.openEngine()

	result, err := e.Apply(Change{Revision: e.State().Revision, Label: "Opacity", Values: Values{"background-opacity": {"0.95"}}})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !strings.Contains(result.Message, "background-opacity") && !strings.Contains(result.Message, "1 setting") {
		t.Errorf("result message must describe what was saved: %q", result.Message)
	}
	if !strings.Contains(result.Message, "Reload") {
		t.Errorf("result message must state that a reload is needed: %q", result.Message)
	}
	if result.BackupPath == "" {
		t.Error("a persistent backup path must be reported")
	}

	content := fx.readConfig()
	if !strings.HasPrefix(content, original) {
		t.Errorf("the original configuration must stay byte-identical at the front:\n%s", content)
	}
	if !strings.Contains(content, managedIncludeLine(managedPath(fx.config))) {
		t.Errorf("the managed include directive is missing:\n%s", content)
	}
	managed := filepath.Join(filepath.Dir(fx.config), managedFileName)
	data, err := os.ReadFile(managed)
	if err != nil {
		t.Fatalf("managed file: %v", err)
	}
	if !strings.Contains(string(data), "background-opacity = 0.95") {
		t.Errorf("managed file must carry the exact value:\n%s", data)
	}
	if strings.Contains(string(data), "0.9\n") {
		t.Errorf("0.95 must never be truncated to 0.9:\n%s", data)
	}
	// The value the studio reports must be the exact staged value.
	if got := e.State().Current.Last("background-opacity"); got != "0.95" {
		t.Errorf("effective opacity = %q, want 0.95", got)
	}
	if e.State().Revision == "" {
		t.Error("a revision must be reported after applying")
	}
}

func TestApplyUnsetsManagedValuesAndRemovesInclude(t *testing.T) {
	fx := newFixture(t)
	original := "background-opacity = 0.8\n"
	fx.writeConfig(original)
	e := fx.openEngine()

	if _, err := e.Apply(Change{Revision: e.State().Revision, Label: "Opacity", Values: Values{"background-opacity": {"0.6"}}}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	managed := filepath.Join(filepath.Dir(fx.config), managedFileName)
	if _, err := os.Stat(managed); err != nil {
		t.Fatalf("managed file should exist: %v", err)
	}
	if _, err := e.Apply(Change{Revision: e.State().Revision, Label: "Unset", Values: Values{"background-opacity": nil}}); err != nil {
		t.Fatalf("unset: %v", err)
	}
	if _, err := os.Stat(managed); !os.IsNotExist(err) {
		t.Errorf("the managed file must be removed when nothing is managed, stat = %v", err)
	}
	if got := fx.readConfig(); got != original {
		t.Errorf("the configuration file must return to its original bytes:\ngot  %q\nwant %q", got, original)
	}
	if got := e.State().Current.Last("background-opacity"); got != "0.8" {
		t.Errorf("the user's own value must apply again, got %q", got)
	}
}

func TestApplyRejectsStaleRevisionWithoutWriting(t *testing.T) {
	fx := newFixture(t)
	original := "background-opacity = 0.8\n"
	fx.writeConfig(original)
	e := fx.openEngine()
	stale := e.State().Revision

	if _, err := e.Apply(Change{Revision: stale, Label: "First", Values: Values{"font-size": {"15"}}}); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	before := fx.readConfig()
	_, err := e.Apply(Change{Revision: stale, Label: "Stale", Values: Values{"background-opacity": {"0.5"}}})
	if err == nil {
		t.Fatal("a stale revision must be rejected")
	}
	if !strings.Contains(err.Error(), "staged against revision") {
		t.Errorf("the error must explain the stale revision: %v", err)
	}
	if after := fx.readConfig(); after != before {
		t.Error("a rejected apply must not write anything")
	}
}

func TestApplyRejectsExternalEditWithoutWriting(t *testing.T) {
	fx := newFixture(t)
	fx.writeConfig("background-opacity = 0.8\n")
	e := fx.openEngine()
	revision := e.State().Revision

	// The user edits the file behind the studio's back.
	fx.writeConfig("background-opacity = 0.8\n# external edit\n")
	before := fx.readConfig()
	_, err := e.Apply(Change{Revision: revision, Label: "Stale", Values: Values{"background-opacity": {"0.5"}}})
	if err == nil {
		t.Fatal("an external edit must invalidate the staged revision")
	}
	if after := fx.readConfig(); after != before {
		t.Error("a rejected apply must not write anything")
	}
}

func TestInvalidPatchesWriteNothing(t *testing.T) {
	fx := newFixture(t)
	fx.writeConfig("background-opacity = 0.8\n")
	e := fx.openEngine()
	before := fx.readConfig()

	cases := []struct {
		name   string
		values Values
		want   string
	}{
		{"newline injection", Values{"background-opacity": {"0.5\ncommand = touch /tmp/ghostty-studio-pwned"}}, "line break"},
		{"carriage return", Values{"font-size": {"14\r"}}, "line break"},
		{"unknown key", Values{"totally-unknown-setting": {"1"}}, "unknown Ghostty setting"},
		{"relative shader path", Values{"custom-shader": {"../evil.glsl"}}, "absolute path"},
	}
	for _, c := range cases {
		_, err := e.Apply(Change{Revision: e.State().Revision, Label: c.name, Values: c.values})
		if err == nil {
			t.Fatalf("%s must be rejected", c.name)
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: error %q does not mention %q", c.name, err, c.want)
		}
	}
	if after := fx.readConfig(); after != before {
		t.Errorf("rejected patches must not touch the configuration:\n%s", after)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(fx.config), managedFileName)); !os.IsNotExist(err) {
		t.Error("rejected patches must not create the managed file")
	}
	if _, err := os.Stat("/tmp/ghostty-studio-pwned"); !os.IsNotExist(err) {
		t.Error("nothing must ever execute a staged value")
	}
}

func TestRejectedByNativeValidatorWritesNothing(t *testing.T) {
	fx := newFixture(t)
	original := "background-opacity = 0.8\n"
	fx.writeConfig(original)
	e := fx.openEngine()

	_, err := e.Apply(Change{Revision: e.State().Revision, Label: "Invalid", Values: Values{"background-opacity": {"nonsense"}}})
	if err == nil {
		t.Fatal("the native validator must reject an invalid value")
	}
	if !strings.Contains(err.Error(), "invalid value") {
		t.Errorf("the native diagnostics must be surfaced: %v", err)
	}
	if got := fx.readConfig(); got != original {
		t.Error("a failed validation must leave the configuration untouched")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(fx.config), managedFileName)); !os.IsNotExist(err) {
		t.Error("a failed validation must not create the managed file")
	}
}

func TestUnknownKeyInUserConfigurationFailsValidation(t *testing.T) {
	fx := newFixture(t)
	original := "bogus-setting = 1\n"
	fx.writeConfig(original)
	e := fx.openEngine()

	_, err := e.Apply(Change{Revision: e.State().Revision, Label: "Patch", Values: Values{"font-size": {"15"}}})
	if err == nil {
		t.Fatal("Ghostty rejects unknown fields, so the apply must fail")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Errorf("the diagnostics must be surfaced: %v", err)
	}
	if got := fx.readConfig(); got != original {
		t.Error("nothing may be written when the graph does not validate")
	}
}

func TestApplyWithoutNativeGhosttyRefusesToWrite(t *testing.T) {
	fx := newFixture(t)
	fx.writeConfig("background-opacity = 0.8\n")
	hideInstalledBundle(t)
	e, err := open(
		Options{ConfigPath: fx.config, StateDir: fx.state},
		fx.runner,
		func(string) string { return "" },
		fx.home,
		func(string) (string, error) { return "", os.ErrNotExist },
	)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	state := e.State()
	if state.GhosttyPath != "" {
		t.Fatalf("native ghostty should be unavailable, got %q", state.GhosttyPath)
	}
	var explained bool
	for _, warning := range state.Warnings {
		if strings.Contains(warning, "ghostty executable was not found") {
			explained = true
		}
	}
	if !explained {
		t.Errorf("the missing executable must be reported: %v", state.Warnings)
	}
	if _, err := e.Apply(Change{Revision: state.Revision, Label: "x", Values: Values{"font-size": {"15"}}}); err == nil {
		t.Fatal("writing without native validation must be refused")
	}
	if got := fx.readConfig(); got != "background-opacity = 0.8\n" {
		t.Error("nothing may be written without native validation")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(fx.config), managedFileName)); !os.IsNotExist(err) {
		t.Error("no managed file may be created without native validation")
	}
}

func TestUndoRefusesAfterExternalEdits(t *testing.T) {
	fx := newFixture(t)
	original := "# comment\nfont-family = Menlo\n"
	fx.writeConfig(original)
	e := fx.openEngine()

	if _, err := e.Apply(Change{Revision: e.State().Revision, Label: "Font", Values: Values{"font-size": {"16"}}}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	applied := fx.readConfig()

	// An external edit after the apply must block the undo.
	fx.writeConfig(applied + "# external\n")
	_, err := e.Undo()
	if err == nil {
		t.Fatal("undo must refuse when the graph changed after the apply")
	}
	if !strings.Contains(err.Error(), "graph changed") {
		t.Errorf("the refusal must explain the drift: %v", err)
	}
	if got := fx.readConfig(); got != applied+"# external\n" {
		t.Error("a refused undo must not write anything")
	}
}

func TestUndoRestoresBytesAndManagedFile(t *testing.T) {
	fx := newFixture(t)
	original := "# comment\nfont-family = Menlo\n"
	fx.writeConfig(original)
	e := fx.openEngine()

	if _, err := e.Apply(Change{Revision: e.State().Revision, Label: "Font", Values: Values{"font-size": {"16"}}}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	managed := filepath.Join(filepath.Dir(fx.config), managedFileName)
	if _, err := os.Stat(managed); err != nil {
		t.Fatalf("managed file: %v", err)
	}

	result, err := e.Undo()
	if err != nil {
		t.Fatalf("undo: %v", err)
	}
	if !strings.Contains(result.Message, "Undid") {
		t.Errorf("undo result must describe what happened: %q", result.Message)
	}
	if got := fx.readConfig(); got != original {
		t.Errorf("undo must restore byte-identical content:\ngot  %q\nwant %q", got, original)
	}
	if _, err := os.Stat(managed); !os.IsNotExist(err) {
		t.Error("undo must remove the managed file it created")
	}
	if _, err := e.Undo(); err == nil {
		t.Error("undoing twice must report that there is nothing left")
	}
}

func TestUndoRefusesAfterTheManagedFileIsEdited(t *testing.T) {
	fx := newFixture(t)
	fx.writeConfig("font-size = 13\n")
	e := fx.openEngine()
	if _, err := e.Apply(Change{Revision: e.State().Revision, Label: "Size", Values: Values{"font-size": {"16"}}}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	managed := filepath.Join(filepath.Dir(fx.config), managedFileName)
	if err := os.WriteFile(managed, []byte("# hand edited\nfont-size = 20\n"), 0o644); err != nil {
		t.Fatalf("edit managed file: %v", err)
	}
	_, err := e.Undo()
	if err == nil {
		t.Fatal("undo must refuse when the managed file was edited by hand")
	}
	if !strings.Contains(err.Error(), "changed after") && !strings.Contains(err.Error(), "graph changed") {
		t.Errorf("the refusal must explain the edit: %v", err)
	}
}

// hideInstalledBundle points the last-resort bundle locations at paths that do
// not exist, so a test can run as if Ghostty were absent on this machine.
func hideInstalledBundle(t *testing.T) {
	t.Helper()
	previousBinary, previousResources := bundleBinary, bundleResources
	bundleBinary = filepath.Join(t.TempDir(), "Absent.app", "Contents", "MacOS", "ghostty")
	bundleResources = filepath.Join(t.TempDir(), "Absent.app", "Contents", "Resources", "ghostty")
	t.Cleanup(func() {
		bundleBinary, bundleResources = previousBinary, previousResources
	})
}

func TestIncludeGraphOrderAndRelativeIncludesSurviveApply(t *testing.T) {
	fx := newFixture(t)
	childDir := filepath.Join(filepath.Dir(fx.config), "auto")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	childPath := filepath.Join(childDir, "theme.ghostty")
	childOriginal := "# child theme\nbackground = #001122\nforeground = #ddeeff\n"
	if err := os.WriteFile(childPath, []byte(childOriginal), 0o644); err != nil {
		t.Fatalf("write child: %v", err)
	}
	rootOriginal := "font-size = 13\nconfig-file = ?auto/theme.ghostty\n"
	fx.writeConfig(rootOriginal)

	e := fx.openEngine()
	sources := e.State().Sources
	if len(sources) != 2 || sources[0] != fx.config || sources[1] != childPath {
		t.Fatalf("include graph order = %v, want [%s %s]", sources, fx.config, childPath)
	}
	if got := e.State().Current.Last("background"); got != "#001122" {
		t.Errorf("the included file must contribute values, got %q", got)
	}

	if _, err := e.Apply(Change{Revision: e.State().Revision, Label: "Size", Values: Values{"font-size": {"16"}}}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	// The include is appended to the root, never to the user's own include.
	if got, err := os.ReadFile(childPath); err != nil || string(got) != childOriginal {
		t.Errorf("the included file must stay byte-identical, got %q err %v", got, err)
	}
	root := fx.readConfig()
	if !strings.HasPrefix(root, rootOriginal) {
		t.Errorf("the root file must keep its content, got:\n%s", root)
	}
	if !strings.Contains(root, managedIncludeLine(managedPath(fx.config))) {
		t.Errorf("the managed include must be added to the root:\n%s", root)
	}
	if got := e.State().Current.Last("font-size"); got != "16" {
		t.Errorf("managed values must win over the included file, got %q", got)
	}
}

func TestThemeDirectivePrecedenceMatchesNativeBehaviour(t *testing.T) {
	fx := newFixture(t)
	fx.writeTheme("MyDracula", "#282a36", "#f8f8f2")
	fx.writeTheme("MyNord", "#2e3440", "#d8dee9")

	childPath := filepath.Join(filepath.Dir(fx.config), "later.conf")
	if err := os.WriteFile(childPath, []byte("theme = MyNord\n"), 0o644); err != nil {
		t.Fatalf("write child: %v", err)
	}
	// The explicit colour is written before the theme directive, and a later
	// include sets a different theme. Verified against the installed build:
	// native keeps the explicit colour and the last theme.
	fx.writeConfig("foreground = #123abc\ntheme = MyDracula\nconfig-file = later.conf\n")

	e := fx.openEngine()
	current := e.State().Current
	if got := current.Last("theme"); got != "MyNord" {
		t.Errorf("the last theme directive must win, got %q", got)
	}
	if got := current.Last("foreground"); got != "#123abc" {
		t.Errorf("an explicit colour must beat every theme, got %q", got)
	}
	if got := current.Last("background"); got != "#2e3440" {
		t.Errorf("the winning theme must still supply its other colours, got %q", got)
	}
}

func TestSymlinkedConfigurationIsWrittenThroughToItsTarget(t *testing.T) {
	fx := newFixture(t)
	target := filepath.Join(fx.root, "real-config")
	original := "background-opacity = 0.8\n"
	if err := os.WriteFile(target, []byte(original), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	link := filepath.Join(fx.home, ".config", "ghostty", "linked-config")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	fx.config = link
	e := fx.openEngine()

	if _, err := e.Apply(Change{Revision: e.State().Revision, Label: "Size", Values: Values{"font-size": {"15"}}}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	// The symlink must survive and the target must carry the change.
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("stat link: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("the configuration symlink must not be replaced by a regular file")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if !strings.HasPrefix(string(data), original) || !strings.Contains(string(data), managedIncludeLine(managedPath(link))) {
		t.Errorf("the symlink target must receive the include:\n%s", data)
	}
}

func TestMissingConfigurationIsCreatedOnApply(t *testing.T) {
	fx := newFixture(t)
	missing := filepath.Join(fx.root, "nested", "dir", "config.ghostty")
	e, err := open(
		Options{ConfigPath: missing, StateDir: fx.state, GhosttyPath: fx.bin},
		fx.runner,
		func(string) string { return "" },
		fx.home,
		func(string) (string, error) { return fx.bin, nil },
	)
	if err != nil {
		t.Fatalf("open with a missing configuration: %v", err)
	}
	state := e.State()
	if len(state.Sources) != 0 {
		t.Errorf("a missing configuration has no sources, got %v", state.Sources)
	}
	var warned bool
	for _, warning := range state.Warnings {
		if strings.Contains(warning, "does not exist yet") {
			warned = true
		}
	}
	if !warned {
		t.Errorf("the missing configuration must be reported: %v", state.Warnings)
	}
	if err := e.Refresh(); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Error("reading a missing configuration must not create it")
	}

	if _, err := e.Apply(Change{Revision: e.State().Revision, Label: "First", Values: Values{"background-opacity": {"0.9"}}}); err != nil {
		t.Fatalf("apply into a missing configuration: %v", err)
	}
	data, err := os.ReadFile(missing)
	if err != nil {
		t.Fatalf("read created configuration: %v", err)
	}
	if !strings.Contains(string(data), managedIncludeLine(managedPath(missing))) {
		t.Errorf("the created configuration must include the managed file:\n%s", data)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(missing), managedFileName)); err != nil {
		t.Errorf("the managed file must be created next to the configuration: %v", err)
	}
}

func TestConcurrentTransactionsAreSerialised(t *testing.T) {
	fx := newFixture(t)
	fx.writeConfig("font-size = 13\n")
	e := fx.openEngine()
	revision := e.State().Revision

	done := make(chan error, 2)
	for _, size := range []string{"15", "16"} {
		go func(size string) {
			_, err := e.Apply(Change{Revision: revision, Label: "Size", Values: Values{"font-size": {size}}})
			done <- err
		}(size)
	}
	successes := 0
	for range 2 {
		if err := <-done; err == nil {
			successes++
		}
	}
	// The lock serialises the writes; the stale revision check rejects the
	// second transaction, which was staged against the same revision.
	if successes != 1 {
		t.Errorf("exactly one of two racing applies must succeed, got %d", successes)
	}
	if err := e.Refresh(); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if got := e.State().Current.Last("font-size"); got != "15" && got != "16" {
		t.Errorf("the surviving transaction must be intact, got %q", got)
	}
}
