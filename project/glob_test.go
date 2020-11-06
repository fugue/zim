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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		{"**", []string{}},
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
		assert.Nil(t, err)
		assert.Equal(t, tc.want, result)
	}
}
