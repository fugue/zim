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
package graph

import (
	"reflect"
	"testing"
)

type testNode struct {
	id string
}

func (n *testNode) NodeID() string { return n.id }

func TestGraphBasics(t *testing.T) {

	a := &testNode{"a"}
	b := &testNode{"b"}
	c := &testNode{"c"}

	g := NewGraph()
	g.Add(a).Add(b).Add(c)

	// Duplicate call ignored
	g.Add(c)

	g.Connect(c, b)
	g.Connect(b, a)

	nodes, err := g.Sort()
	if err != nil {
		t.Fatal(err)
	}

	expected := []Node{c, b, a}

	if !reflect.DeepEqual(expected, nodes) {
		t.Error("Sort failed", nodes)
	}
}

func TestAnotherGraph(t *testing.T) {
	g := NewGraph()

	a := &testNode{"a"}
	b := &testNode{"b"}
	c := &testNode{"c"}
	d := &testNode{"d"}
	e := &testNode{"e"}

	g.Add(a, b, c, d, e)

	// Nodes can be added implicitly via edges as well
	g.Connect(b, c)
	g.Connect(a, d)
	g.Connect(c, e)
	g.Connect(d, e)

	// b -> c
	//        \
	//          -> e
	//        /
	// a -> d

	nodes, err := g.Sort()
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 5 {
		t.Error("Unexpected result length")
	}

	aIdx := nodeIndex(a, nodes)
	bIdx := nodeIndex(b, nodes)
	cIdx := nodeIndex(c, nodes)
	dIdx := nodeIndex(d, nodes)
	eIdx := nodeIndex(e, nodes)

	if eIdx != 4 {
		t.Error("Incorrect sort (e):", eIdx)
	}
	if cIdx < bIdx {
		t.Error("Incorrect sort (c < b):", cIdx, bIdx)
	}
	if dIdx < aIdx {
		t.Error("Incorrect sort (d < a):", dIdx, aIdx)
	}
}

func TestGraphFrom(t *testing.T) {

	g := NewGraph()

	a := &testNode{"a"}
	b := &testNode{"b"}
	c := &testNode{"c"}

	g.Connect(a, c)
	g.Connect(b, c)

	if !reflect.DeepEqual(g.From(a), []Node{c}) {
		t.Error("Expected connection to C")
	}
	if !reflect.DeepEqual(g.From(b), []Node{c}) {
		t.Error("Expected connection to C")
	}
	if !reflect.DeepEqual(g.From(c), []Node{}) {
		t.Error("Expected nil connection")
	}
}

func TestGraphFilter(t *testing.T) {

	g := NewGraph()

	a := &testNode{"a"}
	b := &testNode{"b"}
	c := &testNode{"c"}

	g.Connect(c, b)
	g.Connect(b, a)

	nodes := g.Filter(func(n Node) bool {
		return n.NodeID() == "b"
	})

	if !reflect.DeepEqual(nodes, []Node{b}) {
		t.Error("Expected b only")
	}
}

func TestGraphRemove(t *testing.T) {

	g := NewGraph()

	a := &testNode{"a"}
	b := &testNode{"b"}
	c := &testNode{"c"}

	g.Connect(c, b)
	g.Connect(b, a)

	if len(g.From(b)) != 1 {
		t.Error("Expected 1 edge from b")
	}
	if len(g.To(a)) != 1 {
		t.Error("Expected 1 edge to a")
	}
	if g.Count() != 3 {
		t.Error("Expected graph count to be 3")
	}
	if _, found := g.GetNode("a"); !found {
		t.Error("Expected a to be found")
	}

	g.Remove(a)

	if len(g.From(b)) != 0 {
		t.Error("Expected 0 edges from b to remain")
	}
	if _, found := g.GetNode("a"); found {
		t.Error("Expected a not to be found")
	}
	if g.Count() != 2 {
		t.Error("Expected graph count to be 2")
	}
}

func nodeIndex(n Node, slice []Node) int {
	for i, value := range slice {
		if value == n {
			return i
		}
	}
	return -1
}
