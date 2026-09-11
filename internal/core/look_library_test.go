package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// lookNames lists a catalog section's names in order.
func lookNames(choices []Choice) []string {
	names := make([]string, 0, len(choices))
	for _, choice := range choices {
		names = append(names, choice.Name)
	}
	return names
}

// findLook resolves a Look the way the CLI does: by ID or name, case-insensitively.
func findLook(choices []Choice, name string) (Choice, bool) {
	for _, choice := range choices {
		if strings.EqualFold(choice.ID, name) || strings.EqualFold(choice.Name, name) {
			return choice, true
		}
	}
	return Choice{}, false
}

func TestSaveLookIsSelectableImmediatelyAndKeepsStableID(t *testing.T) {
	fx := newFixture(t)
	fx.writeConfig("background-opacity = 0.8\nfont-size = 14\n")
	e := fx.openEngine()

	before := e.Catalog(SectionLooks)
	if len(before) != 5 {
		t.Fatalf("expected the five presets, got %v", lookNames(before))
	}

	if err := e.SaveLook("My reading room", e.State().Current); err != nil {
		t.Fatalf("save: %v", err)
	}
	// The save must be visible without reopening the engine.
	after := e.Catalog(SectionLooks)
	if len(after) != 6 {
		t.Fatalf("a saved Look must appear in the catalog immediately, got %v", lookNames(after))
	}
	if !equalStrings(lookNames(after)[:5], lookNames(before)) {
		t.Errorf("the presets must keep their order: %v", lookNames(after))
	}
	saved := after[5]
	if saved.ID != "look:my-reading-room" {
		t.Errorf("saved Look id = %q, want look:my-reading-room", saved.ID)
	}
	if saved.Name != "My reading room" || saved.Source != savedLookSource || saved.Category != "look" {
		t.Errorf("saved Look metadata = %+v", saved)
	}
	if got := saved.Values.First("background-opacity"); got != "0.8" {
		t.Errorf("the saved appearance must carry the current value, got %q", got)
	}
	if _, ok := saved.Values["font-size"]; !ok {
		t.Error("the saved appearance must carry the typography keys")
	}
	for key := range saved.Values {
		if !lookAllowlist[key] {
			t.Errorf("a saved Look must not carry %q", key)
		}
	}

	// A fresh engine must see it too, and the ID must stay stable.
	e2 := fx.openEngine()
	found, ok := findLook(e2.Catalog(SectionLooks), "My reading room")
	if !ok || found.ID != "look:my-reading-room" {
		t.Fatalf("a saved Look must load from disk by name and ID: %+v ok=%t", found, ok)
	}
	if found, ok := findLook(e2.Catalog(SectionLooks), "look:my-reading-room"); !ok || found.Name != "My reading room" {
		t.Error("a saved Look must resolve by its stable ID")
	}
	// Re-saving the same name updates one entry rather than adding another.
	if err := e2.SaveLook("My reading room", e2.State().Current); err != nil {
		t.Fatalf("re-save: %v", err)
	}
	if got := len(e2.Catalog(SectionLooks)); got != 6 {
		t.Errorf("re-saving must update the entry, got %d entries", got)
	}
}

func TestSavedLookIDFollowsTheFileNameNotTheDisplayName(t *testing.T) {
	fx := newFixture(t)
	fx.writeConfig("font-size = 14\n")
	e := fx.openEngine()
	if err := e.SaveLook("My reading room", Values{"font-size": {"14"}}); err != nil {
		t.Fatalf("save: %v", err)
	}
	path := filepath.Join(e.looksDir(), "my-reading-room.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved Look: %v", err)
	}
	var doc lookFile
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("saved Look must be valid JSON: %v", err)
	}
	doc.Name = "Renamed look"
	renamed, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, renamed, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	e2 := fx.openEngine()
	found, ok := findLook(e2.Catalog(SectionLooks), "Renamed look")
	if !ok {
		t.Fatalf("the renamed Look must still load: %v", lookNames(e2.Catalog(SectionLooks)))
	}
	if found.ID != "look:my-reading-room" {
		t.Errorf("the ID must follow the file name, got %q", found.ID)
	}
}

func TestSavedLookLibraryIsDeterministicAndSkipsBadFilesWithAWarning(t *testing.T) {
	fx := newFixture(t)
	fx.writeConfig("font-size = 14\n")
	e := fx.openEngine()
	dir := e.looksDir()
	if err := ensureDir(dir); err != nil {
		t.Fatalf("ensure dir: %v", err)
	}

	// Two valid Looks written out of alphabetical order.
	for _, name := range []string{"Zebra afternoon", "Amber evening"} {
		doc, err := json.Marshal(lookFile{
			Format: lookFormat,
			Name:   name,
			Values: Values{"font-size": {"15"}},
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		file := filepath.Join(dir, slugify(name)+".json")
		if err := os.WriteFile(file, append(doc, '\n'), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	// A corrupt document and one that tries to smuggle an executable command.
	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatalf("write broken: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hostile.json"),
		[]byte(`{"format":"ghostty-studio/look@1","name":"Hostile","values":{"command":["touch /tmp/should-not-run"]}}`), 0o600); err != nil {
		t.Fatalf("write hostile: %v", err)
	}
	// A non-Look file must be ignored entirely.
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not a look"), 0o600); err != nil {
		t.Fatalf("write notes: %v", err)
	}

	e2 := fx.openEngine()
	looks := e2.Catalog(SectionLooks)
	names := lookNames(looks)
	want := []string{"Everyday", "Daylight", "Dense", "Presentation", "After Hours", "Amber evening", "Zebra afternoon"}
	if !equalStrings(names, want) {
		t.Fatalf("catalog order = %v, want %v", names, want)
	}
	for _, name := range names {
		if name == "Hostile" || name == "Broken" || strings.Contains(name, "notes") {
			t.Errorf("invalid library entries must be skipped, got %q", name)
		}
	}
	var warned int
	state := e2.State()
	for _, warning := range state.Warnings {
		if strings.Contains(warning, "saved Look") && strings.Contains(warning, "skipped") {
			warned++
		}
	}
	if warned != 2 {
		t.Errorf("the corrupt and hostile files must warn and be skipped, warnings: %v", state.Warnings)
	}
	// The library problem must never hide the presets.
	if len(state.Warnings) == 0 {
		t.Error("a broken library entry must be reported")
	}
	// Repeated refreshes must produce the same order and IDs.
	first := make([]string, 0, len(looks))
	for _, look := range looks {
		first = append(first, look.ID)
	}
	for range 3 {
		if err := e2.Refresh(); err != nil {
			t.Fatalf("refresh: %v", err)
		}
		again := e2.Catalog(SectionLooks)
		ids := make([]string, 0, len(again))
		for _, look := range again {
			ids = append(ids, look.ID)
		}
		if !equalStrings(ids, first) {
			t.Fatalf("catalog order must be deterministic: %v vs %v", ids, first)
		}
	}
}

func TestLookLibraryRoundTripThroughExportAndImport(t *testing.T) {
	fx := newFixture(t)
	fx.writeConfig("background-opacity = 0.8\nfont-feature = -calt\n")
	e := fx.openEngine()

	// The CLI sequence: save, export, import, then select the imported Look.
	if err := e.SaveLook("My reading room", e.State().Current); err != nil {
		t.Fatalf("save: %v", err)
	}
	saved, ok := findLook(e.Catalog(SectionLooks), "My reading room")
	if !ok {
		t.Fatalf("saved Look missing: %v", lookNames(e.Catalog(SectionLooks)))
	}
	exported := filepath.Join(fx.root, "reading room.json")
	if err := e.ExportLook(saved.Name, saved.Values, exported); err != nil {
		t.Fatalf("export: %v", err)
	}
	imported, err := e.ImportLook(exported)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if imported.ID != "look:my-reading-room" {
		t.Errorf("imported id = %q", imported.ID)
	}
	if err := e.SaveLook(imported.Name, imported.Values); err != nil {
		t.Fatalf("import into library: %v", err)
	}
	if _, ok := findLook(e.Catalog(SectionLooks), "My reading room"); !ok {
		t.Fatalf("the imported Look must be selectable: %v", lookNames(e.Catalog(SectionLooks)))
	}
	// An exported Look carries appearance settings only.
	raw := mustRead(t, exported)
	if strings.Contains(string(raw), `"command"`) || strings.Contains(string(raw), "keybind") {
		t.Errorf("an exported Look must strip unrelated configuration:\n%s", raw)
	}
	if !strings.Contains(string(raw), "background-opacity") {
		t.Errorf("an exported Look must keep the appearance:\n%s", raw)
	}
	// Importing an executable command must still be refused.
	hostile := filepath.Join(fx.root, "hostile.json")
	if err := os.WriteFile(hostile, []byte(`{"format":"ghostty-studio/look@1","name":"Bad","values":{"command":["touch /tmp/should-not-run"]}}`), 0o600); err != nil {
		t.Fatalf("write hostile: %v", err)
	}
	if _, err := e.ImportLook(hostile); err == nil {
		t.Fatal("an executable command must be refused on import")
	}
	if _, err := os.Stat("/tmp/should-not-run"); !os.IsNotExist(err) {
		t.Error("nothing may execute a Look value")
	}
}

func TestSavedLookAppliesAsAStagedPatch(t *testing.T) {
	fx := newFixture(t)
	fx.writeConfig("background-opacity = 0.8\nfont-size = 14\n")
	e := fx.openEngine()
	if err := e.SaveLook("Night desk", Values{"background-opacity": {"0.95"}, "font-size": {"16"}}); err != nil {
		t.Fatalf("save: %v", err)
	}
	look, ok := findLook(e.Catalog(SectionLooks), "Night desk")
	if !ok {
		t.Fatalf("saved Look missing: %v", lookNames(e.Catalog(SectionLooks)))
	}
	result, err := e.Apply(Change{Revision: e.State().Revision, Label: look.Name, Values: look.Values})
	if err != nil {
		t.Fatalf("apply saved Look: %v", err)
	}
	if !strings.Contains(result.Message, "Reload") {
		t.Errorf("apply must explain the reload: %q", result.Message)
	}
	current := e.State().Current
	if got := current.Last("background-opacity"); got != "0.95" {
		t.Errorf("saved opacity = %q, want 0.95", got)
	}
	if got := current.Last("font-size"); got != "16" {
		t.Errorf("saved font size = %q, want 16", got)
	}
}

// TestSavedLookRoundTripThroughTheStateDir covers the CLI's own path shape: the
// library lives under the state directory and survives a restart.
func TestSavedLookRoundTripThroughTheStateDir(t *testing.T) {
	fx := newFixture(t)
	fx.writeConfig("font-size = 14\n")
	e := fx.openEngine()
	exported := filepath.Join(fx.root, "shared look.json")
	if err := e.ExportLook("Shared look", Values{"font-size": {"18"}}, exported); err != nil {
		t.Fatalf("export: %v", err)
	}
	imported, err := e.ImportLook(exported)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if err := e.SaveLook(imported.Name, imported.Values); err != nil {
		t.Fatalf("save imported: %v", err)
	}
	if _, err := os.Stat(filepath.Join(fx.state, "looks", "shared-look.json")); err != nil {
		t.Fatalf("the library entry must live under the state directory: %v", err)
	}
	e2 := fx.openEngine()
	found, ok := findLook(e2.Catalog(SectionLooks), "Shared look")
	if !ok {
		t.Fatalf("restart must load the saved Look: %v", lookNames(e2.Catalog(SectionLooks)))
	}
	if got := found.Values.First("font-size"); got != "18" {
		t.Errorf("loaded value = %q, want 18", got)
	}
	var reported bool
	for _, finding := range e2.Diagnose() {
		if finding.Title == "Look library" && strings.Contains(finding.Detail, "1 saved Look") {
			reported = true
		}
	}
	if !reported {
		t.Errorf("health must report the Look library: %+v", e2.Diagnose())
	}
}
