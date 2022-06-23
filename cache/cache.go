// Copyright 2020 Fugue, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cache

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/fugue/zim/hash"
	"github.com/fugue/zim/project"
	"github.com/fugue/zim/store"
)

const (
	// ReadWrite is the default cache mode
	ReadWrite = "read-write"

	// WriteOnly mode is used to write to the cache but not read from it
	WriteOnly = "write-only"

	// Disabled mode bypasses all cache interactions
	Disabled = "disabled"
)

// Error is used to handle cache misses and the like
type Error string

func (e Error) Error() string { return string(e) }

// CacheMiss indicates the cache did not contain a match
const CacheMiss = Error("Item not found in cache")

// Opts defines options for initializing a Cache
type Opts struct {
	Store  store.Store
	Hasher hash.Hasher
	User   string
	Mode   string
}

// Cache for rule outputs
type Cache struct {
	store  store.Store
	hasher hash.Hasher
	user   string
	mode   string
}

// New returns a Cache
func New(opts Opts) *Cache {

	if opts.Hasher == nil {
		opts.Hasher = hash.SHA1()
	}

	c := &Cache{
		store:  opts.Store,
		hasher: opts.Hasher,
		user:   opts.User,
		mode:   opts.Mode,
	}
	return c
}

// Write rule outputs to the cache
func (c *Cache) Write(ctx context.Context, r *project.Rule) ([]string, error) {

	outputs := r.Outputs().Paths()

	// If the rule has no outputs then there is nothing to cache
	if len(outputs) == 0 {
		return nil, fmt.Errorf("rule has no outputs: %s", r.NodeID())
	}

	key, err := c.Key(ctx, r)
	if err != nil {
		return nil, err
	}
	storageKey := key.String()

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

	infoKey := fmt.Sprintf("%s.json", key.String())
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
		return nil, fmt.Errorf("rule has no outputs: %s", r.NodeID())
	}

	key, err := c.Key(ctx, r)
	if err != nil {
		return nil, err
	}
	storageKey := key.String()

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
	hash, err := c.hasher.File(src)
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
	if localHash, err := c.hasher.File(dst); err == nil {
		if remoteHash == localHash {
			return nil
		}
	}

	// Download the file from the cache
	return c.store.Get(ctx, key, dst)
}

// Key returns a struct of information that uniquely identifies the Rule's
// inputs and configuration. This used to store Rule outputs in the cache.
func (c *Cache) Key(ctx context.Context, r *project.Rule) (*Key, error) {
	// Previously had locking and an internal cache of keys that were
	// already computed. Removed for now to focus on correctness first.
	return c.buildKey(ctx, r)
}

// Internal function that populates the Key data structure for a rule.
// This call must remain safe for concurrent calls from multiple goroutines!
func (c *Cache) buildKey(ctx context.Context, r *project.Rule) (*Key, error) {

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

	// Bail early if the context was canceled. The toolchain step above
	// sometimes takes a bit so this is a good spot to check the context.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	root := r.Project().RootAbsPath()
	deps := r.Dependencies()
	cmds := r.Commands()
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
		Commands:    make([]string, 0, len(r.Commands())),
		OutputCount: len(r.Outputs()),
		Version:     version,
		Native:      r.IsNative(),
	}

	// Include the hash of every input file in the key
	for _, input := range inputs.Paths() {
		hash, err := c.hasher.File(input)
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
		hash, err := c.hasher.String(env[k])
		if err != nil {
			return nil, err
		}
		key.Env = append(key.Env, newEntry(k, hash))
	}

	// Include the key of every dependency in this key
	for _, dep := range deps {
		depKey, err := c.buildKey(ctx, dep)
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

	// Include rule commands in the key
	for _, cmd := range cmds {
		// For standard "run" commands, use the command text directly.
		// This maintains cache key compatibility with older versions of Zim.
		if cmd.Kind == "run" {
			key.Commands = append(key.Commands, cmd.Argument)
		} else {
			// For new built-in commands, reduce the command to a hash.
			hashStr, err := HashCommand(cmd)
			if err != nil {
				return nil, fmt.Errorf("failed to hash command: %s", err)
			}
			key.Commands = append(key.Commands, hashStr)
		}
	}

	// Determine the hex string for this key
	if err := key.Compute(); err != nil {
		return nil, err
	}
	return key, nil
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

// Command returns a SHA1 hash of the command configuration
func HashCommand(cmd *project.Command) (string, error) {
	entry := &command{
		Kind:       cmd.Kind,
		Argument:   cmd.Argument,
		Attributes: cmd.Attributes,
	}
	h := sha1.New()
	if err := json.NewEncoder(h).Encode(entry); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
