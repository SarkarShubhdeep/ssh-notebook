// Package content provides read access to the notebook's Markdown files.
// In v0.1 it is read-only; writes are added in v0.3.
package content

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Store is a filesystem-backed notebook rooted at Root.
type Store struct {
	Root string
}

// New returns a Store rooted at the given directory.
func New(root string) *Store {
	return &Store{Root: root}
}

// List returns the paths of all Markdown files under Root, relative to Root,
// sorted lexicographically.
func (s *Store) List() ([]string, error) {
	var files []string
	err := filepath.WalkDir(s.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".md") {
			rel, relErr := filepath.Rel(s.Root, path)
			if relErr != nil {
				return relErr
			}
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// Read returns the contents of a Markdown file identified by a path relative to
// Root. It refuses to read outside the notebook root (path traversal guard).
func (s *Store) Read(rel string) (string, error) {
	full, err := s.resolve(rel)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// resolve turns a relative notebook path into an absolute filesystem path,
// guaranteeing the result stays within Root.
func (s *Store) resolve(rel string) (string, error) {
	// Clean against a virtual root so any leading "../" is neutralised.
	clean := filepath.Clean(string(filepath.Separator) + rel)
	full := filepath.Join(s.Root, clean)

	rootAbs, err := filepath.Abs(s.Root)
	if err != nil {
		return "", err
	}
	fullAbs, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	if fullAbs != rootAbs && !strings.HasPrefix(fullAbs, rootAbs+string(filepath.Separator)) {
		return "", os.ErrPermission
	}
	return fullAbs, nil
}
