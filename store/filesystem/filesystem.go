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

package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/fugue/zim/store"
)

type fileStore struct {
	rootDirectory string
}

// New returns a store.Store interface backed by the local filesystem
func New(rootDirectory string) store.Store {
	return &fileStore{rootDirectory: rootDirectory}
}

func (s *fileStore) path(key string) string {

	// Use a nested subdirectory tree for the cache to help avoid
	// having one folder with many 10s of thousands of files in it, since
	// that can be difficult to navigate in file explorers and similar.

	keyLen := len(key)

	// Two levels of nesting, e.g. "abcd" is stored at "<cache>/ab/abcd"
	if keyLen >= 4 {
		prefix1 := key[:2]
		prefix2 := key[2:4]
		return filepath.Join(s.rootDirectory, prefix1, prefix2, key)
	}

	// One level of nesting, e.g. "abc" is stored at "<cache>/ab/abc"
	if keyLen >= 2 {
		prefix := key[:2]
		return filepath.Join(s.rootDirectory, prefix, key)
	}

	// No nesting for short keys (uncommon)
	return filepath.Join(s.rootDirectory, key)
}

func (s *fileStore) Get(ctx context.Context, key, dst string) error {

	path := s.path(key)
	srcFile, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("not found: %s", key)
		}
		return fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer srcFile.Close()

	return copyFile(srcFile, dst)
}

func (s *fileStore) Put(ctx context.Context, key, src string, meta map[string]string) error {

	path := s.path(key)
	pathDir := filepath.Dir(path)

	if err := os.MkdirAll(pathDir, 0755); err != nil {
		return err
	}

	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", src, err)
	}
	defer f.Close()

	if err := copyFile(f, path); err != nil {
		return err
	}

	metaPath := fmt.Sprintf("%s.meta", path)
	metaBytes, err := json.Marshal(store.ItemMeta{Meta: meta})
	if err != nil {
		return fmt.Errorf("failed to marshal metadata; key: %s err: %w", key, err)
	}
	metaFile, err := os.Create(metaPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", metaPath, err)
	}
	defer metaFile.Close()

	if _, err := metaFile.Write(metaBytes); err != nil {
		return fmt.Errorf("failed to write file %s: %w", metaPath, err)
	}
	return nil
}

func copyFile(f *os.File, dstPath string) error {

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", dstPath, err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, f); err != nil {
		return fmt.Errorf("failed to write file %s: %w", dstPath, err)
	}
	return nil
}

// Head checks if the item exists in the store
func (s *fileStore) Head(ctx context.Context, key string) (store.ItemMeta, error) {

	metaPath := fmt.Sprintf("%s.meta", s.path(key))
	metaBytes, err := ioutil.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return store.ItemMeta{}, store.NotFound(fmt.Sprintf("not found: %s", key))
		}
		return store.ItemMeta{}, err
	}

	var itemMeta store.ItemMeta
	if err := json.Unmarshal(metaBytes, &itemMeta); err != nil {
		return store.ItemMeta{}, fmt.Errorf("failed to parse metadata: %w", err)
	}
	return itemMeta, nil
}
