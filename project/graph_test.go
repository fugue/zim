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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleRuleGraph(t *testing.T) {

	c := &Component{name: "foo"}

	rules := []*Rule{
		{component: c, name: "build"},
		{component: c, name: "test"},
	}

	g := GraphFromRules(rules)

	assert.Equal(t, g.Count(), 2, "Expected graph size of 2")

	buildRule, found := g.GetNode("foo.build")
	require.True(t, found)
	assert.Equal(t, buildRule.(*Rule), rules[0], "Expected build rule")

	testRule, found := g.GetNode("foo.test")
	require.True(t, found)
	assert.Equal(t, testRule.(*Rule), rules[1], "Expected test rule")
}

func TestConnectedRuleGraph(t *testing.T) {

	c := &Component{name: "foo"}

	rules := []*Rule{
		{component: c, name: "build"},
		{component: c, name: "test"},
	}
	// Make "build" depend on "test"
	rules[0].resolvedDeps = []*Rule{rules[1]}

	g := GraphFromRules(rules)
	assert.Equal(t, 2, g.Count(), "Expected graph size of 2")

	buildRule, found := g.GetNode("foo.build")
	require.True(t, found)

	fromNodes := g.From(buildRule)
	require.Len(t, fromNodes, 1)

	assert.Equal(t, "foo.test", fromNodes[0].(*Rule).NodeID())
}
