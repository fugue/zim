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

	// The exported resource paths are relative to their Component.
	// Prepend the Component path to the path patterns.
	cDir := e.Component.RelPath()

	// Discover exported resources
	resources, err := e.Provider.Match(
		joinPaths(cDir, e.Resources),
		joinPaths(cDir, e.Ignore))

	e.resolved = true
	if err != nil {
		e.resolvedResources = nil
		e.resolvedError = err
		return nil, err
	}
	// Cache the resources for future calls
	e.resolvedResources = resources
	e.resolvedError = nil
	return resources, nil
}
