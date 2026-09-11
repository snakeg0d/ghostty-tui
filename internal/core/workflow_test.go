package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// keybindFixture prepares a configuration with user bindings, including one the
// scrollback search workflow collides with, plus default bindings Ghostty ships.
func keybindFixture(t *testing.T) *testFixture {
	t.Helper()
	fx := newFixture(t)
	// Ghostty's own defaults, as the build reports them.
	fx.runner.ghostty.defaults["keybind"] = []string{
		"super+shift+,=reload_config",
		"super+q=quit",
		"super+t=new_tab",
		"super+w=close_surface",
		"copy=copy_to_clipboard:mixed",
	}
	fx.runner.known = knownKeys(fx.runner.ghostty.defaults)
	fx.writeConfig(strings.Join([]string{
		"# my bindings",
		"keybind = super+f=toggle_fullscreen",
		"keybind = super+shift+g=text:git status\\r",
		"keybind = global:ctrl+grave_accent=toggle_quick_terminal",
		"", // trailing newline
	}, "\n"))
	return fx
}

// workflowChoice returns one workflow from the catalog.
func workflowChoice(t *testing.T, e *Engine, id string) Choice {
	t.Helper()
	for _, choice := range e.Catalog(SectionWorkflows) {
		if choice.ID == id {
			return choice
		}
	}
	t.Fatalf("workflow %s is missing from the catalog", id)
	return Choice{}
}

// TestWorkflowApplyPreservesUserAndDefaultKeybinds is the regression for the
// destructive binding reset: applying a workflow must never erase the user's
// bindings or Ghostty's defaults.
func TestWorkflowApplyPreservesUserAndDefaultKeybinds(t *testing.T) {
	fx := keybindFixture(t)
	e := fx.openEngine()
	original := fx.readConfig()

	before := e.State().Current["keybind"]
	if len(before) != 8 {
		t.Fatalf("expected five defaults plus three user bindings, got %v", before)
	}

	search := workflowChoice(t, e, "workflow:scrollback-search")
	if !strings.Contains(search.Description, "Conflict") {
		t.Errorf("the same-trigger conflict must stay disclosed before applying: %q", search.Description)
	}

	result, err := e.Apply(Change{Revision: e.State().Revision, Label: "Scrollback search", Values: search.Values})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if result.Message == "" {
		t.Fatal("apply must report what happened")
	}

	// The user's own file still holds every line they wrote.
	if got := fx.readConfig(); !strings.HasPrefix(got, original) {
		t.Errorf("the user's configuration must keep their bindings:\n%s", got)
	}
	managed := mustRead(t, filepath.Join(filepath.Dir(fx.config), managedFileName))
	if strings.Contains(string(managed), `keybind = ""`) {
		t.Errorf("the managed include must never reset the binding list:\n%s", managed)
	}
	if !strings.Contains(string(managed), "keybind = super+f=start_search\n") {
		t.Errorf("the managed include must add the workflow binding:\n%s", managed)
	}

	after := e.State().Current["keybind"]
	// Defaults survive.
	for _, binding := range []string{"super+shift+,=reload_config", "super+q=quit", "super+t=new_tab", "super+w=close_surface", "copy=copy_to_clipboard:mixed"} {
		if !containsString(after, binding) {
			t.Errorf("default binding %q was lost: %v", binding, after)
		}
	}
	// Unrelated user bindings survive.
	for _, binding := range []string{"super+shift+g=text:git status\\r", "global:ctrl+grave_accent=toggle_quick_terminal"} {
		if !containsString(after, binding) {
			t.Errorf("user binding %q was lost: %v", binding, after)
		}
	}
	// The requested trigger is still present exactly once and the workflow's
	// binding wins it, because Ghostty documents that a duplicate trigger
	// overwrites the previously set value.
	if countString(after, "super+f=toggle_fullscreen") != 1 {
		t.Errorf("the user's shadowed binding must stay listed exactly once: %v", after)
	}
	if after[len(after)-1] != "super+f=start_search" {
		t.Errorf("the workflow binding must be last so it wins its trigger: %v", after)
	}
	if countString(after, "super+f=start_search") != 1 {
		t.Errorf("the workflow binding must appear once: %v", after)
	}
	// After the apply the conflict is resolved, so it is no longer reported.
	if again := workflowChoice(t, e, "workflow:scrollback-search"); strings.Contains(again.Description, "Conflict") {
		t.Errorf("the resolved conflict must not be reported again: %q", again.Description)
	}
}

// TestWorkflowApplyKeepsEarlierWorkflowBindings covers the second half of the
// managed list: applying another workflow must not drop what the studio already
// bound.
func TestWorkflowApplyKeepsEarlierWorkflowBindings(t *testing.T) {
	fx := keybindFixture(t)
	e := fx.openEngine()
	original := fx.readConfig()

	quick := workflowChoice(t, e, "workflow:quick-terminal")
	if _, err := e.Apply(Change{Revision: e.State().Revision, Label: "Quick terminal", Values: quick.Values}); err != nil {
		t.Fatalf("apply quick terminal: %v", err)
	}
	search := workflowChoice(t, e, "workflow:scrollback-search")
	if _, err := e.Apply(Change{Revision: e.State().Revision, Label: "Scrollback search", Values: search.Values}); err != nil {
		t.Fatalf("apply scrollback search: %v", err)
	}

	after := e.State().Current["keybind"]
	for _, binding := range []string{
		"global:super+backquote=toggle_quick_terminal",
		"super+f=start_search",
		"super+q=quit",
		"super+shift+g=text:git status\\r",
	} {
		if !containsString(after, binding) {
			t.Errorf("binding %q must survive the second workflow: %v", binding, after)
		}
	}
	if got := fx.readConfig(); !strings.HasPrefix(got, original) {
		t.Error("neither workflow may rewrite the user's own configuration")
	}
	managed := string(mustRead(t, filepath.Join(filepath.Dir(fx.config), managedFileName)))
	for _, want := range []string{
		"keybind = global:super+backquote=toggle_quick_terminal\n",
		"keybind = super+f=start_search\n",
	} {
		if !strings.Contains(managed, want) {
			t.Errorf("the managed include must keep %q:\n%s", want, managed)
		}
	}
	if strings.Contains(managed, `keybind = ""`) {
		t.Errorf("the managed include must never reset the binding list:\n%s", managed)
	}
}

// TestWorkflowBindingApplyPassesNativeValidation runs the serialized result
// through the installed Ghostty validator. It only reads the binary and writes
// to a temporary directory.
func TestWorkflowBindingApplyPassesNativeValidation(t *testing.T) {
	bin := "/Applications/Ghostty.app/Contents/MacOS/ghostty"
	if _, err := os.Stat(bin); err != nil {
		t.Skip("no installed ghostty to validate against")
	}
	fx := keybindFixture(t)
	e, err := open(Options{ConfigPath: fx.config, StateDir: fx.state, GhosttyPath: bin}, fx.runner,
		func(string) string { return "" }, fx.home, func(string) (string, error) { return bin, nil })
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	// The workflow catalog is built from the same state the engine applies, and
	// the real validator runs inside Apply because GhosttyPath is the real one.
	for _, id := range []string{"workflow:quick-terminal", "workflow:scrollback-search"} {
		choice := workflowChoice(t, e, id)
		if _, err := e.Apply(Change{Revision: e.State().Revision, Label: choice.Name, Values: choice.Values}); err != nil {
			t.Fatalf("apply %s: %v", id, err)
		}
	}
	// The committed graph is what Ghostty loads; validate it directly.
	n := &native{bin: bin, runner: osRunner{}}
	diagnostics, err := n.validate(context.Background(), fx.config)
	if err != nil {
		t.Fatalf("ghostty could not validate the committed graph: %v", err)
	}
	if len(diagnostics) > 0 {
		t.Errorf("ghostty rejected the committed graph: %v", diagnostics)
	}
	// Every binding either file declares is still present in the effective set.
	effective := e.State().Current["keybind"]
	for _, binding := range []string{
		"super+f=toggle_fullscreen",
		"super+shift+g=text:git status\\r",
		"global:ctrl+grave_accent=toggle_quick_terminal",
		"global:super+backquote=toggle_quick_terminal",
		"super+f=start_search",
	} {
		if !containsString(effective, binding) {
			t.Errorf("binding %q must survive the native-validated apply: %v", binding, effective)
		}
	}
}

func containsString(list []string, want string) bool {
	for _, value := range list {
		if value == want {
			return true
		}
	}
	return false
}

func countString(list []string, want string) int {
	count := 0
	for _, value := range list {
		if value == want {
			count++
		}
	}
	return count
}

// TestMergedBindingsKeepsUntouchedTriggers pins the merge itself.
func TestMergedBindingsKeepsUntouchedTriggers(t *testing.T) {
	existing := []string{"global:super+backquote=toggle_quick_terminal", "super+f=start_search"}
	got := mergedBindings(existing, []string{"super+f=start_search", "super+k=clear_screen"})
	want := []string{"global:super+backquote=toggle_quick_terminal", "super+f=start_search", "super+k=clear_screen"}
	if !equalStrings(got, want) {
		t.Errorf("mergedBindings = %v, want %v", got, want)
	}
	if got := mergedBindings(nil, []string{"super+f=start_search"}); !equalStrings(got, []string{"super+f=start_search"}) {
		t.Errorf("a first apply must stage only its own binding, got %v", got)
	}
	if got := mergedBindings([]string{"", "super+a=reload_config"}, nil); !equalStrings(got, []string{"super+a=reload_config"}) {
		t.Errorf("a stale reset entry must never survive the merge, got %v", got)
	}
}
