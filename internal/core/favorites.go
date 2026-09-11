package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// favoritesFile is the persisted favorites document.
type favoritesFile struct {
	Format    string   `json:"format"`
	Favorites []string `json:"favorites"`
}

// favoritesPath locates the favorites document.
func (e *Engine) favoritesPath() string {
	return filepath.Join(e.paths.stateDir, "favorites.json")
}

// readFavorites loads the favorites document. A missing file is an empty list:
// reading never creates state.
func (e *Engine) readFavorites() ([]string, error) {
	data, err := os.ReadFile(e.favoritesPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var doc favoritesFile
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s is unreadable: %w", e.favoritesPath(), err)
	}
	ids := append([]string(nil), doc.Favorites...)
	sort.Strings(ids)
	return ids, nil
}

// writeFavorites persists the favorites document.
func (e *Engine) writeFavorites(ids []string) error {
	if err := ensureDir(e.paths.stateDir); err != nil {
		return err
	}
	if ids == nil {
		ids = []string{}
	}
	data, err := json.MarshalIndent(favoritesFile{Format: lookFormat, Favorites: ids}, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(e.favoritesPath(), append(data, '\n'), 0o600)
}
