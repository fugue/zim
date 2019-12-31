package project

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/LuminalHQ/zim/definitions"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestStandardRunner(t *testing.T) {

	dir := testDir()
	defer os.RemoveAll(dir)
	cDir, cYaml := testComponentDir(dir, "a")
	testComponentFile(cDir, "main.go", "package main")

	defs := []*definitions.Component{
		&definitions.Component{
			Name: "a",
			Path: cYaml,
			Kind: "flurble",
			Rules: map[string]definitions.Rule{
				"build": definitions.Rule{
					Description: "build",
					Inputs:      []string{"main.go"},
					Outputs:     []string{"myartifact"},
					Commands: []string{
						"ls ${NAME}",
					},
				},
			},
		},
	}

	p, err := NewWithOptions(Opts{Root: dir, ComponentDefs: defs})
	require.Nil(t, err)

	c := p.Components().First()
	require.NotNil(t, c)

	rule := c.MustRule("build")
	require.NotNil(t, rule)

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expectedEnv := flattenEnvironment(map[string]string{
		"COMPONENT": "a",
		"DEP":       "",
		"DEPS":      "",
		"INPUT":     "main.go",
		"INPUTS":    "main.go",
		"KIND":      "flurble",
		"NAME":      "a",
		"NODE_ID":   "a.build",
		"OUTPUT":    "../../artifacts/myartifact",
		"OUTPUTS":   "../../artifacts/myartifact",
		"RULE":      "build",
	})
	buf := bytes.Buffer{}
	var writer io.Writer
	writer = &buf

	m := NewMockExecutor(ctrl)
	m.EXPECT().Execute(ctx, ExecOpts{
		Name:             "a.build.0",
		Command:          "ls ${NAME}",
		WorkingDirectory: cDir,
		Env:              expectedEnv,
		Stdout:           writer,
		Stderr:           writer,
	}).DoAndReturn(func(ctx context.Context, opts ExecOpts) error {
		// Create output artifact as if the build happened
		artifact := path.Join(dir, "artifacts", "myartifact")
		return ioutil.WriteFile(artifact, []byte{}, 0644)
	})

	runner := &StandardRunner{}
	code, err := runner.Run(ctx, rule, RunOpts{
		BuildID:  "1234",
		Executor: m,
		Output:   writer,
	})
	require.Nil(t, err)
	require.Equal(t, OK, code)
}
