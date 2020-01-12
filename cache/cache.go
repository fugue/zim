package cache

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/LuminalHQ/zim/project"
	"github.com/LuminalHQ/zim/store"
)

const WriteOnly = "WRITE_ONLY"

// Error is used to handle cache misses and the like
type Error string

func (e Error) Error() string { return string(e) }

// CacheMiss indicates the cache did not contain a match
const CacheMiss = Error("Item not found in cache")

// Entry carries the name and hash for one item within a Key
type Entry struct {
	Name string `json:"name"`
	Hash string `json:"hash"`
}

func newEntry(name, hash string) *Entry {
	return &Entry{Name: name, Hash: hash}
}

// Key contains information used to build a key
type Key struct {
	Project     string   `json:"project"`
	Component   string   `json:"component"`
	Rule        string   `json:"rule"`
	Image       string   `json:"image"`
	OutputCount int      `json:"output_count"`
	Inputs      []*Entry `json:"inputs"`
	Deps        []*Entry `json:"deps"`
	Env         []*Entry `json:"env"`
	Toolchain   []*Entry `json:"toolchain"`
	Version     string   `json:"version"`
	Commands    []string `json:"commands"`
	hex         string
}

// String returns the key as a hexadecimal string
func (k *Key) String() string {
	return k.hex
}

// Compute determines the hash for this key
func (k *Key) Compute() error {
	h := sha1.New()
	if err := json.NewEncoder(h).Encode(k); err != nil {
		return err
	}
	k.hex = hex.EncodeToString(h.Sum(nil))
	return nil
}

// NewMiddleware returns caching middleware
func NewMiddleware(s store.Store, user, mode string) project.RunnerBuilder {

	c := New(s)
	c.user = user
	c.mode = mode

	return project.RunnerBuilder(func(runner project.Runner) project.Runner {
		return project.RunnerFunc(func(ctx context.Context, r *project.Rule, opts project.RunOpts) (project.Code, error) {

			// Caching is only applicable for rules that have cacheable
			// outputs. If this is not the case, run the rule normally.
			outputs := r.Outputs()
			if len(outputs) == 0 || !outputs[0].Cacheable() {
				return runner.Run(ctx, r, opts)
			}

			if mode != WriteOnly {
				// Download matching outputs from the cache if they exist
				_, err := c.Read(ctx, r)
				if err == nil {
					return project.Cached, nil // Cache hit
				}
				if err != CacheMiss {
					return project.Error, err // Cache error
				}
			}

			// At this point, the outputs were not cached so build the rule
			code, err := runner.Run(ctx, r, opts)

			// Code "OK" indicates the rule was built which means we can
			// store its outputs in the cache
			if code == project.OK {
				if _, err := c.Write(ctx, r); err != nil {
					return project.Error, err
				}
			}
			return code, err
		})
	})
}

// Cache used to determine rule keys
type Cache struct {
	store store.Store
	user  string
	mode  string
}

// New returns a cache
func New(s store.Store) *Cache {
	return &Cache{store: s}
}

// Write rule outputs to the cache
func (c *Cache) Write(ctx context.Context, r *project.Rule) ([]string, error) {

	outputs := r.Outputs().Paths()

	// If the rule has no outputs then there is nothing to cache
	if len(outputs) == 0 {
		return nil, fmt.Errorf("Rule has no outputs: %s", r.NodeID())
	}

	key, err := c.Key(r)
	if err != nil {
		return nil, err
	}
	storageKey := path.Join("cache", key.String())

	var storagePaths []string
	if len(outputs) == 1 {
		if err := c.put(ctx, storageKey, outputs[0]); err != nil {
			return nil, err
		}
		storagePaths = append(storagePaths, storageKey)
	} else {
		for i, out := range outputs {
			storageKeyOfs := fmt.Sprintf("%s-%d", storageKey, i)
			if err := c.put(ctx, storageKeyOfs, out); err != nil {
				return nil, err
			}
			storagePaths = append(storagePaths, storageKeyOfs)
		}
	}

	// The above would be sufficient for the cache to operate.
	// Let's also upload some metadata for now to sanity check results.
	keyPath, err := writeJSON(key)
	if err != nil {
		return nil, err
	}
	defer os.Remove(keyPath)

	infoKey := path.Join("cache", fmt.Sprintf("%s.json", key.String()))
	if err := c.put(ctx, infoKey, keyPath); err != nil {
		return nil, err
	}

	return storagePaths, nil
}

// Read rule outputs from the cache
func (c *Cache) Read(ctx context.Context, r *project.Rule) ([]string, error) {

	outputs := r.Outputs().Paths()

	// If the rule has no outputs then there is nothing to read from the cache
	if len(outputs) == 0 {
		return nil, fmt.Errorf("Rule has no outputs: %s", r.NodeID())
	}

	key, err := c.Key(r)
	if err != nil {
		return nil, err
	}
	storageKey := path.Join("cache", key.String())

	var storagePaths []string
	if len(outputs) == 1 {
		if err := c.get(ctx, storageKey, outputs[0]); err != nil {
			return nil, err
		}
		storagePaths = append(storagePaths, storageKey)
	} else {
		for i, out := range outputs {
			storageKeyOfs := fmt.Sprintf("%s-%d", storageKey, i)
			if err := c.get(ctx, storageKeyOfs, out); err != nil {
				return nil, err
			}
			storagePaths = append(storagePaths, storageKeyOfs)
		}
	}
	return storagePaths, nil
}

func (c *Cache) put(ctx context.Context, key, src string) error {

	// The file hash will be added to the cache item metadata
	hash, err := HashFile(src)
	if err != nil {
		return err
	}
	meta := map[string]string{
		"Hash": hash,
		"User": c.user,
	}

	// Store the file in the cache
	return c.store.Put(ctx, key, src, meta)
}

func (c *Cache) get(ctx context.Context, key, dst string) error {

	// Determine if the cache contains an item for the key
	remoteInfo, err := c.store.Head(ctx, key)
	if err != nil {
		if _, ok := err.(store.NotFound); ok {
			return CacheMiss
		}
		return err
	}
	remoteHash := remoteInfo.Meta["Hash"]

	// If a local file exists that is identical to the one in the cache,
	// then there is nothing to do
	if localHash, err := HashFile(dst); err == nil {
		if remoteHash == localHash {
			return nil
		}
	}
	// Download the file from the cache
	return c.store.Get(ctx, key, dst)
}

// Key returns a struct of information that uniquely identifies the Rule's
// inputs and configuration. This used to store Rule outputs in the cache.
func (c *Cache) Key(r *project.Rule) (*Key, error) {
	// Previously had locking and an internal cache of keys that were
	// already computed. Removed for now to focus on correctness first.
	return c.buildKey(r)
}

// Internal function that populates the Key data structure for a rule.
// This call must remain safe for concurrent calls from multiple goroutines!
func (c *Cache) buildKey(r *project.Rule) (*Key, error) {

	inputs, err := r.Inputs()
	if err != nil {
		return nil, err
	}
	env, err := r.Environment()
	if err != nil {
		return nil, err
	}
	toolchain, err := r.Component().Toolchain()
	if err != nil {
		return nil, err
	}

	root := r.Project().RootAbsPath()
	deps := r.Dependencies()
	version := "0.0.4"

	key := &Key{
		Project:     r.Project().Name(),
		Component:   r.Component().Name(),
		Rule:        r.Name(),
		Image:       r.Image(),
		Inputs:      make([]*Entry, 0, len(inputs)),
		Deps:        make([]*Entry, 0, len(deps)),
		Env:         make([]*Entry, 0, len(env)),
		Toolchain:   make([]*Entry, 0, len(toolchain)),
		OutputCount: len(r.Outputs()),
		Commands:    make([]string, 0, len(r.Commands())),
		Version:     version,
	}

	// Include the hash of every input file in the key
	for _, input := range inputs.Paths() {
		hash, err := c.hashFile(input)
		if err != nil {
			return nil, err
		}
		// Use relative paths for key stability on diff machines
		relInput, err := filepath.Rel(root, input)
		if err != nil {
			return nil, err
		}
		key.Inputs = append(key.Inputs, newEntry(relInput, hash))
	}

	// Include rule environment variables in the key
	for _, k := range MapKeys(env) {
		hash, err := c.hashString(env[k])
		if err != nil {
			return nil, err
		}
		key.Env = append(key.Env, newEntry(k, hash))
	}

	// Include the key of every dependency in this key
	for _, dep := range deps {
		depKey, err := c.buildKey(dep)
		if err != nil {
			return nil, err
		}
		key.Deps = append(key.Deps, newEntry(dep.NodeID(), depKey.String()))
	}

	// Capture toolchain information so that different compilers will
	// result in different keys (for example)
	for _, k := range MapKeys(toolchain) {
		key.Toolchain = append(key.Toolchain, newEntry(k, toolchain[k]))
	}

	// Include the commands used to build the rule in the key
	for _, cmd := range r.Commands() {
		key.Commands = append(key.Commands, cmd)
	}

	// Determine the hex string for this key
	if err := key.Compute(); err != nil {
		return nil, err
	}
	return key, nil
}

// hashFile returns the SHA1 hash of a given file
func (c *Cache) hashFile(p string) (string, error) {
	// No caching for now
	return HashFile(p)
}

// hashString returns the SHA1 hash of a given string
func (c *Cache) hashString(s string) (string, error) {
	// No caching for now
	return HashString(s)
}

// HashFile returns the SHA1 hash of file contents
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

// HashString returns the SHA1 hash of a string
func HashString(s string) (string, error) {
	h := sha1.New()
	if _, err := h.Write([]byte(s)); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// MapKeys returns a sorted slice containing all keys from the given map
func MapKeys(m map[string]string) (result []string) {
	result = make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	sort.Strings(result)
	return
}

func writeJSON(key *Key) (string, error) {
	js, err := json.Marshal(key)
	if err != nil {
		return "", err
	}
	f, err := ioutil.TempFile("", "zim-key-")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(js); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func writeKey(path string, key *Key) error {
	js, err := json.Marshal(key)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(js)
	return err
}
