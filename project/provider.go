package project

// Provider describes an interface for managing Resources
type Provider interface {

	// Init gives options to the Provider
	Init(options map[string]interface{}) error

	// Name identifies the Provider type
	Name() string

	// New creates a Resource
	New(path string) Resource

	// Match Resources according to the given pattern
	Match(pattern string) (Resources, error)
}
