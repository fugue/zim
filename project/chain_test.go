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
	"context"
	"errors"
	"testing"
)

type testRunner struct {
	wrapped Runner
	id      string
	done    *bool
}

func (tr *testRunner) Run(ctx context.Context, r *Rule, opts RunOpts) (Code, error) {
	*tr.done = true
	return tr.wrapped.Run(ctx, r, opts)
}

func TestChain(t *testing.T) {

	var done1, done2 bool

	builder1 := RunnerBuilder(func(r Runner) Runner {
		return &testRunner{wrapped: r, id: "1", done: &done1}
	})
	builder2 := RunnerBuilder(func(r Runner) Runner {
		return &testRunner{wrapped: r, id: "2", done: &done2}
	})

	chain := NewChain(builder1, builder2).
		Then(RunnerFunc(func(ctx context.Context, rule *Rule, opts RunOpts) (Code, error) {
			if rule.Name() != "bar" {
				t.Fatal("Incorrect rule")
			}
			return ExecError, errors.New("boom")
		}))

	ctx := context.Background()
	rule := &Rule{name: "bar"}
	code, err := chain.Run(ctx, rule, RunOpts{})

	if code != ExecError {
		t.Error("Incorrect code:", code)
	}
	if err.Error() != "boom" {
		t.Error("Incorrect err:", err)
	}

	if !done1 {
		t.Error("Middleware 1 did not execute")
	}
	if !done2 {
		t.Error("Middleware 2 did not execute")
	}
}
