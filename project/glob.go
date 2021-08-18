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
	"sort"

	glob "github.com/bmatcuk/doublestar"
)

// MatchFiles returns files within the directory that match the pattern
func MatchFiles(dir, pattern string) ([]string, error) {
	matches, err := glob.Glob(path.Join(dir, pattern))
	if err != nil {
		return nil, fmt.Errorf("invalid source glob %s", pattern)
	}
	// Filter out directories
	results := make([]string, 0, len(matches))
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			results = append(results, match)
		}
	}
	sort.Strings(results)
	return results, nil
}
