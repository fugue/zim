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
package cache

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/fugue/zim/exec"
	"github.com/fugue/zim/project"

	"github.com/fugue/zim/definitions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(name, text string) {
	f, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.WriteString(text)
}

func TestCacheKey(t *testing.T) {

	ctx := context.Background()

	tmpDir, err := ioutil.TempDir("", "zim-testing-")
	require.Nil(t, err)
	defer os.RemoveAll(tmpDir)

	repoDir := path.Join(tmpDir, "myrepo")
	require.Nil(t, os.MkdirAll(repoDir, 0755))

	cDir := path.Join(repoDir, "foo")
	require.Nil(t, os.MkdirAll(cDir, 0755))

	cPath := func(name string) string {
		return path.Join(cDir, name)
	}

	writeFile(cPath("go.mod"), "")
	writeFile(cPath("foo_test.go"), "package main")
	writeFile(cPath("foo.go"), "package main")
	writeFile(cPath("exclude_me.go"), "package main")

	cDef := &definitions.Component{
		Path:   cPath("component.yaml"),
		Docker: definitions.Docker{Image: "repo/img:1.2.3"},
		Rules: map[string]definitions.Rule{
			"test": {
				Description: "test it!",
				Inputs:      []string{"${NAME}_test.go", "go.mod"},
				Ignore:      []string{"exclude_me.go"},
				Outputs:     []string{"test_results"},
				Command:     "go test -v",
			},
			"build": {
				Description: "build it!",
				Inputs:      []string{"${NAME}.go", "go.mod"},
				Ignore:      []string{"exclude_me.go"},
				Outputs:     []string{"foo"},
				Command:     "go build",
				Requires: []definitions.Dependency{
					{Rule: "test"},
				},
			},
		},
		Environment: map[string]string{"VOLUME": "11"},
		Toolchain: definitions.Toolchain{
			Items: []definitions.ToolchainItem{
				{Name: "example", Command: "echo EXAMPLE"},
			},
		},
	}

	executor := &exec.FakeExecutor{
		Docker:  true,
		Wrapped: exec.NewBashExecutor(),
	}

	p, err := project.NewWithOptions(project.Opts{
		Root:          repoDir,
		ComponentDefs: []*definitions.Component{cDef},
		Executor:      executor,
	})
	require.Nil(t, err)

	c := p.Components().WithName("foo").First()
	require.NotNil(t, c)
	testRule := c.MustRule("test")
	buildRule := c.MustRule("build")
	buildDeps := buildRule.Dependencies()
	require.Len(t, buildDeps, 1)

	cache := New(Opts{})

	key1, err := cache.Key(ctx, testRule)
	require.Nil(t, err, "Error getting cache key")

	key2, err := cache.Key(ctx, buildRule)
	require.Nil(t, err, "Error getting cache key")

	key1Str := key1.String()
	key2Str := key2.String()

	fmt.Println("key1", key1Str)
	fmt.Println("key2", key2Str)

	// js, _ := json.MarshalIndent(key2, "", "  ")
	// fmt.Println(string(js))

	// Known / golden values
	assert.Equal(t, "76210a1b69110fbab4368f0943451c43d132dbf2", key1Str)
	assert.Equal(t, "dfc17112c86e320d733966fd38da435e4a0804c6", key2Str)
}

func TestCacheKeyNonDocker(t *testing.T) {

	ctx := context.Background()

	tmpDir, err := ioutil.TempDir("", "zim-testing-")
	require.Nil(t, err)
	defer os.RemoveAll(tmpDir)

	cDir := path.Join(tmpDir, "my-component")
	require.Nil(t, os.MkdirAll(cDir, 0755))
	srcPath := path.Join(cDir, "main.go")
	ioutil.WriteFile(srcPath, []byte("some source code"), 0644)

	cDef := &definitions.Component{
		Name: "my-component",
		Path: path.Join(cDir, "component.yaml"),
		Rules: map[string]definitions.Rule{
			"test": {
				Inputs:  []string{"main.go"},
				Outputs: []string{"my-exe"},
				Command: "touch my-exe",
			},
		},
	}

	p, err := project.NewWithOptions(project.Opts{
		ProjectDef: &definitions.Project{
			Name: "test-project",
		},
		Root:          tmpDir,
		ComponentDefs: []*definitions.Component{cDef},
	})
	require.Nil(t, err)

	c := p.Components().WithName("my-component").First()
	require.NotNil(t, c)
	testRule := c.MustRule("test")

	cache := New(Opts{})

	key, err := cache.Key(ctx, testRule)
	require.Nil(t, err, "Error getting cache key")
	keyStr := key.String()

	// js, _ := json.MarshalIndent(key, "", "  ")
	// fmt.Println(string(js))

	// Known / golden values
	assert.Equal(t, "a7c87a2e99c0bbc18b3afbbd65737d8538f33111", keyStr)
}
