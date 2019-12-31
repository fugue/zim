package project

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSimpleRuleGraph(t *testing.T) {

	c := &Component{name: "foo"}

	rules := []*Rule{
		&Rule{component: c, name: "build"},
		&Rule{component: c, name: "test"},
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
		&Rule{component: c, name: "build"},
		&Rule{component: c, name: "test"},
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
