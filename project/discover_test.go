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
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscover(t *testing.T) {

	dir := testDir()

	testComponentDir(dir, "hammer")
	testComponentDir(dir, "nail")

	pDef, defs, err := Discover(dir)
	require.Nil(t, pDef)
	require.Nil(t, err)
	require.Len(t, defs, 2)

	def0 := defs[0]
	def1 := defs[1]

	assert.Equal(t, "hammer", def0.Name)
	assert.Equal(t, "nail", def1.Name)

	assert.Equal(t, path.Join(dir, "src", "hammer", "component.yaml"), def0.Path)
	assert.Equal(t, path.Join(dir, "src", "nail", "component.yaml"), def1.Path)
}
