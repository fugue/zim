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
	"path/filepath"
	"strings"

	walk "github.com/karrick/godirwalk"
)

// Glob returns a list of all files found that match the given pattern.
// This has been optimized since it is a primary source of loading in Zim.
func Glob(pattern string) ([]string, error) {

	doubleStarCount := strings.Count(pattern, "**")

	if doubleStarCount > 1 {
		// At most one occurence of "**" is allowed in patterns for now
		return nil, fmt.Errorf("Invalid pattern: %s", pattern)
	} else if doubleStarCount == 0 {
		// Delegate patterns without double stars to filepath.Glob
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		// Exclude directories and files that we can't read for whatever reason
		results := make([]string, 0, len(matches))
		for _, match := range matches {
			fileInfo, err := os.Stat(match)
			if err == nil && !fileInfo.IsDir() {
				results = append(results, match)
			}
		}
		return results, nil
	}

	// If we made it to this point, it means there is one "**" in the pattern.
	// We will intentionally only handle simple forms of this for now, since
	// this search is the most intensive aspect of Zim execution in many cases.
	// Specifically, we'll support the following forms:
	//  1. src/** (every file under src, recursively)
	//  2. src/**/* (equivalent to the above)
	//  3. src/**/*suffix (every file under src with the suffix, recursively)
	//  4. src/**/prefix* (every file under src with the prefix, recursively)
	//  5. src/**/main.go (every file named main.go under src, recursively)

	// Disallow a single star preceding a double star
	singleStarIndex := strings.Index(pattern, "*")
	doubleStarIndex := strings.Index(pattern, "**")
	if singleStarIndex >= 0 && singleStarIndex < doubleStarIndex {
		return nil, fmt.Errorf("Invalid pattern: %s", pattern)
	}

	pattern = filepath.ToSlash(pattern)
	parts := strings.Split(pattern, "/")

	// Disallow a double star joined to any other string (e.g. foo** or **foo)
	if doubleStarIndex > 0 && pattern[doubleStarIndex-1] != filepath.Separator {
		return nil, fmt.Errorf("Invalid pattern: %s", pattern)
	}
	if doubleStarIndex < len(pattern)-2 && pattern[doubleStarIndex+2] != filepath.Separator {
		return nil, fmt.Errorf("Invalid pattern: %s", pattern)
	}

	// The base is the path prefix preceding the "**"
	base, rest := splitPathParts("**", parts)
	baseStr := filepath.Join(base...)
	if pattern[0] == filepath.Separator {
		baseStr = string(filepath.Separator) + baseStr
	}

	// We will only support simple glob patterns for now, which are limited to
	// at most a star directory and a star file pattern. Raise an error if the
	// pattern suffix is more complex than this.
	if len(rest) > 2 {
		badSuffix := filepath.Join(rest...)
		return nil, fmt.Errorf("Invalid pattern suffix: %s", badSuffix)
	}

	// Recursively search on double stars only
	lastPatternPart := rest[len(rest)-1]
	var namePattern string
	if lastPatternPart != "*" && lastPatternPart != "**" {
		namePattern = lastPatternPart
	}

	// Find all files and then filter according to the name pattern
	files := findAllFiles(baseStr, true)
	return filterFiles(files, namePattern), nil
}

func findAllFiles(dir string, recurse bool) (files []string) {
	walk.Walk(dir, &walk.Options{
		Callback: func(path string, de *walk.Dirent) error {
			if de.IsRegular() {
				files = append(files, path)
			} else if de.IsDir() && path != dir && !recurse {
				return walk.SkipThis
			}
			return nil
		},
		ErrorCallback: func(path string, err error) walk.ErrorAction {
			return walk.SkipNode
		},
		Unsorted: true,
	})
	return
}

func filterFiles(paths []string, namePattern string) (result []string) {

	if namePattern == "" {
		return paths
	}

	var matchStr string
	var checkFull, checkSuffix, checkPrefix bool

	starIndex := strings.LastIndex(namePattern, "*")
	if starIndex == -1 {
		checkFull = true
		matchStr = namePattern
	} else if starIndex == 0 {
		checkSuffix = true
		matchStr = namePattern[1:len(namePattern)]
	} else {
		checkPrefix = true
		matchStr = namePattern[0 : len(namePattern)-1]
	}

	for _, path := range paths {
		pathLen := len(path)
		if path == "" || path[pathLen-1] == '/' {
			// Ignore empty paths and directories
			continue
		}
		// Slice the filename part of the path
		lastSlashIndex := strings.LastIndex(path, "/")
		name := path[lastSlashIndex+1 : pathLen]
		// Match the filename according to one of the three pattern types
		if checkFull {
			if name == namePattern {
				result = append(result, path)
			}
		} else if checkPrefix {
			if strings.HasPrefix(name, matchStr) {
				result = append(result, path)
			}
		} else if checkSuffix {
			if strings.HasSuffix(name, matchStr) {
				result = append(result, path)
			}
		}
	}

	if result == nil {
		return []string{}
	}
	return
}

func splitPathParts(splitStr string, pathParts []string) ([]string, []string) {
	consumed := 0
	base := make([]string, 0, len(pathParts))
	for _, part := range pathParts {
		if strings.Contains(part, splitStr) {
			break
		}
		base = append(base, part)
		consumed++
	}
	rest := pathParts[consumed:]
	return base, rest
}
