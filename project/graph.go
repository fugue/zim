package project

import (
	"github.com/LuminalHQ/zim/graph"
)

// GraphFromRules builds a dependency graph originating from the specified
// Rules. The returned Graph contains the specified Rules plus all their
// direct and transitive dependencies.
func GraphFromRules(rules []*Rule) *graph.Graph {

	visited := map[*Rule]bool{}

	g := graph.NewGraph()
	for _, r := range rules {
		addToGraph(g, r, visited)
	}

	return g
}

// Recursive function used to add a rule and its dependencies to a graph
func addToGraph(g *graph.Graph, r *Rule, visited map[*Rule]bool) {

	if visited[r] {
		return
	}

	g.Add(r)
	for _, dep := range r.Dependencies() {
		g.Add(dep)
		g.Connect(r, dep)
		addToGraph(g, dep, visited)
	}
}
