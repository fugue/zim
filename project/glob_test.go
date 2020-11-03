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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchFiles(t *testing.T) {

	dir := testDir()
	cDir, yamlPath := testComponentDir(dir, "widget")
	defer os.RemoveAll(dir)

	// Directories should be ignored
	matches, err := MatchFiles(dir, "*")
	require.Nil(t, err)
	require.Equal(t, []string{}, matches)

	// Find the widget/component.yaml
	matches, err = MatchFiles(cDir, "*")
	require.Nil(t, err)
	require.Equal(t, []string{yamlPath}, matches)

	go1 := path.Join(cDir, "main.go")
	go2 := path.Join(cDir, "blah.go")

	ioutil.WriteFile(go1, []byte("foo"), 0644)
	ioutil.WriteFile(go2, []byte("blah"), 0644)

	// Find the two new go files. They should be sorted alphabetically.
	matches, err = MatchFiles(cDir, "*.go")
	require.Nil(t, err)
	require.Equal(t, []string{go2, go1}, matches)

	// Find a specific go file by full name
	matches, err = MatchFiles(cDir, "main.go")
	require.Nil(t, err)
	require.Equal(t, []string{go1}, matches)
}

func TestGlob(t *testing.T) {

	dir := testDir()
	cDir, _ := testComponentDir(dir, "widget")
	ioutil.WriteFile(filepath.Join(cDir, "main.go"), []byte("package main"), 0644)
	defer os.RemoveAll(dir)

	// <component>/src/widget/main.go
	// <component>/src/widget/component.yaml

	expMainPath := filepath.Join(cDir, "main.go")
	expCompPath := filepath.Join(cDir, "component.yaml")

	type test struct {
		pattern string
		want    []string
	}
	tests := []test{
		{"src/*/*.yaml", []string{expCompPath}},
		{"src/widget/*.yaml", []string{expCompPath}},
		{"src/widget/component.yaml", []string{expCompPath}},
		{"src/*/*", []string{expCompPath, expMainPath}},
		{"src/*/*.c", []string{}},
		{"**", []string{expCompPath, expMainPath}},
		{"**/*", []string{expCompPath, expMainPath}},
		{"**/*.yaml", []string{expCompPath}},
		{"**/*ml", []string{expCompPath}},
		{"**/*.go", []string{expMainPath}},
		{"**/*o", []string{expMainPath}},
		{"**/ma*", []string{expMainPath}},
		{"**/co*", []string{expCompPath}},
		{"**/xy*", []string{}},
		{"**/*xy", []string{}},
		{"**/component.yaml", []string{expCompPath}},
	}
	for _, tc := range tests {
		pat := filepath.Join(dir, tc.pattern)
		result, err := Glob(pat)
		// fmt.Println(pat, result, err)
		require.Nil(t, err)
		require.Equal(t, tc.want, result)
	}
}

func TestBadGlob(t *testing.T) {
	type test struct {
		input   string
		wantErr error
	}
	tests := []test{
		{"/foo/bar/*/**/foo", errors.New("Invalid pattern: /foo/bar/*/**/foo")},
		{"/foo/*/bar/**/foo", errors.New("Invalid pattern: /foo/*/bar/**/foo")},
		{"**/**/*.go", errors.New("Invalid pattern: **/**/*.go")},
		{"**/foo/*.go", errors.New("Invalid pattern suffix: **/foo/*.go")},
	}
	for _, tc := range tests {
		_, err := Glob(tc.input)
		require.NotNil(t, err)
		require.Equal(t, tc.wantErr, err)
	}
}

func TestGlobFilterFiles(t *testing.T) {
	type test struct {
		paths   []string
		pattern string
		want    []string
	}
	tests := []test{
		{[]string{"a/b/c.go", "b/c/d.txt"}, "*.go", []string{"a/b/c.go"}},
		{[]string{"a/b/c.go", "b/c/d.txt"}, "*.txt", []string{"b/c/d.txt"}},
		{[]string{"a/b/c.go", "b/c/d.txt"}, "*.txt", []string{"b/c/d.txt"}},
		{[]string{"a/b/c.go", "b/c/d.txt"}, "d.txt", []string{"b/c/d.txt"}},
		{[]string{"a/b/c.go", "b/c/d.txt"}, "missing", []string{}},
		{[]string{"foo.c", "bar.abc"}, "*c", []string{"foo.c", "bar.abc"}},
		{[]string{"cook.txt", "bar.abc"}, "c*", []string{"cook.txt"}},
		{[]string{"a.blah.txt", "bar.abc"}, "a.blah.txt", []string{"a.blah.txt"}},
	}
	for _, tc := range tests {
		result := filterFiles(tc.paths, tc.pattern)
		assert.Equal(t, tc.want, result)
	}
}
