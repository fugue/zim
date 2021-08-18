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
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"
)

// UUID returns a unique ID as a string
func UUID() string {
	id := uuid.NewV4()
	return id.String()
}

func combineEnvironment(envs ...map[string]string) map[string]string {
	result := map[string]string{}
	for _, env := range envs {
		if env != nil {
			for k, v := range env {
				result[k] = v
			}
		}
	}
	return result
}

func copyEnvironment(env map[string]string) map[string]string {
	new := map[string]string{}
	for k, v := range env {
		new[k] = v
	}
	return new
}

func flattenEnvironment(env map[string]string) []string {
	var index int
	result := make([]string, len(env))
	for k, v := range env {
		result[index] = fmt.Sprintf("%s=%s", k, v)
		index++
	}
	sort.Strings(result)
	return result
}

func latestModification(files []string) (time.Time, error) {
	if len(files) == 0 {
		return time.Time{}, errors.New("No input files")
	}
	var latestMod time.Time
	for _, fpath := range files {
		info, err := os.Stat(fpath)
		if err != nil {
			return time.Time{}, fmt.Errorf("Failed to stat %s: %s", fpath, err)
		}
		if info.ModTime().After(latestMod) {
			latestMod = info.ModTime()
		}
	}
	return latestMod, nil
}

func substituteVarsSlice(strings []string, variables map[string]string) []string {
	if len(strings) == 0 || len(variables) == 0 {
		return strings
	}
	result := make([]string, len(strings))
	for i, s := range strings {
		result[i] = substituteVars(s, variables)
	}
	return result
}

func substituteVars(s string, variables map[string]string) string {
	if s == "" || len(variables) == 0 {
		return s
	}
	// ${NAME}.zip -> foo.zip
	for k, v := range variables {
		s = strings.ReplaceAll(s, fmt.Sprintf("${%s}", k), v)
	}
	return s
}

func copyStrings(input []string) []string {
	if input == nil {
		return nil
	}
	result := make([]string, len(input))
	copy(result, input)
	return result
}
