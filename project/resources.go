package project

import (
	"path/filepath"
	"time"
)

// Resources is shorthand for a slice of Resources
type Resources []Resource

// Paths of all the Resources
func (rs Resources) Paths() (paths []string) {
	paths = make([]string, 0, len(rs))
	for _, f := range rs {
		paths = append(paths, f.Path())
	}
	return
}

// RelativePaths returns paths to these Resources relative to the given directory
func (rs Resources) RelativePaths(base string) (paths []string, err error) {
	paths = make([]string, 0, len(rs))
	for _, r := range rs {
		if !r.OnFilesystem() {
			paths = append(paths, r.Path())
			continue
		}
		rel, err := filepath.Rel(base, r.Path())
		if err != nil {
			return nil, err
		}
		paths = append(paths, rel)
	}
	return
}

// LastModified returns the most recent modification of all these Resources
func (rs Resources) LastModified() (t time.Time, err error) {
	for _, f := range rs {
		mod, err := f.LastModified()
		if err != nil {
			return t, err
		}
		if mod.After(t) {
			t = mod
		}
	}
	return
}
