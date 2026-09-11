package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// These exercise the public CLI through a real installed Ghostty validator.
// Every writable path is under t.TempDir; no personal configuration is changed.
func nativeCLI(t *testing.T) (func(...string) (string, error), string, string) {
	t.Helper()
	bin := os.Getenv("GHOSTTY_BIN")
	if bin == "" {
		bin = "/Applications/Ghostty.app/Contents/MacOS/ghostty"
		if _, err := os.Stat(bin); err != nil {
			bin, _ = exec.LookPath("ghostty")
		}
	}
	if bin == "" {
		t.Skip("native integration requires Ghostty")
	}
	root := t.TempDir()
	config := filepath.Join(root, "config.ghostty")
	original := "# Personal comments stay exactly as written.\nfont-family = Menlo\nfont-family = monospace\nbackground-opacity = 0.8\n"
	if err := os.WriteFile(config, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	assets, err := filepath.Abs("../../assets")
	if err != nil {
		t.Fatal(err)
	}
	invoke := func(args ...string) (string, error) {
		var out, diagnostic bytes.Buffer
		full := []string{"--ghostty", bin, "--config", config, "--state-dir", filepath.Join(root, "state"), "--assets", assets}
		err := run(append(full, args...), &out, &diagnostic)
		return out.String() + diagnostic.String(), err
	}
	return invoke, config, original
}

func TestNativeCLIChangeLifecycle(t *testing.T) {
	invoke, config, original := nativeCLI(t)
	output, err := invoke("diff", "--set", "background-opacity=0.95")
	if err != nil || !strings.Contains(output, "0.95") {
		t.Fatalf("diff: %s %v", output, err)
	}
	if b, _ := os.ReadFile(config); string(b) != original {
		t.Fatal("diff changed the configuration")
	}
	output, err = invoke("apply", "--set", "background-opacity=0.95")
	if err != nil || !strings.Contains(output, "Reload Ghostty") {
		t.Fatalf("apply must save and explain reload: %s %v", output, err)
	}
	managed, err := os.ReadFile(filepath.Join(filepath.Dir(config), "studio.ghostty"))
	if err != nil || !strings.Contains(string(managed), "background-opacity = 0.95") {
		t.Fatalf("exact opacity was not saved: %s %v", managed, err)
	}
	before, _ := os.ReadFile(config)
	if _, err = invoke("apply", "--set", "background-opacity=banana"); err == nil {
		t.Fatal("invalid value was accepted")
	}
	if after, _ := os.ReadFile(config); !bytes.Equal(before, after) {
		t.Fatal("rejected change mutated the configuration")
	}
	if output, err = invoke("undo"); err != nil {
		t.Fatalf("undo: %s %v", output, err)
	}
	if b, _ := os.ReadFile(config); string(b) != original {
		t.Fatal("undo did not restore original bytes")
	}
}

func TestNativeCLILookRoundTrip(t *testing.T) {
	invoke, config, original := nativeCLI(t)
	if out, err := invoke("look", "save", "My reading room"); err != nil {
		t.Fatalf("save: %s %v", out, err)
	}
	export := filepath.Join(filepath.Dir(config), "reading room.json")
	if out, err := invoke("look", "export", "My reading room", export); err != nil {
		t.Fatalf("export: %s %v", out, err)
	}
	if out, err := invoke("look", "import", export); err != nil {
		t.Fatalf("import: %s %v", out, err)
	}
	if out, err := invoke("diff", "--look", "My reading room"); err != nil {
		t.Fatalf("imported Look cannot be selected: %s %v", out, err)
	}
	if b, _ := os.ReadFile(config); string(b) != original {
		t.Fatal("Look library operations changed Ghostty configuration")
	}
	malicious := filepath.Join(filepath.Dir(config), "command.json")
	if err := os.WriteFile(malicious, []byte(`{"format":"ghostty-studio/look@1","name":"Bad look","values":{"command":["touch /tmp/should-not-run"]}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := invoke("look", "import", malicious); err == nil {
		t.Fatal("Look import accepted an executable command")
	}
}
