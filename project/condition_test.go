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

	// "when" condition is true
	build, found := comp.Rule("build-when-run")
	require.True(t, found)
	conditionsMet, err := CheckConditions(ctx, build, execOpts, executor)
	require.Nil(t, err)
	require.True(t, conditionsMet)

	// "when" condition is false
	build, found = comp.Rule("build-when-skip")
	require.True(t, found)
	conditionsMet, err = CheckConditions(ctx, build, execOpts, executor)
	require.Nil(t, err)
	require.False(t, conditionsMet)

	// "unless" condition is false (so the rule should run)
	build, found = comp.Rule("build-unless-run")
	require.True(t, found)
	conditionsMet, err = CheckConditions(ctx, build, execOpts, executor)
	require.Nil(t, err)
	require.True(t, conditionsMet)

	// "unless" condition is true (so the rule should NOT run)
	build, found = comp.Rule("build-unless-skip")
	require.True(t, found)
	conditionsMet, err = CheckConditions(ctx, build, execOpts, executor)
	require.Nil(t, err)
	require.False(t, conditionsMet)
}

func TestConditionScript(t *testing.T) {

	dir := testDir()
	defer os.RemoveAll(dir)
	ctx := context.Background()
	executor := NewBashExecutor()

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
		conditionsMet, err := CheckCondition(ctx, r, tc.input, runOpts, executor)
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
