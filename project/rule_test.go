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
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testCompFoo = `
name: foo
rules:
  test:
    local: true
    inputs:
     - main.go
    outputs:
     - test_results.txt
    command: touch ${OUTPUT}
  build:
    requires:
     - rule: test
    inputs:
     - main.go
    outputs:
     - foo
    command: go build -v
`
	testCompBar = `
name: bar
rules:
  build:
    requires:
     - rule: build
       component: foo
    inputs:
     - main.go
    outputs:
     - bar
    commands:
     - go build -v
`
	testGoMain = `
package main
import fmt
func main() { fmt.Println("Hello World!") }
`

	testCompConditions = `
name: conditions-test
rules:
  build-when-run:
    inputs:
     - main.go
    outputs:
     - bar
    when:
      resource_exists: main.go
  build-when-skip:
    inputs:
     - main.go
    outputs:
     - bar
    when:
      resource_exists: missing.go
  build-unless-skip:
    inputs:
     - main.go
    outputs:
     - bar
    unless:
      resource_exists: main.go
  build-unless-run:
    inputs:
     - main.go
    outputs:
     - bar
    unless:
      resource_exists: missing.go
`
)

func TestRule(t *testing.T) {

	dir := testDir()
	defer os.RemoveAll(dir)

	testComponent(dir, "foo", testCompFoo, map[string]string{
		"main.go": testGoMain,
	})
	testComponent(dir, "bar", testCompBar, map[string]string{
		"main.go": testGoMain,
	})

	p, err := New(dir)
	require.Nil(t, err)

	c := p.Components()
	require.Len(t, c, 2)

	foo := c.WithName("foo").First()
	bar := c.WithName("bar").First()

	assert.Equal(t, "foo", foo.Name())
	assert.Equal(t, "bar", bar.Name())

	test, found := foo.Rule("test")
	require.True(t, found)

	build, found := foo.Rule("build")
	require.True(t, found)

	assert.Equal(t, "foo.test", test.NodeID())
	assert.Equal(t, "foo.build", build.NodeID())

	buildDeps := build.Dependencies()
	require.Len(t, buildDeps, 1)

	buildDep := buildDeps[0]
	require.Equal(t, test, buildDep)

	assert.False(t, build.OutputsExist())

	missing, err := build.MissingOutputs().RelativePaths(p.RootAbsPath())
	require.Nil(t, err)
	assert.Equal(t, []string{"artifacts/foo"}, missing)

	env, err := build.Environment()
	require.Nil(t, err)

	assert.Equal(t, map[string]string{
		"COMPONENT": "foo",
		"DEP":       "test_results.txt",
		"DEPS":      "test_results.txt",
		"INPUT":     "main.go",
		"KIND":      "",
		"NAME":      "foo",
		"NODE_ID":   "foo.build",
		"OUTPUT":    "../artifacts/foo",
		"OUTPUTS":   "../artifacts/foo",
		"RULE":      "build",
	}, env)

	inputs, err := build.Inputs()
	require.Nil(t, err)
	assert.Equal(t, []string{path.Join(dir, "foo", "main.go")}, inputs.Paths())
}

func TestRuleMissingDepOutputs(t *testing.T) {

	dir := testDir()
	defer os.RemoveAll(dir)

	testComponent(dir, "bar", testCompBar, map[string]string{
		"main.go": testGoMain,
	})

	p, err := New(dir)
	require.Nil(t, err)

	c := p.Components().WithName("bar").First()
	require.NotNil(t, c)

	_, found := c.Rule("build")
	require.False(t, found)
}

func TestRuleOutputs(t *testing.T) {

	dir := testDir()
	defer os.RemoveAll(dir)

	testComponent(dir, "foo", testCompFoo, map[string]string{
		"main.go": testGoMain,
	})
	testComponent(dir, "bar", testCompBar, map[string]string{
		"main.go": testGoMain,
	})

	p, err := New(dir)
	require.Nil(t, err)
	bar := p.Components().First()
	require.NotNil(t, bar)
	require.Equal(t, "bar", bar.Name())

	build, found := bar.Rule("build")
	require.True(t, found)

	outs, err := build.Outputs().RelativePaths(p.RootAbsPath())
	require.Nil(t, err)
	assert.Equal(t, []string{path.Join("artifacts", "bar")}, outs)

	// absOuts := build.OutputsAbs()
	// assert.Equal(t, []string{path.Join(dir, "artifacts", "bar")}, absOuts)
}
