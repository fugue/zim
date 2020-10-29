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
