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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUUID(t *testing.T) {
	// len("a836f18b-9982-4631-aaad-0925740f8884") == 36
	id := UUID()
	assert.Len(t, id, 36)
}

func TestFlattenEnvironment(t *testing.T) {
	env := map[string]string{
		"NAME":  "widget",
		"ROOT":  "/path/to/root",
		"EMPTY": "",
		"":      "UH",
	}
	flattened := flattenEnvironment(env)
	// Sorted alphabetically by key
	expected := []string{
		"=UH",
		"EMPTY=",
		"NAME=widget",
		"ROOT=/path/to/root",
	}
	assert.Equal(t, expected, flattened)
}

func TestLatestModification(t *testing.T) {
	dir := testDir()
	defer os.RemoveAll(dir)

	now := time.Now()

	_, fooYaml := testComponentDir(dir, "foo")
	_, barYaml := testComponentDir(dir, "bar")

	mtime, err := latestModification([]string{fooYaml, barYaml})
	require.Nil(t, err)

	dt := mtime.Sub(now)
	assert.True(t, dt.Seconds() < 1)
}
