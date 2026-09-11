package ui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"ghostty.studio/internal/core"
)

func TestCompactHelpKeepsItsScrollingHint(t *testing.T) {
	m := loadedModel()
	m.SetSize(80, 24)
	m.openHelp()
	view := stripANSI(m.render())
	if !strings.Contains(view, "PgUp/PgDn scroll") || !strings.Contains(view, "close") {
		t.Fatalf("compact help lost its controls:\n%s", view)
	}
}

func TestEditedFallbackChainActuallyReplacesTheInheritedChain(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "config.ghostty")
	if err := os.WriteFile(config, []byte("font-family = Menlo\nfont-family = monospace\n"), 0600); err != nil {
		t.Fatal(err)
	}
	opts := core.Options{ConfigPath: config, StateDir: filepath.Join(dir, "state")}
	e, err := core.Open(opts)
	if err != nil || e.State().Version == "" {
		t.Skip("native Ghostty validator unavailable")
	}
	m := New(e, opts)
	m.Hydrate()
	m.openSettingKeyEditor("font-family")
	m.modal.fields[0].input.SetValue(`["Monaspace Neon","Menlo"]`)
	m.modal.focusButtons()
	m.submitModal()
	cmd := m.applyStaged()
	if cmd == nil {
		t.Fatal("edited family could not be applied")
	}
	done := cmd().(actionDoneMsg)
	if done.err != nil {
		t.Fatal(done.err)
	}
	if got := e.State().Current["font-family"]; !slices.Equal(got, []string{"Monaspace Neon", "Menlo"}) {
		t.Fatalf("old primary font leaked into new chain: %v", got)
	}
}

func TestResetCanDiscardInvalidListInput(t *testing.T) {
	m := loadedModel()
	m.openSettingKeyEditor("font-family")
	m.modal.fields[0].input.SetValue("not JSON")
	m.modal.focusButtons()
	m.modal.btnIdx = 1
	m.submitModal()
	if m.modal.active() {
		t.Fatal("reset should discard invalid input")
	}
	if value, present := m.staged["font-family"]; !present || len(value) != 0 {
		t.Fatalf("expected a staged removal, got %v", value)
	}
}

func TestRestagingAfterExternalRefreshStartsAFreshTransaction(t *testing.T) {
	m := loadedModel()
	m.state.Revision = "original"
	m.stagedRev = ""
	m.stagePatch(core.Values{"font-size": {"19"}}, "old edit")
	m.Update(stateLoadedMsg{st: core.State{Revision: "external-edit", Current: core.Values{"font-size": {"14"}}}})
	if m.stagedRev != "original" {
		t.Fatal("refresh silently rebased the old patch")
	}
	m.stagePatch(core.Values{"background-opacity": {"0.95"}}, "new choice")
	if m.stagedRev != "external-edit" {
		t.Fatal("explicit re-staging did not start a new transaction")
	}
	if _, remains := m.staged["font-size"]; remains {
		t.Fatal("an unreviewed stale value was carried forward")
	}
}
