package core

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// brewCallsFor returns the Homebrew invocations the fake recorded.
func brewCallsFor(fx *testFixture) []Command {
	var out []Command
	for _, call := range fx.runner.commands() {
		if filepath.Base(call.Path) == "brew" {
			out = append(out, call)
		}
	}
	return out
}

// probeCallsFor returns the recorded targeted family probes.
func probeCallsFor(fx *testFixture) []string {
	var out []string
	for _, call := range fx.runner.commands() {
		for _, arg := range call.Args {
			if strings.HasPrefix(arg, "--family=") {
				out = append(out, strings.TrimPrefix(arg, "--family="))
			}
		}
	}
	return out
}

func TestInstallFontRunsBrewWithoutAShellAndVerifiesDiscovery(t *testing.T) {
	fx := newFixture(t)
	// A real cask install makes the family visible to a targeted probe.
	fx.runner.ghostty.brewInstalls = map[string]string{"font-maple-mono": "Maple Mono"}
	e := fx.openEngine()

	result, err := e.InstallFont("discover:font-maple-mono")
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	brew := brewCallsFor(fx)
	if len(brew) != 1 {
		t.Fatalf("expected exactly one Homebrew invocation, got %d", len(brew))
	}
	if !equalStrings(brew[0].Args, []string{"install", "--cask", "font-maple-mono"}) {
		t.Errorf("brew args = %v, want [install --cask font-maple-mono]", brew[0].Args)
	}
	if !strings.Contains(result.Message, "can be selected") {
		t.Errorf("a confirmed family must be reported as selectable: %q", result.Message)
	}
	if !strings.Contains(result.Message, "Nothing was written to your Ghostty configuration") {
		t.Errorf("the install must state that configuration was untouched: %q", result.Message)
	}
	// The family must now be selectable from the font catalog too.
	var found bool
	for _, font := range e.Catalog(SectionFonts) {
		if font.Name == "Maple Mono" {
			found = true
		}
	}
	if !found {
		t.Error("an installed family must appear in the font catalog")
	}
}

// TestInstallFontReturnsWhenBrewInstallsButGhosttyDoesNotReportTheFamily covers
// the honest-failure branch: the install succeeded, but the studio must not
// claim the family is selectable.
func TestInstallFontReturnsWhenBrewInstallsButGhosttyDoesNotReportTheFamily(t *testing.T) {
	fx := newFixture(t)
	e := fx.openEngine()

	result, err := e.InstallFont("discover:font-maple-mono")
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if strings.Contains(result.Message, "can be selected") {
		t.Errorf("the studio must not claim selection before Ghostty reports it: %q", result.Message)
	}
	if len(result.Warnings) == 0 {
		t.Error("an unconfirmed install must warn")
	}
	if len(brewCallsFor(fx)) != 1 {
		t.Error("the install must have been attempted exactly once")
	}
}

// TestInstallFontReturnsForAnAlreadyAvailableFamily is the regression for the
// deadlock the parent found: InstallFont holds the engine lock, so the
// verification path must never call the public, locking Refresh. A hanging
// implementation fails this test by timing out.
func TestInstallFontReturnsForAnAlreadyAvailableFamily(t *testing.T) {
	fx := newFixture(t)
	// Commit Mono is installed by hand on this machine and is invisible to the
	// unfiltered listing; only the targeted probe finds it.
	fx.runner.ghostty.familyProbes = map[string]string{
		"CommitMono": "CommitMono\n  CommitMono Regular\n  CommitMono Bold\n",
	}
	e := fx.openEngine()

	type outcome struct {
		result Result
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		result, err := e.InstallFont("discover:font-commit-mono")
		done <- outcome{result, err}
	}()
	var got outcome
	select {
	case got = <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("InstallFont did not return: the locked verification path deadlocked")
	}
	if got.err != nil {
		t.Fatalf("install: %v", got.err)
	}
	if !strings.Contains(got.result.Message, "already available") {
		t.Errorf("a family Ghostty already has must not be reinstalled: %q", got.result.Message)
	}
	if brewCallsFor(fx) != nil {
		t.Error("no font may be installed when the family is already available")
	}
}

func TestInstallFontReportsHomebrewFailures(t *testing.T) {
	fx := newFixture(t)
	fx.writeConfig("font-size = 13\n")
	e := fx.openEngine()
	original := fx.readConfig()
	fx.runner.brewErr = &execExitError{code: 1}
	fx.runner.brewOut = "Error: Cask 'font-maple-mono' is unavailable"
	_, err := e.InstallFont("font-maple-mono")
	if err == nil {
		t.Fatal("a failing install must return an error")
	}
	if !strings.Contains(err.Error(), "unavailable") {
		t.Errorf("the Homebrew output must be surfaced: %v", err)
	}
	if got := fx.readConfig(); got != original {
		t.Error("a failed install must not touch the configuration")
	}
}

// TestTargetedFontProbesCoverFamiliesMissingFromTheNativeListing is the
// regression for the real gap found on this machine: Commit Mono is installed
// but absent from `ghostty +list-fonts`, and a bundle such as Monaspace answers
// only to its five registered family names.
func TestTargetedFontProbesCoverFamiliesMissingFromTheNativeListing(t *testing.T) {
	fx := newFixture(t)
	fx.runner.ghostty.fonts = "Iosevka\n  Iosevka\n\nMenlo\n  Menlo\n"
	fx.runner.ghostty.familyProbes = map[string]string{
		"CommitMono":      "CommitMono\n  CommitMono Regular\n  CommitMono Bold Italic\n",
		"Monaspace Argon": "Monaspace Argon\n  Monaspace Argon Regular\n",
		"Monaspace Xenon": "Monaspace Xenon\n  Monaspace Xenon Regular\n",
	}
	e := fx.openEngine()

	names := map[string]bool{}
	for _, font := range e.Catalog(SectionFonts) {
		names[font.Name] = true
	}
	for _, want := range []string{"CommitMono", "Monaspace Argon", "Monaspace Xenon"} {
		if !names[want] {
			t.Errorf("%s is installed but missing from the font catalog: %v", want, names)
		}
	}

	probes := probeCallsFor(fx)
	probed := map[string]bool{}
	for _, name := range probes {
		probed[name] = true
	}
	for _, want := range []string{"Monaspace Argon", "Monaspace Neon", "Monaspace Krypton", "Monaspace Radon", "Monaspace Xenon", "CommitMono"} {
		if !probed[want] {
			t.Errorf("the curated probe list must include %q, got %v", want, probes)
		}
	}
	if probed["Iosevka"] {
		t.Error("families the unfiltered listing already reports must not be probed again")
	}

	// The discover shelf must recognise the probed families as installed.
	shelf := map[string]Choice{}
	for _, choice := range e.Catalog(SectionDiscover) {
		shelf[choice.Name] = choice
	}
	if !shelf["Commit Mono"].Installed {
		t.Error("Commit Mono is available natively and must be marked installed")
	}
	if !shelf["Monaspace"].Installed {
		t.Error("Monaspace is available natively and must be marked installed")
	}
	if shelf["Maple Mono"].Installed {
		t.Error("Maple Mono is not installed in this fixture")
	}

	// Health must state the listing gap instead of hiding it.
	var reported bool
	for _, finding := range e.Diagnose() {
		if strings.Contains(finding.Title, "native listing") && strings.Contains(finding.Detail, "CommitMono") {
			reported = true
		}
	}
	if !reported {
		t.Errorf("the native listing gap must be reported: %+v", e.Diagnose())
	}
	// Probing must not touch the configuration.
	if values, ok := e.State().Current.Get("font-family"); ok && len(values) > 0 {
		t.Errorf("probing must not write configuration values, got %v", values)
	}
}

// TestTargetedProbesFollowConfiguredFamilies covers the second half of the
// requirement: families the user's own configuration asks for are probed too.
func TestTargetedProbesFollowConfiguredFamilies(t *testing.T) {
	fx := newFixture(t)
	fx.runner.ghostty.fonts = ""
	fx.runner.ghostty.familyProbes = map[string]string{
		"CommitMono": "CommitMono\n  CommitMono Regular\n",
	}
	fx.writeConfig("font-family = CommitMono\n")
	e := fx.openEngine()

	var found bool
	for _, font := range e.Catalog(SectionFonts) {
		if font.Name == "CommitMono" {
			found = true
		}
	}
	if !found {
		t.Error("a configured family that the native listing omits must still be listed")
	}
	var probedCommit bool
	for _, name := range probeCallsFor(fx) {
		if name == "CommitMono" {
			probedCommit = true
		}
	}
	if !probedCommit {
		t.Errorf("configured families must be probed: %v", probeCallsFor(fx))
	}
	if got := e.State().Current.Last("font-family"); got != "CommitMono" {
		t.Errorf("probing must not disturb the effective configuration, got %q", got)
	}
}
