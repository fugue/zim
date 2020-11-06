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
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mattn/go-zglob"
)

// Glob returns a list of all files found that match the given pattern.
// This has been optimized since it is a primary source of loading in Zim.
func Glob(pattern string) (results []string, err error) {

	matches, err := glob(pattern)
	if err != nil {
		return nil, err
	}

	// Filter out matches that are directories
	results = make([]string, 0, len(matches))
	for _, m := range matches {
		if statInfo, err := os.Stat(m); err == nil && statInfo.Mode().IsDir() {
			continue
		}
		results = append(results, m)
	}

	sort.Strings(results)
	return results, nil
}

func glob(pattern string) (results []string, err error) {
	// Prefer the built-in glob when possible. It doesn't support ** patterns.
	if !strings.Contains(pattern, "**") {
		return filepath.Glob(pattern)
	}
	return zglob.Glob(pattern)
}
