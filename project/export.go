package project

import (
	"sync"
)

// Export defines resources exposed by a Component. The resources referenced by
// an export must be static, which allows the export to be resolved only once.
type Export struct {
	Component         *Component
	Provider          Provider
	Resources         []string
	Ignore            []string
	mutex             sync.Mutex
	resolvedResources Resources
	resolvedError     error
	resolved          bool
}

// Resolve the specific resources that this export exposes. This often takes
// a glob-like pattern and finds the corresponding specific list of files.
// This will be called from multiple goroutines.
func (e *Export) Resolve() (Resources, error) {

	e.mutex.Lock()
	defer e.mutex.Unlock()

	// If this export was already resolved to specific resources, then just
	// return the results we previously saved.
	if e.resolved {
		return e.resolvedResources, e.resolvedError
	}

	// Discover exported resources
	matches, err := e.Provider.Match(e.Resources, nil)
	if err != nil {
		e.resolved = true
		e.resolvedError = err
		return nil, err
	}

	// Exclude ignored resources
	ignored, err := e.Provider.Match(e.Ignore, nil)
	if err != nil {
		e.resolved = true
		e.resolvedError = err
		return nil, err
	}

	addedPaths := map[string]bool{}
	ignoredPaths := map[string]bool{}
	var resources []Resource

	for _, r := range ignored {
		ignoredPaths[r.Path()] = true
	}

	for _, r := range matches {
		rPath := r.Path()
		if !addedPaths[rPath] && !ignoredPaths[rPath] {
			resources = append(resources, r)
			addedPaths[rPath] = true
		}
	}

	// Save the resources for future calls
	e.resolvedResources = resources
	e.resolvedError = nil
	e.resolved = true
	return resources, nil
}
