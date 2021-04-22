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
	"bytes"
	"fmt"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

// simpleNode is used internally to present the graph.Node interface
// to the graph library used here
type simpleNode struct {
	id   int64
	node Node
}

func (n *simpleNode) ID() int64 {
	return n.id
}

func (n *simpleNode) unwrap() Node {
	return n.node
}

func wrap(n Node, id int64) *simpleNode {
	return &simpleNode{id: id, node: n}
}

// Edge is a simple graph edge.
type Edge struct {
	F, T Node
}

// From returns the from-node of the edge.
func (e Edge) From() Node { return e.F }

// To returns the to-node of the edge.
func (e Edge) To() Node { return e.T }

// Node defines NodeID() which is used to identify a Node in a Graph
type Node interface {
	NodeID() string
}

// Graph used to sort Components based on dependencies
type Graph struct {
	graph   *simple.DirectedGraph
	nodes   map[string]Node
	wrapped map[string]*simpleNode
	index   int64
}

// NewGraph returns an empty Graph
func NewGraph() *Graph {
	return &Graph{
		graph:   simple.NewDirectedGraph(),
		nodes:   map[string]Node{},
		wrapped: map[string]*simpleNode{},
	}
}

// Add a Node to the Graph
func (g *Graph) Add(n ...Node) *Graph {
	for _, node := range n {
		g.add(node)
	}
	return g
}

// Count returns the number of Nodes in the Graph
func (g *Graph) Count() int {
	return len(g.nodes)
}

func (g *Graph) add(n Node) *simpleNode {
	nodeID := n.NodeID()
	if wrapped, found := g.wrapped[nodeID]; found {
		return wrapped
	}
	g.index++
	wrapped := wrap(n, g.index)
	g.nodes[nodeID] = n
	g.wrapped[nodeID] = wrapped
	g.graph.AddNode(wrapped)
	return wrapped
}

// Remove a Node from the Graph
func (g *Graph) Remove(n ...Node) *Graph {
	for _, node := range n {
		nodeID := node.NodeID()
		wrapped, found := g.get(node)
		if found {
			g.graph.RemoveNode(wrapped.id)
			delete(g.nodes, nodeID)
			delete(g.wrapped, nodeID)
		}
	}
	return g
}

func (g *Graph) get(n Node) (*simpleNode, bool) {
	wrapped, found := g.wrapped[n.NodeID()]
	return wrapped, found
}

// Connect declares a directional link two nodes in the Graph
func (g *Graph) Connect(from, to Node) *Graph {
	f := g.add(from)
	t := g.add(to)
	g.graph.SetEdge(simple.Edge{F: f, T: t})
	return g
}

// GetNode returns the Node with the specified ID and a boolean indicating
// whether it was found
func (g *Graph) GetNode(nodeID string) (Node, bool) {
	if node, found := g.nodes[nodeID]; found {
		return node, true
	}
	return nil, false
}

// Sort returns a topological sort of the Graph
func (g *Graph) Sort() ([]Node, error) {
	sorted, err := topo.Sort(g.graph)
	if err != nil {
		return nil, err
	}
	resolved := make([]Node, len(sorted))
	for i, n := range sorted {
		resolved[i] = n.(*simpleNode).unwrap()
	}
	return resolved, nil
}

// From returns all nodes in the graph that can be reached directly from the
// specified Node
func (g *Graph) From(n Node) []Node {
	wrapped, ok := g.get(n)
	if !ok {
		return nil
	}
	return nodesFromIterator(g.graph.From(wrapped.ID()))
}

// To returns all nodes in the graph that can reach the specified node
func (g *Graph) To(n Node) []Node {
	wrapped, ok := g.get(n)
	if !ok {
		return nil
	}
	return nodesFromIterator(g.graph.To(wrapped.ID()))
}

// Visit all nodes in the Graph and call the provided callback for each
func (g *Graph) Visit(callback VisitCallback) {
	for _, n := range g.nodes {
		result := callback(n)
		// Return early if the callback responded with false
		if !result {
			return
		}
	}
}

// VisitCallback is a function signature for visiting nodes in the graph
type VisitCallback func(n Node) bool

// Filter nodes in the Graph
func (g *Graph) Filter(filter NodeFilter) (result []Node) {
	for _, n := range g.nodes {
		if filter(n) {
			result = append(result, n)
		}
	}
	return
}

// NodeFilter is a filter function for Graph nodes
type NodeFilter func(n Node) bool

func nodesFromIterator(iter graph.Nodes) []Node {
	nodes := make([]Node, 0, iter.Len())
	for iter.Next() {
		nodes = append(nodes, iter.Node().(*simpleNode).unwrap())
	}
	return nodes
}

func (g *Graph) GenerateDOT() []byte {
	var buf bytes.Buffer

	buf.WriteString("strict digraph {\n\n\t// Node definitions.\n")
	for k, v := range g.wrapped {
		buf.WriteString("\t" + fmt.Sprint(v.id) + "\t[label=\"" + k + "\"];\n")
	}

	buf.WriteString("\n\t// Edge definitions.\n")
	edges := g.graph.Edges()
	for edges.Next() {
		f := edges.Edge().From().ID()
		t := edges.Edge().To().ID()
		buf.WriteString("\t" + fmt.Sprint(f) + " -> " + fmt.Sprint(t) + ";\n")
	}

	buf.WriteString("}\n")
	return buf.Bytes()
}
