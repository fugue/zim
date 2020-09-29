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
package store

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Download files from the store with the given prefix to the given dst dir
func Download(ctx context.Context, store Store, prefix, dstDir string, ignore []string) ([]string, error) {
	var downloaded []string
	keys, err := store.List(ctx, prefix)
	if err != nil {
		return downloaded, err
	}
	// Could parallelize this
	for _, key := range keys {
		for _, ignorePrefix := range ignore {
			if strings.HasPrefix(key, ignorePrefix) {
				continue
			}
		}
		localPath := path.Join(dstDir, strings.TrimPrefix(key, prefix))
		localDir := path.Dir(localPath)
		if _, err := os.Stat(localDir); err != nil {
			if err := os.MkdirAll(localDir, 0755); err != nil {
				return downloaded, err
			}
		}
		itemInfo, _ := store.Head(ctx, key)
		if itemInfo.Exists() {
			localSize, localModTime := fileStat(localPath)
			if localSize > 0 && localSize == itemInfo.Size {
				if localModTime.Equal(itemInfo.LastModified) || localModTime.After(itemInfo.LastModified) {
					continue // File already exists locally
				}
			}
		}
		if err := store.Get(ctx, key, localPath); err != nil {
			return downloaded, err
		}
		// fmt.Println("Downloaded", key)
		downloaded = append(downloaded, localPath)
	}
	return downloaded, nil
}

// Upload a local directory to the store at the specified prefix
func Upload(ctx context.Context, store Store, prefix, srcDir string, ignore []string) ([]string, error) {
	var uploaded []string
	if stat, err := os.Stat(srcDir); err != nil || !stat.IsDir() {
		return uploaded, fmt.Errorf("No directory %s: %s", srcDir, err)
	}
	callback := func(filePath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fileInfo.IsDir() {
			return nil
		}
		relativePath := strings.TrimPrefix(filePath, srcDir)
		for _, ignorePrefix := range ignore {
			if strings.HasPrefix(relativePath, ignorePrefix) {
				continue
			}
		}
		key := path.Join(prefix, relativePath)
		if itemInfo, _ := store.Head(ctx, key); itemInfo.Exists() {
			if localSize, _ := fileStat(filePath); localSize == itemInfo.Size {
				return nil // File already exists in store
			}
		}
		if err := store.Put(ctx, key, filePath, nil); err != nil {
			return fmt.Errorf("Failed to upload %s to %s: %s", filePath, key, err)
		}
		// fmt.Println("Uploaded", key)
		uploaded = append(uploaded, key)
		return nil
	}
	if err := filepath.Walk(srcDir, callback); err != nil {
		return uploaded, fmt.Errorf("Failed to upload %s: %s", srcDir, err)
	}
	return uploaded, nil
}
