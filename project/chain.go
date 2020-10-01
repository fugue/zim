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

// Concept from:
// https://github.com/justinas/alice/blob/master/chain.go

// RunnerBuilder for creating Runner middleware
type RunnerBuilder func(Runner) Runner

// Chain of middleware
type Chain struct {
	constructors []RunnerBuilder
}

// NewChain creates a new Chain of middleware
func NewChain(constructors ...RunnerBuilder) Chain {
	return Chain{append(([]RunnerBuilder)(nil), constructors...)}
}

// Then chains the middleware and returns the final Runner.
//     NewChain(m1, m2, m3).Then(r)
// is equivalent to:
//     m1(m2(m3(r)))
// When the run call comes in, it will be passed to m1, then m2, then m3
// and finally, the given handler
// (assuming every middleware calls the following one).
func (c Chain) Then(r Runner) Runner {

	if r == nil {
		r = &StandardRunner{}
	}

	for i := range c.constructors {
		r = c.constructors[len(c.constructors)-1-i](r)
	}

	return r
}

// Append extends a chain, adding the specified constructors
// as the last ones in the request flow. A new Chain is returned
// and the original is left untouched.
func (c Chain) Append(constructors ...RunnerBuilder) Chain {
	newCons := make([]RunnerBuilder, 0, len(c.constructors)+len(constructors))
	newCons = append(newCons, c.constructors...)
	newCons = append(newCons, constructors...)
	return Chain{newCons}
}
