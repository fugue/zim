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
	"path"
	"testing"

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
