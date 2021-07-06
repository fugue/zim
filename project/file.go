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
package project

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"
)

// FileSystem implements Provider
type FileSystem struct {
	root string
}

// NewFileSystem returns
func NewFileSystem(root string) (*FileSystem, error) {
	return &FileSystem{root}, nil
}

// Init accepts configuration options from Project configuration
func (fs *FileSystem) Init(opts map[string]interface{}) error {
	return nil
}

// Name identifies the type of the FileSystem Provider
func (fs *FileSystem) Name() string {
	return "file"
}

// New returns a File Resource
func (fs *FileSystem) New(path string) Resource {
	return NewFile(path)
}

// Match files by name
func (fs *FileSystem) Match(pattern string) (Resources, error) {
	if !strings.Contains(pattern, "*") {
		match := path.Join(fs.root, pattern)
		matchInfo, err := os.Stat(match)
		if err != nil {
			if os.IsNotExist(err) {
				return Resources{}, nil
			}
			return nil, fmt.Errorf("Failed to stat input %s: %s",
				pattern, err)
		}
		if matchInfo.IsDir() {
			return nil, fmt.Errorf("Input cannot be a dir: %s", pattern)
		}
		return Resources{fs.New(match)}, nil
	}
	matches, err := MatchFiles(fs.root, pattern)

	if err != nil {
		return nil, fmt.Errorf("Failed to match resources %s: %s", pattern, err)
	}
	rs := make(Resources, 0, len(matches))
	for _, match := range matches {
		rs = append(rs, fs.New(match))
	}
	return rs, nil
}

// File implements the Resource interface
type File struct {
	path string
}

// NewFile returns a File given the path
func NewFile(path string) *File {
	return &File{path}
}

// OnFilesystem is true for files
func (f *File) OnFilesystem() bool {
	return true
}

// Cacheable is true for Files since they can be uploaded to a cache
func (f *File) Cacheable() bool {
	return true
}

// Name of the Resource
func (f *File) Name() string {
	return path.Base(f.path)
}

// Path returns the absolute path to the File
func (f *File) Path() string {
	return f.path
}

// Exists indicates whether the File currently exists
func (f *File) Exists() (bool, error) {
	if _, err := os.Stat(f.path); err == nil {
		return true, nil
	}
	return false, nil
}

// Hash of File contents
func (f *File) Hash() (string, error) {
	return HashFile(f.path)
}

// LastModified time of this File
func (f *File) LastModified() (time.Time, error) {
	info, err := os.Stat(f.path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// AsFile returns the path to the file
func (f *File) AsFile() (string, error) {
	return f.path, nil
}
