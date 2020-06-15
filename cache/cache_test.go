package cache

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

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
			"test": definitions.Rule{
				Description: "test it!",
				Inputs:      []string{"${NAME}_test.go", "go.mod"},
				Ignore:      []string{"exclude_me.go"},
				Outputs:     []string{"test_results"},
				Command:     "go test -v",
			},
			"build": definitions.Rule{
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

	p, err := project.NewWithOptions(project.Opts{
		Root:          repoDir,
		ComponentDefs: []*definitions.Component{cDef},
	})
	require.Nil(t, err)

	c := p.Components().WithName("foo").First()
	require.NotNil(t, c)
	testRule := c.MustRule("test")
	buildRule := c.MustRule("build")
	buildDeps := buildRule.Dependencies()
	require.Len(t, buildDeps, 1)

	cache := New(nil)

	key1, err := cache.Key(testRule)
	require.Nil(t, err, "Error getting cache key")

	key2, err := cache.Key(buildRule)
	require.Nil(t, err, "Error getting cache key")

	key1Str := key1.String()
	key2Str := key2.String()

	fmt.Println("key1", key1Str)
	fmt.Println("key2", key2Str)

	// js, _ := json.MarshalIndent(key2, "", "  ")
	// fmt.Println(string(js))

	// Known / golden values
	assert.Equal(t, "c5cf8e1811b2feef3541fcd84cf633a7d59fa936", key1Str)
	assert.Equal(t, "109ca91d6c59ac00c11c34359f16b750ede451ba", key2Str)
}
