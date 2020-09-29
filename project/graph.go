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
	"github.com/fugue/zim/graph"
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
