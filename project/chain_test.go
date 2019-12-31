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
