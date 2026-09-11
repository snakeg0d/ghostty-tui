package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// entry is one parsed configuration directive, kept in file order.
type entry struct {
	key   string
	value string
	line  int
	raw   string
}

// configFile is one parsed configuration file in the include graph.
type configFile struct {
	// path is the file as referenced, which is what the user sees and what
	// relative includes resolve against.
	path string
	// real is the symlink-resolved path used for reading and writing. It equals
	// path when the file is not a symlink.
	real    string
	dir     string
	content []byte
	mode    fs.FileMode
	hash    string
	entries []entry
}

// include is a config-file directive found in a file.
type include struct {
	// raw is the directive value exactly as written, including any leading '?'.
	raw      string
	path     string
	optional bool
	line     int
}

// missingInclude records a config-file directive whose target is absent.
type missingInclude struct {
	path     string
	from     string
	optional bool
}

// parseEntries extracts the directives of a configuration file in order.
// Comments, blank lines and unrecognised lines are skipped here; the raw file
// content is preserved separately and is what gets rewritten on apply.
func parseEntries(content []byte) []entry {
	lines := strings.Split(string(content), "\n")
	entries := make([]entry, 0, len(lines))
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		eq := strings.IndexByte(trimmed, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:eq])
		if key == "" {
			continue
		}
		value := unquoteValue(strings.TrimSpace(trimmed[eq+1:]))
		entries = append(entries, entry{key: key, value: value, line: i + 1, raw: line})
	}
	return entries
}

// unquoteValue removes one layer of surrounding double quotes and resolves the
// backslash escapes Ghostty accepts inside them.
func unquoteValue(value string) string {
	if len(value) < 2 || value[0] != '"' || !strings.HasSuffix(value, `"`) {
		return value
	}
	inner := value[1 : len(value)-1]
	if !strings.ContainsRune(inner, '\\') {
		return inner
	}
	var b strings.Builder
	b.Grow(len(inner))
	escaped := false
	for _, r := range inner {
		switch {
		case escaped:
			b.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		default:
			b.WriteRune(r)
		}
	}
	if escaped {
		b.WriteRune('\\')
	}
	return b.String()
}

// includes returns the config-file directives of a file, in order.
func (f *configFile) includes() []include {
	var out []include
	for _, e := range f.entries {
		if e.key != "config-file" || e.value == "" {
			continue
		}
		inc := include{raw: e.value, line: e.line}
		ref := strings.TrimSpace(e.value)
		if strings.HasPrefix(ref, "?") {
			inc.optional = true
			ref = strings.TrimPrefix(ref, "?")
		}
		ref = unquoteValue(strings.TrimSpace(ref))
		if ref == "" {
			continue
		}
		inc.path = ref
		out = append(out, inc)
	}
	return out
}

// readConfigFile reads and parses one configuration file. Symlinks are resolved
// for later writes while the referenced path stays authoritative for includes.
func readConfigFile(path string) (*configFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	real := path
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		real = resolved
	}
	info, err := os.Stat(real)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(content)
	return &configFile{
		path:    path,
		real:    real,
		dir:     filepath.Dir(path),
		content: content,
		mode:    info.Mode().Perm(),
		hash:    hex.EncodeToString(sum[:]),
		entries: parseEntries(content),
	}, nil
}

// sourceGraph is the resolved include graph of the user's configuration.
type sourceGraph struct {
	files []*configFile
	// missing lists config-file directives whose target does not exist.
	missing []missingInclude
	// shadowed lists configuration files that exist but are not read because a
	// more specific name exists in the same directory.
	shadowed []string
	// unsafe records a graph shape the studio refuses to rewrite.
	unsafe string
}

// resolveGraph walks the configuration roots in load order, following
// config-file directives depth first. The resulting file order is the order
// Ghostty loads them in: a file, then its includes in order.
func resolveGraph(roots []string) (*sourceGraph, error) {
	g := &sourceGraph{}
	visited := make(map[string]bool)
	var walk func(path string, optional bool, from string) error
	walk = func(path string, optional bool, from string) error {
		key := canonicalKey(path)
		if visited[key] {
			if g.unsafe == "" {
				g.unsafe = fmt.Sprintf("include cycle detected at %s", path)
			}
			return nil
		}
		file, err := readConfigFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				g.missing = append(g.missing, missingInclude{path: path, from: from, optional: optional})
				return nil
			}
			return fmt.Errorf("reading %s: %w", path, err)
		}
		visited[key] = true
		g.files = append(g.files, file)
		for _, inc := range file.includes() {
			target := inc.path
			if !filepath.IsAbs(target) {
				// Relative includes resolve against the file that declares them,
				// so staging must never move a file to another directory.
				target = filepath.Join(file.dir, inc.path)
			}
			if err := walk(target, inc.optional, file.path); err != nil {
				return err
			}
		}
		return nil
	}
	for _, root := range roots {
		if err := walk(root, false, ""); err != nil {
			return nil, err
		}
	}
	return g, nil
}

// canonicalKey identifies a file for cycle detection.
func canonicalKey(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return filepath.Clean(path)
}

// paths returns the contributing files in load order.
func (g *sourceGraph) paths() []string {
	out := make([]string, 0, len(g.files))
	for _, f := range g.files {
		out = append(out, f.path)
	}
	return out
}

// find returns the parsed file for path.
func (g *sourceGraph) find(path string) *configFile {
	for _, f := range g.files {
		if f.path == path || f.real == path {
			return f
		}
	}
	return nil
}

// last returns the last file in load order, which is the winning point for
// scalar keys and the anchor for the managed include.
func (g *sourceGraph) last() *configFile {
	if len(g.files) == 0 {
		return nil
	}
	return g.files[len(g.files)-1]
}

// lastExcluding returns the last file in load order that is not any of the
// excluded paths. The managed include belongs at the end of that file.
func (g *sourceGraph) lastExcluding(excluded ...string) *configFile {
	skip := make(map[string]bool, len(excluded))
	for _, path := range excluded {
		skip[canonicalKey(path)] = true
	}
	for i := len(g.files) - 1; i >= 0; i-- {
		if !skip[canonicalKey(g.files[i].path)] {
			return g.files[i]
		}
	}
	return nil
}

// revision hashes the whole graph: file identity plus content, in load order.
func (g *sourceGraph) revision() string {
	h := sha256.New()
	for _, f := range g.files {
		fmt.Fprintf(h, "file\x00%s\x00%s\x00", f.real, f.hash)
	}
	for _, m := range g.missing {
		fmt.Fprintf(h, "missing\x00%s\x00%s\x00%t\x00", m.from, m.path, m.optional)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// managedIncludeLine is the directive that pulls the studio's managed file into
// the user's configuration graph. The '?' prefix keeps a manually deleted
// managed file from breaking the user's configuration.
func managedIncludeLine(path string) string {
	return "config-file = ?" + path
}

// hasManagedInclude reports whether the file already carries the directive.
func (f *configFile) hasManagedInclude(managedPath string) bool {
	target := canonicalKey(managedPath)
	for _, inc := range f.includes() {
		if canonicalKey(inc.path) == target {
			return true
		}
	}
	return false
}

// withManagedInclude returns file content with the managed include appended at
// the end of the file, which is the last point Ghostty reads.
func withManagedInclude(content []byte, managedPath string) []byte {
	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		content = append(content, '\n')
	}
	block := "# Ghostty Studio managed include. Edits here are rewritten; the values live in:\n" +
		"# " + managedPath + "\n" +
		managedIncludeLine(managedPath) + "\n"
	return append(content, block...)
}

// managedCommentHead is the first line of the block the studio writes.
const managedCommentHead = "# Ghostty Studio managed include. Edits here are rewritten; the values live in:"

// isManagedIncludeLine reports whether a raw line is one of the three lines the
// studio owns in the user's configuration file.
func isManagedIncludeLine(line, managedPath string) bool {
	trimmed := strings.TrimSpace(line)
	switch trimmed {
	case managedCommentHead, "# " + managedPath, managedIncludeLine(managedPath):
		return true
	}
	return false
}

// ensureManagedInclude rewrites a configuration file so the managed include
// appears exactly once, as the last thing Ghostty reads. It reports whether the
// content changed, so an untouched file is never rewritten.
func ensureManagedInclude(content []byte, managedPath string) ([]byte, bool) {
	filtered := stripManagedInclude(content, managedPath)
	updated := withManagedInclude(filtered, managedPath)
	if string(updated) == string(content) {
		return content, false
	}
	return updated, true
}

// stripManagedInclude removes the managed include block without appending it
// again, which is what undoing the very first apply needs.
func stripManagedInclude(content []byte, managedPath string) []byte {
	var kept []string
	for _, line := range strings.Split(string(content), "\n") {
		if isManagedIncludeLine(line, managedPath) {
			continue
		}
		kept = append(kept, line)
	}
	joined := strings.Join(kept, "\n")
	trimmed := strings.TrimRight(joined, "\n")
	if trimmed == "" {
		return nil
	}
	return []byte(trimmed + "\n")
}
