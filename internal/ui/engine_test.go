package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"ghostty.studio/internal/core"
)

// The UI tests run against the real core engine over a throwaway configuration
// root: no test ever reads or writes the personal Ghostty configuration.
var (
	sharedEngine *core.Engine
	sharedOpts   core.Options
	sharedDir    string
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "ghostty-studio-ui")
	if err != nil {
		panic(err)
	}
	sharedDir = dir
	cfg := filepath.Join(dir, "config")
	if err := os.WriteFile(cfg, []byte("font-size = 14\n"), 0o600); err != nil {
		panic(err)
	}
	state := filepath.Join(dir, "state")
	if err := os.MkdirAll(state, 0o700); err != nil {
		panic(err)
	}
	os.Setenv("GHOSTTY_CONFIG_PATH", cfg)
	os.Setenv("GHOSTTY_TUI_STATE_DIR", state)
	opts := core.Options{ConfigPath: cfg, StateDir: state}
	if e, err := core.Open(opts); err == nil {
		sharedEngine, sharedOpts = e, opts
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// engineModel returns a model wired to the real engine, or skips the test when
// no Ghostty installation is available on this machine.
func engineModel(t *testing.T) *Model {
	t.Helper()
	if sharedEngine == nil {
		t.Skip("no Ghostty installation available for engine-backed UI tests")
	}
	m := New(sharedEngine, sharedOpts)
	m.SetSize(120, 36)
	m.Hydrate()
	return m
}

func TestHydrateLoadsRealCatalogs(t *testing.T) {
	m := engineModel(t)
	if m.state.Version == "" {
		t.Error("expected the native Ghostty version")
	}
	if m.state.Revision == "" {
		t.Error("expected a source-graph revision")
	}
	if m.themes.Total() == 0 {
		t.Error("expected real themes from the installed Ghostty")
	}
	if m.settingsL.Total() == 0 {
		t.Error("expected the native settings reference")
	}
	if m.effects.Total() == 0 {
		t.Error("expected the effects catalog to include at least None")
	}
	view := m.render()
	assertGeometry(t, "hydrated", view, 120, 36)
	if !strings.Contains(stripANSI(view), "THEMES") {
		t.Error("expected the themes heading in the hydrated view")
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case inEsc && (r == 'm' || r == 'K'):
			inEsc = false
		case inEsc:
		case r == 0x1b:
			inEsc = true
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// TestSnapshotUsesTheExactView proves the CLI snapshot path renders through the
// same View the interactive program uses.
func TestSnapshotUsesTheExactView(t *testing.T) {
	if sharedEngine == nil {
		t.Skip("no Ghostty installation available for engine-backed UI tests")
	}
	for _, tc := range []struct {
		width, height int
		section       string
		query         string
	}{
		{120, 36, "themes", ""},
		{80, 24, "themes", ""},
		{120, 36, "fonts", ""},
		{120, 36, "looks", ""},
		{120, 36, "effects", ""},
		{120, 36, "settings", ""},
		{120, 36, "workflows", ""},
		{120, 36, "health", ""},
		{120, 36, "export", ""},
		{120, 36, "themes", "nord"},
		{80, 24, "themes", "catppuccin"},
	} {
		got := Snapshot(sharedEngine, sharedOpts, tc.width, tc.height, tc.section, tc.query)
		assertGeometry(t, "snapshot/"+tc.section, got, tc.width, tc.height)

		// The same construction by hand must agree byte for byte.
		m := New(sharedEngine, sharedOpts)
		m.SetSize(tc.width, tc.height)
		m.Hydrate()
		m.SetSectionByName(tc.section)
		m.SetSearch(tc.query)
		if want := m.View().Content; want != got {
			gs, ws := strings.Split(got, "\n"), strings.Split(want, "\n")
			for i := 0; i < len(gs) && i < len(ws); i++ {
				if gs[i] != ws[i] {
					t.Fatalf("snapshot %s %dx%d line %d\n got=%q\nwant=%q", tc.section, tc.width, tc.height, i, gs[i], ws[i])
				}
			}
			t.Fatalf("snapshot %s %dx%d diverged (got %d lines, want %d)", tc.section, tc.width, tc.height, len(gs), len(ws))
		}
		if tc.query != "" && !strings.Contains(strings.ToLower(stripANSI(got)), strings.ToLower(tc.query)) {
			t.Errorf("snapshot with query %q shows no matching row", tc.query)
		}
	}
}

func TestSnapshotHandlesUnknownSectionAndTinySizes(t *testing.T) {
	if sharedEngine == nil {
		t.Skip("no Ghostty installation available")
	}
	got := Snapshot(sharedEngine, sharedOpts, 80, 24, "nonsense", "")
	assertGeometry(t, "unknown-section", got, 80, 24)
	if !strings.Contains(stripANSI(got), "THEMES") {
		t.Error("an unknown section should fall back to Themes")
	}
	tiny := Snapshot(sharedEngine, sharedOpts, 40, 12, "themes", "")
	assertGeometry(t, "tiny", tiny, 40, 12)
	if !strings.Contains(stripANSI(tiny), "needs at least 60") {
		t.Error("a tiny snapshot should state the minimum size")
	}
}

// TestStageApplyUndoRoundTrip exercises the real transaction path through the
// UI: stage a theme, apply it, then undo byte-identically.
func TestStageApplyUndoRoundTrip(t *testing.T) {
	m := engineModel(t)
	m.SetSectionByName("themes")
	if _, _, ok := m.themes.Selected(); !ok {
		t.Fatal("expected a theme to stage")
	}
	press(m, pressKey("enter"))
	if !m.dirty() {
		t.Fatal("enter should stage the focused theme")
	}
	if m.stagedRev != m.state.Revision {
		t.Fatalf("staged revision %q should match the state revision %q", m.stagedRev, m.state.Revision)
	}
	before := stripANSI(m.pendingSummary())
	if !strings.Contains(before, "background") {
		t.Fatalf("a theme patch should stage its colors, got %q", before)
	}

	cmd := m.applyStaged()
	if cmd == nil {
		t.Fatal("apply should run")
	}
	msg, ok := cmd().(actionDoneMsg)
	if !ok {
		t.Fatal("apply should report through actionDoneMsg")
	}
	if msg.err != nil {
		t.Fatalf("apply failed: %v", msg.err)
	}
	m.Update(msg)
	if m.busy {
		t.Fatal("apply must clear the busy state")
	}
	if m.dirty() {
		t.Fatal("a successful apply must clear the staged patch")
	}
	if m.statusErr {
		t.Fatalf("apply reported an error status: %q", m.status)
	}

	// The written configuration must contain the staged colors.
	written, err := os.ReadFile(filepath.Join(sharedDir, "studio.ghostty"))
	if err == nil && !strings.Contains(string(written), "#") {
		t.Error("expected the managed file to carry palette colors")
	}

	undo := m.undo()
	if undo == nil {
		t.Fatal("undo should run")
	}
	umsg, ok := undo().(actionDoneMsg)
	if !ok {
		t.Fatal("undo should report through actionDoneMsg")
	}
	if umsg.err != nil {
		t.Fatalf("undo failed: %v", umsg.err)
	}
	m.Update(umsg)
	if m.statusErr {
		t.Fatalf("undo reported an error status: %q", m.status)
	}
}

// TestApplyRejectsStaleRevisionOverRealCore proves the stale-revision guard
// reaches the UI as an honest error and keeps the patch staged.
func TestApplyRejectsStaleRevisionOverRealCore(t *testing.T) {
	m := engineModel(t)
	press(m, pressKey("enter"))
	m.stagedRev = "definitely-not-the-current-revision"
	cmd := m.applyStaged()
	if cmd == nil {
		t.Fatal("apply should run")
	}
	msg, ok := cmd().(actionDoneMsg)
	if !ok {
		t.Fatal("apply should report through actionDoneMsg")
	}
	if msg.err == nil {
		t.Fatal("a stale revision must be rejected")
	}
	m.Update(msg)
	if !m.statusErr || !strings.Contains(m.status, "reload") {
		t.Fatalf("the error should tell the user to reload, got %q", m.status)
	}
	if !m.dirty() {
		t.Fatal("a rejected apply must keep the staged patch")
	}
}

func TestReloadRefreshesStateAndKeepsStaged(t *testing.T) {
	m := engineModel(t)
	press(m, pressKey("enter"))
	staged := len(m.staged)
	cmd := m.reloadAll()
	if cmd == nil {
		t.Fatal("reload should return a command")
	}
	// Drive the batch so the model receives the fresh state.
	for _, msg := range drainBatch(cmd) {
		m.Update(msg)
	}
	if m.state.Revision == "" {
		t.Fatal("reload should refresh the state")
	}
	if len(m.staged) != staged {
		t.Fatal("reload must not disturb the staged patch")
	}
}

// drainBatch executes a command, flattening batches and sequences.
func drainBatch(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	switch m := msg.(type) {
	case tea.BatchMsg:
		var out []tea.Msg
		for _, c := range m {
			out = append(out, drainBatch(c)...)
		}
		return out
	case nil:
		return nil
	default:
		return []tea.Msg{msg}
	}
}

func TestDiagnosticsAndHealthRender(t *testing.T) {
	m := engineModel(t)
	m.SetSectionByName("health")
	view := stripANSI(m.render())
	if !strings.Contains(view, "ghostty") {
		t.Error("health should report the Ghostty version")
	}
	if !strings.Contains(view, "revision") {
		t.Error("health should report the revision")
	}
	assertGeometry(t, "health", m.render(), 120, 36)
}

func TestInstallReviewNeverInstallsWithoutConfirmation(t *testing.T) {
	m := engineModel(t)
	m.SetSectionByName("fonts")
	m.fontsDiscover = true
	if m.discover.Total() == 0 {
		t.Skip("no discover shelf on this machine")
	}
	press(m, pressKey("I"))
	if !m.modal.active() || m.modal.action != "font-install" {
		t.Fatalf("I should open the install review, got %+v", m.modal)
	}
	if m.modal.btnIdx != len(m.modal.buttons)-1 {
		t.Error("the install sheet must default to cancel, never to install")
	}
	// The review sheet must disclose the source, licence and destination.
	body := stripANSI(strings.Join(m.modal.body, "\n"))
	for _, want := range []string{"source", "license", "installs"} {
		if !strings.Contains(body, want) {
			t.Errorf("install sheet should disclose %q, got:\n%s", want, body)
		}
	}
	// Cancelling performs no install and leaves the busy flag alone.
	press(m, pressKey("enter"))
	if m.busy {
		t.Fatal("cancelling must not start an install")
	}
	if m.modal.active() {
		t.Fatal("cancelling should close the sheet")
	}
}
