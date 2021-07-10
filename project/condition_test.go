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
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRuleCondition(t *testing.T) {

	dir := testDir()
	defer os.RemoveAll(dir)

	testComponent(dir, "conditions-test", testCompConditions,
		map[string]string{
			"main.go": testGoMain,
		})

	p, err := New(dir)
	require.Nil(t, err)

	comp := p.Components().First()
	require.NotNil(t, comp)
	require.Equal(t, "conditions-test", comp.Name())

	var stdout bytes.Buffer
	executor := NewBashExecutor()
	execOpts := RunOpts{
		Output:      &stdout,
		DebugOutput: &stdout,
		Debug:       false,
	}
	ctx := context.Background()
	env := map[string]string{}

	// "when" condition is true
	build, err := comp.Rule("build-when-run")
	require.Nil(t, err)
	conditionsMet, err := CheckConditions(ctx, build, execOpts, executor, env)
	require.Nil(t, err)
	require.True(t, conditionsMet)

	// "when" condition is false
	build, err = comp.Rule("build-when-skip")
	require.Nil(t, err)
	conditionsMet, err = CheckConditions(ctx, build, execOpts, executor, env)
	require.Nil(t, err)
	require.False(t, conditionsMet)

	// "unless" condition is false (so the rule should run)
	build, err = comp.Rule("build-unless-run")
	require.Nil(t, err)
	conditionsMet, err = CheckConditions(ctx, build, execOpts, executor, env)
	require.Nil(t, err)
	require.True(t, conditionsMet)

	// "unless" condition is true (so the rule should NOT run)
	build, err = comp.Rule("build-unless-skip")
	require.Nil(t, err)
	conditionsMet, err = CheckConditions(ctx, build, execOpts, executor, env)
	require.Nil(t, err)
	require.False(t, conditionsMet)
}

func TestConditionScript(t *testing.T) {

	dir := testDir()
	defer os.RemoveAll(dir)
	ctx := context.Background()
	executor := NewBashExecutor()
	env := map[string]string{}

	c := &Component{name: "test-comp", componentDir: dir}
	r := &Rule{component: c, name: "test-rule"}

	type test struct {
		input   Condition
		want    bool
		wantErr error
	}

	// Test cases for various permutations of ConditionScripts
	tests := []test{
		{
			input: Condition{
				ScriptSucceeds: ConditionScript{
					Run:        "echo FOO",
					WithOutput: "FOO",
				},
			},
			want:    true,
			wantErr: nil,
		},
		{
			input: Condition{
				ScriptSucceeds: ConditionScript{
					Run: "echo FOO",
				},
			},
			want:    true,
			wantErr: nil,
		},
		{
			input: Condition{
				ScriptSucceeds: ConditionScript{
					Run:        "echo NOPE",
					WithOutput: "FOO",
				},
			},
			want:    false,
			wantErr: nil,
		},
		{
			input: Condition{
				ScriptSucceeds: ConditionScript{
					Run:           "exit 1",
					SuppressError: true,
				},
			},
			want:    false,
			wantErr: nil,
		},
		{
			input: Condition{
				ScriptSucceeds: ConditionScript{
					Run: "exit 42",
				},
			},
			want:    false,
			wantErr: fmt.Errorf("exit status 42"),
		},
		{
			input:   Condition{},
			want:    true,
			wantErr: nil,
		},
	}

	for _, tc := range tests {
		var stdout bytes.Buffer
		runOpts := RunOpts{
			Output:      &stdout,
			DebugOutput: &stdout,
			Debug:       false,
		}
		conditionsMet, err := CheckCondition(ctx, r, tc.input, runOpts, executor, env)
		if conditionsMet != tc.want {
			t.Errorf("expected: %v, got: %v", tc.want, conditionsMet)
		}
		if errStringOrEmpty(err) != errStringOrEmpty(tc.wantErr) {
			t.Errorf("expected: %v, got: %v", tc.wantErr, err)
		}
	}
}

func TestExistsConditions(t *testing.T) {

	root := testDir()
	defer os.RemoveAll(root)

	// Create test directories and files as follows:
	//  <root>/src/my-component
	//  <root>/src/my-component/test.txt
	//  <root>/src/my-component/<random-subdirectory>
	componentDir, _ := testComponentDir(root, "my-component")
	testComponentFile(componentDir, "test.txt", "some contents here")
	subDir := path.Base(testDir(componentDir))

	ctx := context.Background()
	executor := NewBashExecutor()

	fs, err := NewFileSystem(root)
	if err != nil {
		t.Fatal(err)
	}

	c := &Component{
		name:         "my-component",
		componentDir: componentDir,
		relPath:      path.Join("src", "my-component"),
	}
	r := &Rule{
		component:  c,
		name:       "test-rule",
		inProvider: fs,
	}
	env := map[string]string{}

	type test struct {
		input   Condition
		want    bool
		wantErr error
	}

	tests := []test{
		{
			input:   Condition{DirectoryExists: subDir},
			want:    true,
			wantErr: nil,
		},
		{
			input:   Condition{DirectoryExists: "MISSING"},
			want:    false,
			wantErr: nil,
		},
		{
			input:   Condition{ResourceExists: "test.txt"},
			want:    true,
			wantErr: nil,
		},
		{
			input:   Condition{ResourceExists: "MISSING"},
			want:    false,
			wantErr: nil,
		},
	}

	for _, tc := range tests {
		var stdout bytes.Buffer
		runOpts := RunOpts{Output: &stdout, DebugOutput: &stdout, Debug: false}
		conditionsMet, err := CheckCondition(ctx, r, tc.input, runOpts, executor, env)
		if conditionsMet != tc.want {
			t.Errorf("expected: %v, got: %v", tc.want, conditionsMet)
		}
		if errStringOrEmpty(err) != errStringOrEmpty(tc.wantErr) {
			t.Errorf("expected: %v, got: %v", tc.wantErr, err)
		}
	}
}

func errStringOrEmpty(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
