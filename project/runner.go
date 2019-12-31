package project

import (
	"context"
	"fmt"
	"io"

	"github.com/hashicorp/go-multierror"
)

// RunOpts contains options used to configure the running of Rules
type RunOpts struct {
	BuildID     string
	Executor    Executor
	Output      io.Writer
	DebugOutput io.Writer
	Debug       bool
}

// Runner is an interface used to run Rules. Different implementations may
// decorate the standard behavior.
type Runner interface {

	// Run a Rule
	Run(context.Context, *Rule, RunOpts) (Code, error)
}

// The RunnerFunc type is an adapter to allow the use of ordinary functions as
// Runners. This mimics http.HandlerFunc from the standard library.
type RunnerFunc func(context.Context, *Rule, RunOpts) (Code, error)

// Run calls f(w, r).
func (f RunnerFunc) Run(ctx context.Context, r *Rule, opts RunOpts) (Code, error) {
	return f(ctx, r, opts)
}

// StandardRunner defines good default behavior for running a Rule
type StandardRunner struct{}

// Run a rule with the provided executor and other options
func (runner *StandardRunner) Run(ctx context.Context, r *Rule, opts RunOpts) (Code, error) {

	// Default to a simple bash executor if another one is not specified
	if opts.Executor == nil {
		opts.Executor = NewBashExecutor()
	}

	// Determine the bash environment variables to be available to the execution
	env, err := r.Environment()
	if err != nil {
		return Error, fmt.Errorf("Environment error %s: %s", r.NodeID(), err)
	}

	// Execute the rule's commands one at a time. Each command will have its
	// own bash shell if using the bash executor.
	for i, cmd := range r.Commands() {
		err := opts.Executor.Execute(ctx, ExecOpts{
			Name:             fmt.Sprintf("%s.%d", r.NodeID(), i),
			Command:          cmd,
			WorkingDirectory: r.Component().Directory(),
			Env:              flattenEnvironment(env),
			Stdout:           opts.Output,
			Stderr:           opts.Output,
			Image:            r.Image(),
			Debug:            opts.Debug,
			Cmdout:           opts.DebugOutput,
		})
		if err != nil {
			return ExecError, fmt.Errorf("Exec error %s: %s", r.NodeID(), err)
		}
	}

	// At this point the commands were all successful. If the rule defines
	// outputs but they were not created, this is an error.
	var errs *multierror.Error
	for _, output := range r.MissingOutputs() {
		errs = multierror.Append(errs,
			fmt.Errorf("Rule %s failed to create output %s",
				Bright(r.NodeID()), Bright(output)))
	}
	if errs != nil {
		return MissingOutputError, errs.ErrorOrNil()
	}
	return OK, nil
}
