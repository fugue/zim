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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/fugue/zim/definitions"
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
					Command:     "ls ${NAME}",
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

	artifactsDir := filepath.Join(dir, "artifacts")
	artifactPath := filepath.Join(dir, "artifacts", "myartifact")

	expectedEnv := flattenEnvironment(map[string]string{
		"ARTIFACT":      artifactPath,
		"ARTIFACTS_DIR": artifactsDir,
		"COMPONENT":     "a",
		"DEP":           "",
		"DEPS":          "",
		"INPUT":         "main.go",
		"KIND":          "flurble",
		"NAME":          "a",
		"NODE_ID":       "a.build",
		"OUTPUT":        "../../artifacts/myartifact",
		"OUTPUTS":       "../../artifacts/myartifact",
		"RULE":          "build",
	})
	buf := bytes.Buffer{}
	var writer io.Writer
	writer = &buf

	m := NewMockExecutor(ctrl)
	m.EXPECT().UsesDocker().AnyTimes()
	m.EXPECT().ExecutorPath(gomock.Any()).DoAndReturn(func(in string) (string, error) {
		return in, nil
	}).AnyTimes()
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

func stringSliceAsInterface(s []string) (result []interface{}) {
	for _, item := range s {
		result = append(result, item)
	}
	return
}

func commandSliceAsInterface(cmds []map[string]string) (result []interface{}) {
	for _, item := range cmds {
		itemIf := make(map[interface{}]interface{})
		for k, v := range item {
			itemIf[k] = v
		}
		result = append(result, itemIf)
	}
	return
}

func getTestRule(opts Opts, ruleName string) (*Rule, error) {
	p, err := NewWithOptions(opts)
	if err != nil {
		return nil, err
	}
	c := p.Components().First()
	if c == nil {
		return nil, errors.New("No components found")
	}
	rule := c.MustRule(ruleName)
	if rule == nil {
		return nil, errors.New("Rule was incorrectly nil")
	}
	return rule, nil
}

func TestStandardRunnerDockerized(t *testing.T) {

	dir := testDir()
	defer os.RemoveAll(dir)
	cDir, cYaml := testComponentDir(dir, "a")
	testComponentFile(cDir, "main.go", "package main")

	defs := []*definitions.Component{
		&definitions.Component{
			Name: "widget",
			Path: cYaml,
			Kind: "flurble",
			Docker: definitions.Docker{
				Image: "go:123",
			},
			Rules: map[string]definitions.Rule{
				"twist-it": definitions.Rule{
					Commands: commandSliceAsInterface([]map[string]string{
						map[string]string{"run": "echo TWIST IT"},
						map[string]string{"run": "echo BOP IT"},
					}),
				},
			},
		},
	}

	rule, err := getTestRule(Opts{Root: dir, ComponentDefs: defs}, "twist-it")
	require.Nil(t, err)

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	artifactsDir := "some-arbitrary-dir"

	expectedEnv := flattenEnvironment(map[string]string{
		"ARTIFACTS_DIR": artifactsDir,
		"COMPONENT":     "widget",
		"DEP":           "",
		"DEPS":          "",
		"INPUT":         "",
		"OUTPUT":        "",
		"OUTPUTS":       "",
		"KIND":          "flurble",
		"NAME":          "widget",
		"NODE_ID":       "widget.twist-it",
		"RULE":          "twist-it",
	})
	buf := bytes.Buffer{}
	var writer io.Writer
	writer = &buf

	m := NewMockExecutor(ctrl)
	m.EXPECT().UsesDocker().Return(true).AnyTimes()
	m.EXPECT().ExecutorPath(rule.ArtifactsDir()).Return(artifactsDir, nil)

	m.EXPECT().Execute(ctx, ExecOpts{
		Name:             "widget.twist-it.0",
		Command:          "echo TWIST IT",
		Image:            "go:123",
		WorkingDirectory: cDir,
		Env:              expectedEnv,
		Stdout:           writer,
		Stderr:           writer,
	}).DoAndReturn(func(ctx context.Context, opts ExecOpts) error {
		return nil
	})

	m.EXPECT().Execute(ctx, ExecOpts{
		Name:             "widget.twist-it.1",
		Command:          "echo BOP IT",
		Image:            "go:123",
		WorkingDirectory: cDir,
		Env:              expectedEnv,
		Stdout:           writer,
		Stderr:           writer,
	}).DoAndReturn(func(ctx context.Context, opts ExecOpts) error {
		return nil
	})

	runner := &StandardRunner{}
	code, err := runner.Run(ctx, rule, RunOpts{
		BuildID:  "777",
		Executor: m,
		Output:   writer,
	})
	require.Nil(t, err)
	require.Equal(t, OK, code)
}

func TestStandardRunnerWhenCondition(t *testing.T) {

	// Tests that rule execution is skipped due to a "when" condition

	dir := testDir()
	defer os.RemoveAll(dir)
	cDir, cYaml := testComponentDir(dir, "a")
	testComponentFile(cDir, "main.go", "package main")

	defs := []*definitions.Component{
		&definitions.Component{
			Name: "widget",
			Path: cYaml,
			Kind: "flurble",
			Rules: map[string]definitions.Rule{
				"twist-it": definitions.Rule{
					When: definitions.Condition{
						ScriptSucceeds: definitions.ConditionScript{
							Run:           "exit 1", // Prevents execution
							SuppressError: true,
						},
					},
					Commands: commandSliceAsInterface([]map[string]string{
						map[string]string{"run": "echo TWIST IT"},
					}),
				},
			},
		},
	}

	rule, err := getTestRule(Opts{Root: dir, ComponentDefs: defs}, "twist-it")
	require.Nil(t, err)

	var outputBuffer bytes.Buffer

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := NewMockExecutor(ctrl)

	expectedEnv := flattenEnvironment(map[string]string{
		"COMPONENT": "widget",
		"KIND":      "flurble",
		"NAME":      "widget",
		"NODE_ID":   "widget.twist-it",
		"RULE":      "twist-it",
	})

	m.EXPECT().UsesDocker().Return(false).AnyTimes()
	m.EXPECT().Execute(ctx, ExecOpts{
		Command:          "exit 1",
		WorkingDirectory: cDir,
		Env:              expectedEnv,
		Image:            "",
		Name:             "widget.twist-it.condition",
		Stdout:           &outputBuffer,
		Stderr:           &outputBuffer,
	}).DoAndReturn(func(ctx context.Context, opts ExecOpts) error {
		return errors.New("Exiting with 1")
	})

	runner := &StandardRunner{}
	code, err := runner.Run(ctx, rule, RunOpts{
		BuildID:  "777",
		Executor: m,
		Output:   &outputBuffer,
	})
	require.Nil(t, err)
	require.Equal(t, Skipped, code)
}

func TestStandardRunnerUnlessCondition(t *testing.T) {

	// Tests that rule execution is skipped due to an "unless" condition

	dir := testDir()
	defer os.RemoveAll(dir)
	cDir, cYaml := testComponentDir(dir, "a")
	testComponentFile(cDir, "main.go", "package main")

	defs := []*definitions.Component{
		&definitions.Component{
			Name: "widget",
			Path: cYaml,
			Kind: "flurble",
			Rules: map[string]definitions.Rule{
				"twist-it": definitions.Rule{
					Unless: definitions.Condition{
						ScriptSucceeds: definitions.ConditionScript{
							Run: "exit 0", // Prevents execution
						},
					},
					Commands: commandSliceAsInterface([]map[string]string{
						map[string]string{"run": "echo TWIST IT"},
					}),
				},
			},
		},
	}

	rule, err := getTestRule(Opts{Root: dir, ComponentDefs: defs}, "twist-it")
	require.Nil(t, err)

	var outputBuffer bytes.Buffer

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := NewMockExecutor(ctrl)

	expectedEnv := flattenEnvironment(map[string]string{
		"COMPONENT": "widget",
		"KIND":      "flurble",
		"NAME":      "widget",
		"NODE_ID":   "widget.twist-it",
		"RULE":      "twist-it",
	})

	m.EXPECT().UsesDocker().Return(false).AnyTimes()
	m.EXPECT().ExecutorPath(gomock.Any()).Return("ignored", nil).AnyTimes()
	m.EXPECT().Execute(ctx, ExecOpts{
		Command:          "exit 0",
		WorkingDirectory: cDir,
		Env:              expectedEnv,
		Image:            "",
		Name:             "widget.twist-it.condition",
		Stdout:           &outputBuffer,
		Stderr:           &outputBuffer,
	}).DoAndReturn(func(ctx context.Context, opts ExecOpts) error {
		// Return nil corresponds to running a script that exits without error
		return nil
	})

	runner := &StandardRunner{}
	code, err := runner.Run(ctx, rule, RunOpts{
		BuildID:  "777",
		Executor: m,
		Output:   &outputBuffer,
	})
	require.Nil(t, err)
	require.Equal(t, Skipped, code)
}

func TestRunnerBuiltIns(t *testing.T) {

	dir := testDir()
	// defer os.RemoveAll(dir)
	fmt.Println(dir)
	ctx := context.Background()
	executor := NewBashExecutor()
	runner := &StandardRunner{}

	testComponentFile(dir, "main.go", "package main")

	c := &Component{name: "test-comp", componentDir: dir}
	r := &Rule{component: c, name: "test-rule", local: true}

	type test struct {
		input     Command
		checkFunc func() (bool, error)
		wantErr   error
	}

	// Sequence of commands and a test for each.
	// These test cases ARE intentionally dependent on each other!
	tests := []test{
		// Create a zip file
		{
			input: Command{
				Kind: "zip",
				Attributes: map[string]interface{}{
					"input":  "main.go",
					"output": "main.go.zip",
				},
			},
			checkFunc: func() (bool, error) {
				return fileExists(filepath.Join(dir, "main.go.zip")), nil
			},
			wantErr: nil,
		},
		// Remove main.go which is now in the zip
		{
			input: Command{
				Kind:     "remove",
				Argument: "main.go",
			},
			checkFunc: func() (bool, error) {
				return !fileExists(filepath.Join(dir, "main.go")), nil
			},
			wantErr: nil,
		},
		// Unzip to get main.go back
		{
			input: Command{
				Kind: "unzip",
				Attributes: map[string]interface{}{
					"input": "main.go.zip",
				},
			},
			checkFunc: func() (bool, error) {
				return fileExists(filepath.Join(dir, "main.go")), nil
			},
			wantErr: nil,
		},
		// Archive main.go next
		{
			input: Command{
				Kind: "archive",
				Attributes: map[string]interface{}{
					"input":  "main.go",
					"output": "main.go.tgz",
				},
			},
			checkFunc: func() (bool, error) {
				return fileExists(filepath.Join(dir, "main.go.tgz")), nil
			},
			wantErr: nil,
		},
		// Move main.go to main.go.bak
		{
			input: Command{
				Kind: "move",
				Attributes: map[string]interface{}{
					"src": "main.go",
					"dst": "main.go.bak",
				},
			},
			checkFunc: func() (bool, error) {
				return (fileExists(filepath.Join(dir, "main.go.bak")) &&
					!fileExists(filepath.Join(dir, "main.go"))), nil
			},
			wantErr: nil,
		},
		// Unarchive main.go.tgz to get main.go back
		{
			input: Command{
				Kind: "unarchive",
				Attributes: map[string]interface{}{
					"input": "main.go.tgz",
				},
			},
			checkFunc: func() (bool, error) {
				return fileExists(filepath.Join(dir, "main.go")), nil
			},
			wantErr: nil,
		},
		// Create an empty directory
		{
			input: Command{
				Kind:     "mkdir",
				Argument: "the-directory",
			},
			checkFunc: func() (bool, error) {
				return fileExists(filepath.Join(dir, "the-directory")), nil
			},
			wantErr: nil,
		},
		// Create a file in the directory
		{
			input: Command{
				Kind:     "run",
				Argument: "touch the-directory/foo",
			},
			checkFunc: func() (bool, error) {
				return fileExists(filepath.Join(dir, "the-directory/foo")), nil
			},
			wantErr: nil,
		},
		// Copy the directory to a new location
		{
			input: Command{
				Kind: "copy",
				Attributes: map[string]interface{}{
					"src": "the-directory",
					"dst": "the-directory-copy",
				},
			},
			checkFunc: func() (bool, error) {
				return fileExists(filepath.Join(dir, "the-directory-copy/foo")), nil
			},
			wantErr: nil,
		},
		// Clean the directory
		{
			input: Command{
				Kind:     "cleandir",
				Argument: "the-directory-copy",
			},
			checkFunc: func() (bool, error) {
				return (fileExists(filepath.Join(dir, "the-directory-copy")) &&
					!fileExists(filepath.Join(dir, "the-directory-copy/foo"))), nil
			},
			wantErr: nil,
		},
	}

	for _, tc := range tests {
		var stdout bytes.Buffer
		opts := RunOpts{
			Executor:    executor,
			Output:      &stdout,
			DebugOutput: &stdout,
			Debug:       true,
		}
		r.commands = []*Command{&tc.input}
		code, err := runner.Run(ctx, r, opts)
		if errStringOrEmpty(err) != errStringOrEmpty(tc.wantErr) {
			t.Errorf("expected: %v, got: %v", tc.wantErr, err)
		}
		if code != OK {
			fmt.Println(stdout.String())
			t.Errorf("expected code to be ok for %+v", tc.input)
		}
		ok, err := tc.checkFunc()
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Errorf("Check failed on %+v", tc.input)
		}
	}
}
