package hash

// Hasher is an interface for hashing objects, files, or strings.
// Different implementations may exist for SHA1, SHA256, etc.
type Hasher interface {

	// Object returns the hash of a given object
	Object(obj interface{}) (string, error)

	// File returns the hash of a given file on disk
	File(path string) (string, error)

	// String returns the hash of a given string
	String(s string) (string, error)
}
