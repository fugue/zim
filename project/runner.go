package project

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/fatih/color"
	"github.com/hashicorp/go-multierror"
)

var cmdColor *color.Color

func init() {
	cmdColor = color.New(color.FgMagenta)
}

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

	if opts.Executor == nil {
		// Default to a simple bash executor if another one is not specified
		opts.Executor = NewBashExecutor()
	} else if opts.Executor.UsesDocker() && r.native {
		// The rule is set to "native" which means it is opting out of docker
		opts.Executor = NewBashExecutor()
	}

	// Determine the bash environment variables to be available to the execution
	env, err := r.Environment()
	if err != nil {
		return Error, fmt.Errorf("Environment error %s: %s", r.NodeID(), err)
	}

	// Execute the rule's commands one at a time
	for i, cmd := range r.Commands() {
		var execError error
		switch cmd.Kind {
		case "run":
			execError = runner.execRunCommand(ctx, r, opts, env, cmd, i)
		case "zip":
			execError = runner.execZipCommand(ctx, r, opts, env, cmd, i)
		case "archive":
			execError = runner.execArchiveCommand(ctx, r, opts, env, cmd, i)
		case "mkdir":
			execError = runner.execMkdirCommand(ctx, r, opts, env, cmd, i)
		case "cleandir":
			execError = runner.execCleandirCommand(ctx, r, opts, env, cmd, i)
		case "remove":
			execError = runner.execRemoveCommand(ctx, r, opts, env, cmd, i)
		default:
			return Error, fmt.Errorf("Unknown command kind in %s: %s",
				r.NodeID(), cmd.Kind)
		}
		if execError != nil {
			return ExecError, execError
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

// Runs commands in bash
func (runner *StandardRunner) execRunCommand(
	ctx context.Context,
	r *Rule,
	runOpts RunOpts,
	env map[string]string,
	cmd *Command,
	cmdIndex int,
) error {

	execOpts := ExecOpts{
		Command:          cmd.Argument,
		WorkingDirectory: r.Component().Directory(),
		Env:              flattenEnvironment(env),
		Stdout:           runOpts.Output,
		Stderr:           runOpts.Output,
		Image:            r.Image(),
		Debug:            runOpts.Debug,
		Cmdout:           runOpts.DebugOutput,
		Name:             fmt.Sprintf("%s.%d", r.NodeID(), cmdIndex),
	}
	if err := runOpts.Executor.Execute(ctx, execOpts); err != nil {
		return fmt.Errorf("Exec error %s: %s", r.NodeID(), err)
	}
	return nil
}

// Creates a zip file with the specified contents. By default, the options
// `-qrFS` are used. The `cd` attribute may be used to change into the specified
// directory before running the command.
func (runner *StandardRunner) execZipCommand(
	ctx context.Context,
	r *Rule,
	runOpts RunOpts,
	env map[string]string,
	cmd *Command,
	cmdIndex int,
) error {

	opts := getCommandAttr(cmd, "options", "-qrFS")
	input := getCommandAttr(cmd, "input", ".")
	output := getCommandAttr(cmd, "output", "")
	dir := getCommandAttr(cmd, "cd", "")

	if output == "" {
		return fmt.Errorf("Zip command has no output specified")
	}
	script := fmt.Sprintf("zip %s %s %s", opts, output, input)
	if dir != "" {
		script = fmt.Sprintf("cd %s && %s", dir, script)
	}
	buffer := &bytes.Buffer{}

	if err := NewBashExecutor().Execute(ctx, ExecOpts{
		Command:          script,
		WorkingDirectory: r.Component().Directory(),
		Env:              flattenEnvironment(env),
		Stdout:           runOpts.Output,
		Stderr:           runOpts.Output,
		Debug:            runOpts.Debug,
		Cmdout:           buffer,
	}); err != nil {
		return err
	}
	fmt.Fprintln(runOpts.Output, "cmd:",
		cmdColor.Sprintf("[zip] created %s", output))
	return nil
}

// Runs the `tar` command to create an archive using gzip if the default options
// are used. Equivalent to `tar -czf $OUTPUT $INPUT`.
func (runner *StandardRunner) execArchiveCommand(
	ctx context.Context,
	r *Rule,
	runOpts RunOpts,
	env map[string]string,
	cmd *Command,
	cmdIndex int,
) error {

	opts := getCommandAttr(cmd, "options", "-czf")
	input := getCommandAttr(cmd, "input", "")
	output := getCommandAttr(cmd, "output", "")

	if input == "" {
		return fmt.Errorf("Archive command has no input specified")
	}
	if output == "" {
		return fmt.Errorf("Archive command has no output specified")
	}
	script := fmt.Sprintf("tar %s %s %s", opts, output, input)
	buffer := &bytes.Buffer{}

	if err := NewBashExecutor().Execute(ctx, ExecOpts{
		Command:          script,
		WorkingDirectory: r.Component().Directory(),
		Env:              flattenEnvironment(env),
		Stdout:           runOpts.Output,
		Stderr:           runOpts.Output,
		Debug:            runOpts.Debug,
		Cmdout:           buffer,
	}); err != nil {
		return err
	}
	fmt.Fprintln(runOpts.Output, "cmd:",
		cmdColor.Sprintf("[archive] created %s", output))
	return nil
}

// Creates a directory and its parents as needed. Equivalent to `mkdir -p`.
func (runner *StandardRunner) execMkdirCommand(
	ctx context.Context,
	r *Rule,
	runOpts RunOpts,
	env map[string]string,
	cmd *Command,
	cmdIndex int,
) error {
	arg := substituteVars(strings.TrimSpace(cmd.Argument), env)
	if arg == "" {
		return fmt.Errorf("mkdir command has no directory specified")
	}
	var dir string
	if !path.IsAbs(dir) {
		dir = path.Join(r.Component().Directory(), dir)
	} else {
		dir = arg
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	fmt.Fprintln(runOpts.Output, "cmd:", cmdColor.Sprintf("[mkdir] %s", arg))
	return nil
}

// Ensures an empty directory with the given name exists. If the directory
// existed previously it is first removed. Useful for build temp directories.
// Equivalent to `rm -rf $DIR && mkdir -p $DIR`.
func (runner *StandardRunner) execCleandirCommand(
	ctx context.Context,
	r *Rule,
	runOpts RunOpts,
	env map[string]string,
	cmd *Command,
	cmdIndex int,
) error {
	arg := substituteVars(strings.TrimSpace(cmd.Argument), env)
	if arg == "" {
		return fmt.Errorf("cleandir command has no directory specified")
	}
	if arg == "/" {
		return fmt.Errorf("cleandir cannot run against /")
	}
	var dir string
	if !path.IsAbs(dir) {
		dir = path.Join(r.Component().Directory(), arg)
	} else {
		dir = arg
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	fmt.Fprintln(runOpts.Output, "cmd:", cmdColor.Sprintf("[cleandir] %s", arg))
	return nil
}

// Remove one or more files or directories. Equivalent to `rm -rf`.
func (runner *StandardRunner) execRemoveCommand(
	ctx context.Context,
	r *Rule,
	runOpts RunOpts,
	env map[string]string,
	cmd *Command,
	cmdIndex int,
) error {
	arg := substituteVars(strings.TrimSpace(cmd.Argument), env)
	if arg == "" {
		return fmt.Errorf("rm command has no targets specified")
	}
	for _, target := range strings.Split(arg, " ") {
		if target == "/" {
			return fmt.Errorf("rm cannot run against /")
		}
		if !path.IsAbs(target) {
			target = path.Join(r.Component().Directory(), target)
		}
		if err := os.RemoveAll(target); err != nil {
			return err
		}
	}
	fmt.Fprintln(runOpts.Output, "cmd:", cmdColor.Sprintf("[remove] %s", arg))
	return nil
}

func getCommandAttr(cmd *Command, attr, defaultValue string) string {
	value, ok := cmd.Attributes[attr].(string)
	if !ok {
		return defaultValue
	}
	return value
}
