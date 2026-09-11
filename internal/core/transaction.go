package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// journalFile records one file touched by a transaction.
type journalFile struct {
	// Path is the resolved write target.
	Path string `json:"path"`
	// Display is the path the user recognises.
	Display string `json:"display"`
	// Backup is the copy made before writing, empty when the file was created.
	Backup string `json:"backup,omitempty"`
	// HashBefore and HashAfter are sha256 hex digests of the file bytes.
	HashBefore string `json:"hash_before"`
	HashAfter  string `json:"hash_after"`
	// Created marks a file that did not exist before the transaction.
	Created bool `json:"created"`
	// Mode is the original permission bits, restored on undo.
	Mode uint32 `json:"mode"`
}

// journalEntry records one applied transaction.
type journalEntry struct {
	ID             string        `json:"id"`
	Label          string        `json:"label"`
	Time           string        `json:"time"`
	RevisionBefore string        `json:"revision_before"`
	RevisionAfter  string        `json:"revision_after"`
	Files          []journalFile `json:"files"`
}

// journal is the append-only undo stack.
type journal struct {
	Entries []journalEntry `json:"entries"`
}

// journalPath is the undo stack location inside the state directory.
func (e *Engine) journalPath() string {
	return filepath.Join(e.paths.stateDir, "journal.json")
}

// backupDir is where byte-identical pre-write copies live.
func (e *Engine) backupDir(txnID string) string {
	return filepath.Join(e.paths.stateDir, "backups", txnID)
}

// ensureDir creates a directory with a private mode.
func ensureDir(path string) error {
	if path == "" {
		return errors.New("no directory configured")
	}
	return os.MkdirAll(path, 0o700)
}

// writeFileAtomic writes data to path through a same-directory temporary file
// and a rename, so a reader never observes a partial file.
func writeFileAtomic(path string, data []byte, mode fs.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		// The rename removes the temporary file; this only fires on failure.
		if _, err := os.Stat(tmpName); err == nil {
			os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// hashFile returns the sha256 of a file's bytes.
func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// lock takes an exclusive lock on the state directory. Concurrent transactions
// are serialised; the lock is released when the returned function is called.
func (e *Engine) lock() (func(), error) {
	if err := ensureDir(e.paths.stateDir); err != nil {
		return nil, fmt.Errorf("state directory %s: %w", e.paths.stateDir, err)
	}
	file, err := os.OpenFile(filepath.Join(e.paths.stateDir, "lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		file.Close()
		return nil, fmt.Errorf("locking the state directory: %w", err)
	}
	return func() {
		syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		file.Close()
	}, nil
}

// readJournal loads the undo stack. A missing journal is an empty stack.
func (e *Engine) readJournal() (journal, error) {
	var j journal
	data, err := os.ReadFile(e.journalPath())
	if err != nil {
		if os.IsNotExist(err) {
			return j, nil
		}
		return j, err
	}
	if err := json.Unmarshal(data, &j); err != nil {
		return j, fmt.Errorf("undo journal %s is unreadable: %w", e.journalPath(), err)
	}
	return j, nil
}

// writeJournal persists the undo stack atomically.
func (e *Engine) writeJournal(j journal) error {
	if err := ensureDir(e.paths.stateDir); err != nil {
		return err
	}
	if len(j.Entries) == 0 {
		if err := os.Remove(e.journalPath()); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	data, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(e.journalPath(), append(data, '\n'), 0o600)
}

// transaction accumulates the writes of one apply so they can be journalled and
// undone as a unit.
type transaction struct {
	id      string
	label   string
	files   []journalFile
	backups []string
}

// stage records the pre-write state of a file.
func (t *transaction) stage(e *Engine, path string) error {
	if path == "" {
		return errors.New("empty write target")
	}
	real := path
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		real = resolved
	}
	entry := journalFile{Path: real, Display: path, Mode: 0o644}
	for _, existing := range t.files {
		if existing.Path == real {
			return nil
		}
	}
	if info, err := os.Stat(real); err == nil {
		entry.Mode = uint32(info.Mode().Perm())
		hash, err := hashFile(real)
		if err != nil {
			return err
		}
		entry.HashBefore = hash
	} else if os.IsNotExist(err) {
		entry.Created = true
	} else {
		return err
	}
	t.files = append(t.files, entry)
	return nil
}

// backup copies the current bytes of every staged file into the transaction's
// backup directory. It must run before any write.
func (t *transaction) backup(e *Engine) error {
	if len(t.files) == 0 {
		return nil
	}
	dir := e.backupDir(t.id)
	if err := ensureDir(dir); err != nil {
		return err
	}
	for i := range t.files {
		file := &t.files[i]
		if file.Created {
			continue
		}
		data, err := os.ReadFile(file.Path)
		if err != nil {
			return fmt.Errorf("backing up %s: %w", file.Path, err)
		}
		name := fmt.Sprintf("%02d-%s", i, filepath.Base(file.Path))
		target := filepath.Join(dir, name)
		if err := writeFileAtomic(target, data, 0o600); err != nil {
			return fmt.Errorf("backing up %s: %w", file.Path, err)
		}
		file.Backup = target
		t.backups = append(t.backups, target)
	}
	return nil
}

// commit records the post-write hashes and appends the transaction to the undo
// stack.
func (t *transaction) commit(e *Engine, revisionBefore, revisionAfter string) error {
	for i := range t.files {
		file := &t.files[i]
		if _, err := os.Stat(file.Path); err != nil {
			if os.IsNotExist(err) {
				file.HashAfter = ""
				continue
			}
			return err
		}
		hash, err := hashFile(file.Path)
		if err != nil {
			return err
		}
		file.HashAfter = hash
	}
	j, err := e.readJournal()
	if err != nil {
		return err
	}
	j.Entries = append(j.Entries, journalEntry{
		ID:             t.id,
		Label:          t.label,
		Time:           time.Now().Format(time.RFC3339),
		RevisionBefore: revisionBefore,
		RevisionAfter:  revisionAfter,
		Files:          t.files,
	})
	return e.writeJournal(j)
}

// backupPathFor reports the most useful backup path to surface to the user.
func (t *transaction) backupPathFor() string {
	if len(t.backups) > 0 {
		return filepath.Dir(t.backups[0])
	}
	return ""
}

// newTransactionID builds a sortable, unique transaction identifier.
func newTransactionID() string {
	return fmt.Sprintf("%s-%d", time.Now().UTC().Format("20060102T150405"), os.Getpid())
}

// undoLast reverses the most recent transaction. It refuses to touch a file
// that changed after the transaction, so an external edit is never clobbered.
func (e *Engine) undoLast() (Result, error) {
	unlock, err := e.lock()
	if err != nil {
		return Result{}, err
	}
	defer unlock()

	j, err := e.readJournal()
	if err != nil {
		return Result{}, err
	}
	if len(j.Entries) == 0 {
		return Result{}, errors.New("there is nothing to undo: Ghostty Studio has not applied a change yet")
	}
	entry := j.Entries[len(j.Entries)-1]
	// The whole source graph must be exactly as the change left it. A file the
	// studio never wrote, such as an include owned by the user, still counts:
	// undoing while the graph has drifted would restore a state that never
	// existed.
	if entry.RevisionAfter != "" {
		graph, err := resolveGraph(e.roots)
		if err != nil {
			return Result{}, fmt.Errorf("cannot undo: re-reading the configuration graph failed: %w", err)
		}
		if current := graph.revision(); current != entry.RevisionAfter {
			return Result{}, fmt.Errorf("cannot undo: the configuration graph changed after this change was applied (%s -> %s), so undoing would discard that edit", shortHash(entry.RevisionAfter), shortHash(current))
		}
	}
	for _, file := range entry.Files {
		current, err := hashFile(file.Path)
		switch {
		case os.IsNotExist(err) && file.HashAfter == "":
			continue
		case err != nil:
			return Result{}, fmt.Errorf("cannot undo: %s is unreadable (%v)", file.Display, err)
		case current != file.HashAfter:
			return Result{}, fmt.Errorf("cannot undo: %s changed after Ghostty Studio applied the change, so undoing would discard that edit", file.Display)
		}
	}
	restored := 0
	for _, file := range entry.Files {
		switch {
		case file.Created:
			if err := os.Remove(file.Path); err != nil && !os.IsNotExist(err) {
				return Result{}, fmt.Errorf("cannot undo: removing %s failed: %w", file.Display, err)
			}
		case file.Backup == "":
			return Result{}, fmt.Errorf("cannot undo: no backup exists for %s", file.Display)
		default:
			data, err := os.ReadFile(file.Backup)
			if err != nil {
				return Result{}, fmt.Errorf("cannot undo: reading the backup for %s failed: %w", file.Display, err)
			}
			if err := writeFileAtomic(file.Path, data, fs.FileMode(file.Mode)); err != nil {
				return Result{}, fmt.Errorf("cannot undo: restoring %s failed: %w", file.Display, err)
			}
		}
		restored++
	}
	j.Entries = j.Entries[:len(j.Entries)-1]
	if err := e.writeJournal(j); err != nil {
		return Result{}, err
	}
	label := entry.Label
	if label == "" {
		label = "the last change"
	}
	result := Result{
		Message: fmt.Sprintf("Undid %s. %s restored byte-identically.", label, plural(restored, "1 file", fmt.Sprintf("%d files", restored))),
	}
	return result, nil
}

// fileStatus describes one file's relation to the undo stack, used by health
// reporting.
type fileStatus struct {
	path    string
	changed bool
}

// changedSince reports whether any of a transaction's files changed after it was
// applied.
func changedSince(files []journalFile) []string {
	var out []string
	for _, file := range files {
		current, err := hashFile(file.Path)
		if err != nil || current != file.HashAfter {
			out = append(out, file.Display)
		}
	}
	return out
}

// backupGuidance describes where backups live, for health output.
func (e *Engine) backupGuidance() string {
	return filepath.Join(e.paths.stateDir, "backups") + " holds one copy of every file Ghostty Studio rewrote, and " + e.journalPath() + " records the undo stack"
}

// managedState reads the current managed values from disk, if the managed file
// exists.
func (e *Engine) managedState() Values {
	path := e.managedFile
	if path == "" {
		return Values{}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return Values{}
	}
	values := Values{}
	for _, entry := range parseEntries(content) {
		values[entry.key] = append(values[entry.key], entry.value)
	}
	return values
}

// renderManagedFile turns managed values into the managed include file content.
func renderManagedFile(values Values, revision string) []byte {
	var b strings.Builder
	b.WriteString("# Written by Ghostty Studio. Edits are overwritten on the next apply.\n")
	if revision != "" {
		fmt.Fprintf(&b, "# revision: %s\n", revision)
	}
	for _, key := range values.Keys() {
		list := values[key]
		if len(list) == 0 {
			continue
		}
		for _, value := range list {
			b.WriteString(renderDirective(key, value))
			b.WriteByte('\n')
		}
	}
	return []byte(b.String())
}

// renderDirective renders one configuration line, quoting when a value could
// otherwise be misread.
func renderDirective(key, value string) string {
	if value == "" {
		return key + ` = ""`
	}
	if needsQuoting(value) {
		escaped := strings.ReplaceAll(value, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		return key + ` = "` + escaped + `"`
	}
	return key + " = " + value
}

// needsQuoting reports whether a value must be quoted to survive parsing.
func needsQuoting(value string) bool {
	if strings.HasPrefix(value, "?") || strings.HasPrefix(value, `"`) {
		return true
	}
	if value != strings.TrimSpace(value) {
		return true
	}
	// A '#' only starts a comment at the beginning of a line, so colors and
	// other values need no quoting; leading '?' and '"' and surrounding
	// whitespace would change meaning, so those are quoted.
	return false
}
