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
	"fmt"
	"io"
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

	// Get a plain bash executor: either the provided one, or a newly created
	// one if the provided one is dockerized.
	var bashExecutor Executor
	if opts.Executor != nil && !opts.Executor.UsesDocker() {
		bashExecutor = opts.Executor
	} else {
		bashExecutor = NewBashExecutor()
	}

	// Determine the primary executor to be used for "run" commands.
	// Use the provided one unless it conflicts in terms of native vs. docker.
	var primaryExecutor Executor
	if opts.Executor == nil || (r.IsNative() && opts.Executor.UsesDocker()) {
		primaryExecutor = bashExecutor
	} else {
		primaryExecutor = opts.Executor
	}

	// Evaluate rule conditions which could lead to rule execution being skipped.
	// Any scripting done to check the condition will be via the bash executor.
	conditionsMet, err := CheckConditions(ctx, r, opts, bashExecutor)
	if err != nil {
		return Error, fmt.Errorf("Error checking conditions on rule %s: %s", r.NodeID(), err)
	}
	if !conditionsMet {
		return Skipped, nil
	}

	// Generate the bash environment variables to be available to the execution
	bashEnv, err := r.Environment()
	if err != nil {
		return Error, fmt.Errorf("Environment error %s: %s", r.NodeID(), err)
	}
	if err := runner.setArtifactVariables(r, bashExecutor, bashEnv); err != nil {
		return Error, err
	}
	// Generate a second set of environment variables for the primary executor.
	// This supports the primary executor being dockerized, in which case the
	// ARTIFACTS_DIR and ARTIFACT variables differ due to absolute paths changing.
	primaryEnv := copyEnvironment(bashEnv)
	if err := runner.setArtifactVariables(r, primaryExecutor, primaryEnv); err != nil {
		return Error, err
	}

	// Execute each of the rule's commands
	for i, cmd := range r.Commands() {
		env := bashEnv
		exec := bashExecutor
		if cmd.Kind == "run" {
			// Run commands use the primary environment and executor
			env = primaryEnv
			exec = primaryExecutor
		}
		execOpts := ExecOpts{
			WorkingDirectory: r.Component().Directory(),
			Env:              flattenEnvironment(env),
			Stdout:           opts.Output,
			Stderr:           opts.Output,
			Debug:            opts.Debug,
			Cmdout:           opts.DebugOutput,
			Image:            r.Image(),
			Name:             fmt.Sprintf("%s.%d", r.NodeID(), i),
		}
		// Run the command
		var execError error
		switch cmd.Kind {
		case "run":
			execError = runner.execRunCommand(ctx, r, exec, execOpts, cmd)
		case "zip":
			execError = runner.execZipCommand(ctx, r, exec, execOpts, cmd)
		case "archive":
			execError = runner.execArchiveCommand(ctx, r, exec, execOpts, cmd)
		case "mkdir":
			execError = runner.execMkdirCommand(ctx, r, exec, execOpts, cmd)
		case "cleandir":
			execError = runner.execCleandirCommand(ctx, r, exec, execOpts, cmd)
		case "remove":
			execError = runner.execRemoveCommand(ctx, r, exec, execOpts, cmd)
		case "move":
			execError = runner.execMoveCommand(ctx, r, exec, execOpts, cmd)
		case "copy":
			execError = runner.execCopyCommand(ctx, r, exec, execOpts, cmd)
		default:
			return Error, fmt.Errorf("Unknown command kind in %s: %s",
				r.NodeID(), cmd.Kind)
		}
		if execError != nil {
			return ExecError, fmt.Errorf("Error running rule command. Rule: %s. Error: %s",
				r.NodeID(), execError)
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

// Add absolute paths within the executor to the artifacts directory for
// this rule and the rule output artifact, if there is one.
func (runner *StandardRunner) setArtifactVariables(
	r *Rule,
	executor Executor,
	env map[string]string,
) error {
	ruleArtifactsDir, err := executor.ExecutorPath(r.ArtifactsDir())
	if err != nil {
		return err
	}
	env["ARTIFACTS_DIR"] = ruleArtifactsDir
	ruleOutputs := r.Outputs()
	if len(ruleOutputs) > 0 {
		firstRuleOutput := ruleOutputs[0]
		if firstRuleOutput.OnFilesystem() {
			artifactPath, err := executor.ExecutorPath(firstRuleOutput.Path())
			if err != nil {
				return err
			}
			env["ARTIFACT"] = artifactPath
		}
	}
	return nil
}

// Runs commands in bash
func (runner *StandardRunner) execRunCommand(
	ctx context.Context,
	r *Rule,
	executor Executor,
	execOpts ExecOpts,
	cmd *Command,
) error {
	execOpts.Command = cmd.Argument
	return executor.Execute(ctx, execOpts)
}

// Creates a zip file with the specified contents. By default, the options
// `-qrFS` are used. The `cd` attribute may be used to change into the specified
// directory before running the command.
func (runner *StandardRunner) execZipCommand(
	ctx context.Context,
	r *Rule,
	executor Executor,
	execOpts ExecOpts,
	cmd *Command,
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
	execOpts.Command = script
	return executor.Execute(ctx, execOpts)
}

// Runs the `tar` command to create an archive using gzip if the default options
// are used. Equivalent to `tar -czf $OUTPUT $INPUT`.
func (runner *StandardRunner) execArchiveCommand(
	ctx context.Context,
	r *Rule,
	executor Executor,
	execOpts ExecOpts,
	cmd *Command,
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
	execOpts.Command = fmt.Sprintf("tar %s %s %s", opts, output, input)
	return executor.Execute(ctx, execOpts)
}

// Creates a directory and its parents as needed. Equivalent to `mkdir -p`.
func (runner *StandardRunner) execMkdirCommand(
	ctx context.Context,
	r *Rule,
	executor Executor,
	execOpts ExecOpts,
	cmd *Command,
) error {
	arg := strings.TrimSpace(cmd.Argument)
	if arg == "" {
		return fmt.Errorf("mkdir command has no targets specified")
	}
	execOpts.Command = fmt.Sprintf("mkdir -p %s", arg)
	return executor.Execute(ctx, execOpts)
}

// Ensures an empty directory with the given name exists. If the directory
// existed previously it is first removed. Useful for build temp directories.
// Equivalent to `rm -rf $DIR && mkdir -p $DIR`.
func (runner *StandardRunner) execCleandirCommand(
	ctx context.Context,
	r *Rule,
	executor Executor,
	execOpts ExecOpts,
	cmd *Command,
) error {
	arg := strings.TrimSpace(cmd.Argument)
	if arg == "" {
		return fmt.Errorf("cleandir command has no targets specified")
	}
	if arg == "/" {
		return fmt.Errorf("cleandir cannot run against /")
	}
	execOpts.Command = fmt.Sprintf("rm -rf %s && mkdir -p %s", arg, arg)
	return executor.Execute(ctx, execOpts)
}

// Remove one or more files or directories. Equivalent to `rm -rf`.
func (runner *StandardRunner) execRemoveCommand(
	ctx context.Context,
	r *Rule,
	executor Executor,
	execOpts ExecOpts,
	cmd *Command,
) error {
	arg := strings.TrimSpace(cmd.Argument)
	if arg == "" {
		return fmt.Errorf("remove command has no targets specified")
	}
	execOpts.Command = fmt.Sprintf("rm -rf %s", arg)
	return executor.Execute(ctx, execOpts)
}

// Move one or more files or directories.
func (runner *StandardRunner) execMoveCommand(
	ctx context.Context,
	r *Rule,
	executor Executor,
	execOpts ExecOpts,
	cmd *Command,
) error {
	src := getCommandAttr(cmd, "src", "")
	dst := getCommandAttr(cmd, "dst", "")
	if src == "" {
		return fmt.Errorf("Move command has no src specified")
	}
	if dst == "" {
		return fmt.Errorf("Move command has no dst specified")
	}
	execOpts.Command = fmt.Sprintf("mv %s %s", src, dst)
	return executor.Execute(ctx, execOpts)
}

// Copy one or more files or directories.
func (runner *StandardRunner) execCopyCommand(
	ctx context.Context,
	r *Rule,
	executor Executor,
	execOpts ExecOpts,
	cmd *Command,
) error {
	src := getCommandAttr(cmd, "src", "")
	dst := getCommandAttr(cmd, "dst", "")
	if src == "" {
		return fmt.Errorf("Copy command has no src specified")
	}
	if dst == "" {
		return fmt.Errorf("Copy command has no dst specified")
	}
	execOpts.Command = fmt.Sprintf("cp %s %s", src, dst)
	return executor.Execute(ctx, execOpts)
}

func getCommandAttr(cmd *Command, attr, defaultValue string) string {
	value, ok := cmd.Attributes[attr].(string)
	if !ok {
		return defaultValue
	}
	return value
}
