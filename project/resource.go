package project

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
	"time"
)

// Resource is in an interface representing an artifact created by a rule
type Resource interface {

	// Name of the Resource
	Name() string

	// Path to the output
	Path() string

	// Exists indicates whether the Resource is present
	Exists() (bool, error)

	// Hash of this Resource
	Hash() (string, error)

	// LastModified time of this Resource
	LastModified() (time.Time, error)

	// OnFilesystem returns true if it is backed by a file on disk
	OnFilesystem() bool

	// Cacheable indicates whether the resource can be stored in a cache
	Cacheable() bool

	// AsFile returns the path to a file containing the Resource itself, or
	// a representation of the Resource
	AsFile() (string, error)
}

// HashFile returns the SHA1 hash of File contents
func HashFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	h := sha1.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// LastModified returns the most recent modification time for the given Resources
func LastModified(outputs []Resource) (time.Time, error) {
	var t time.Time
	for _, out := range outputs {
		outTime, err := out.LastModified()
		if err != nil {
			return t, err
		}
		if outTime.After(t) {
			t = outTime
		}
	}
	return t, nil
}
