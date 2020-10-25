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

	defaultBashExecutor := NewBashExecutor()

	// Determine whether the rule should run based on its conditions
	conditionsMet, err := r.CheckConditions(ctx, defaultBashExecutor)
	if err != nil {
		return Error, fmt.Errorf("Error checking conditions on rule %s: %s", r.NodeID(), err)
	}
	if !conditionsMet {
		return Skipped, nil
	}

	// Determine which executor to use. Prefer the provided one but if the provided
	// one is Docker-enabled and the rule is native, use a default bash executor instead.
	executor := opts.Executor
	if executor == nil || r.native && executor.UsesDocker() {
		executor = defaultBashExecutor
	}
	isDockerized := !r.native && executor.UsesDocker()

	// Determine the bash environment variables to be available to the execution
	bashEnv, err := r.Environment()
	if err != nil {
		return Error, fmt.Errorf("Environment error %s: %s", r.NodeID(), err)
	}
	// Environment variables are adjusted slightly when the rule runs in Docker.
	// Set ARTIFACTS_DIR and ARTIFACT environment variables.
	var dockerEnv map[string]string
	if isDockerized {
		dockerEnv = copyEnvironment(bashEnv)
		if err := runner.setArtifactVariables(r, executor, dockerEnv); err != nil {
			return Error, err
		}
	}
	if err := runner.setArtifactVariables(r, defaultBashExecutor, bashEnv); err != nil {
		return Error, err
	}

	// Execute each of the rule's commands
	for i, cmd := range r.Commands() {
		// Choose which environment variables depending on whether Docker is used
		env := bashEnv
		if cmd.Kind == "run" && isDockerized {
			env = dockerEnv
		}
		// Run the command
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
		case "move":
			execError = runner.execMoveCommand(ctx, r, opts, env, cmd, i)
		case "copy":
			execError = runner.execCopyCommand(ctx, r, opts, env, cmd, i)
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

	return NewBashExecutor().Execute(ctx, ExecOpts{
		Command:          script,
		WorkingDirectory: r.Component().Directory(),
		Env:              flattenEnvironment(env),
		Stdout:           runOpts.Output,
		Stderr:           runOpts.Output,
		Debug:            runOpts.Debug,
		Cmdout:           runOpts.DebugOutput,
	})
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

	return NewBashExecutor().Execute(ctx, ExecOpts{
		Command:          script,
		WorkingDirectory: r.Component().Directory(),
		Env:              flattenEnvironment(env),
		Stdout:           runOpts.Output,
		Stderr:           runOpts.Output,
		Debug:            runOpts.Debug,
		Cmdout:           runOpts.DebugOutput,
	})
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

	arg := strings.TrimSpace(cmd.Argument)
	if arg == "" {
		return fmt.Errorf("mkdir command has no targets specified")
	}
	script := fmt.Sprintf("mkdir -p %s", arg)

	return NewBashExecutor().Execute(ctx, ExecOpts{
		Command:          script,
		WorkingDirectory: r.Component().Directory(),
		Env:              flattenEnvironment(env),
		Stdout:           runOpts.Output,
		Stderr:           runOpts.Output,
		Debug:            runOpts.Debug,
		Cmdout:           runOpts.DebugOutput,
	})
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

	arg := strings.TrimSpace(cmd.Argument)
	if arg == "" {
		return fmt.Errorf("cleandir command has no targets specified")
	}
	if arg == "/" {
		return fmt.Errorf("cleandir cannot run against /")
	}
	script := fmt.Sprintf("rm -rf %s && mkdir -p %s", arg, arg)

	return NewBashExecutor().Execute(ctx, ExecOpts{
		Command:          script,
		WorkingDirectory: r.Component().Directory(),
		Env:              flattenEnvironment(env),
		Stdout:           runOpts.Output,
		Stderr:           runOpts.Output,
		Debug:            runOpts.Debug,
		Cmdout:           runOpts.DebugOutput,
	})
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

	arg := strings.TrimSpace(cmd.Argument)
	if arg == "" {
		return fmt.Errorf("remove command has no targets specified")
	}
	script := fmt.Sprintf("rm -rf %s", arg)

	return NewBashExecutor().Execute(ctx, ExecOpts{
		Command:          script,
		WorkingDirectory: r.Component().Directory(),
		Env:              flattenEnvironment(env),
		Stdout:           runOpts.Output,
		Stderr:           runOpts.Output,
		Debug:            runOpts.Debug,
		Cmdout:           runOpts.DebugOutput,
	})
}

// Move one or more files or directories.
func (runner *StandardRunner) execMoveCommand(
	ctx context.Context,
	r *Rule,
	runOpts RunOpts,
	env map[string]string,
	cmd *Command,
	cmdIndex int,
) error {

	src := getCommandAttr(cmd, "src", "")
	dst := getCommandAttr(cmd, "dst", "")

	if src == "" {
		return fmt.Errorf("Move command has no src specified")
	}
	if dst == "" {
		return fmt.Errorf("Move command has no dst specified")
	}
	script := fmt.Sprintf("mv %s %s", src, dst)

	return NewBashExecutor().Execute(ctx, ExecOpts{
		Command:          script,
		WorkingDirectory: r.Component().Directory(),
		Env:              flattenEnvironment(env),
		Stdout:           runOpts.Output,
		Stderr:           runOpts.Output,
		Debug:            runOpts.Debug,
		Cmdout:           runOpts.DebugOutput,
	})
}

// Copy one or more files or directories.
func (runner *StandardRunner) execCopyCommand(
	ctx context.Context,
	r *Rule,
	runOpts RunOpts,
	env map[string]string,
	cmd *Command,
	cmdIndex int,
) error {

	src := getCommandAttr(cmd, "src", "")
	dst := getCommandAttr(cmd, "dst", "")

	if src == "" {
		return fmt.Errorf("Copy command has no src specified")
	}
	if dst == "" {
		return fmt.Errorf("Copy command has no dst specified")
	}
	script := fmt.Sprintf("cp %s %s", src, dst)

	return NewBashExecutor().Execute(ctx, ExecOpts{
		Command:          script,
		WorkingDirectory: r.Component().Directory(),
		Env:              flattenEnvironment(env),
		Stdout:           runOpts.Output,
		Stderr:           runOpts.Output,
		Debug:            runOpts.Debug,
		Cmdout:           runOpts.DebugOutput,
	})
}

func getCommandAttr(cmd *Command, attr, defaultValue string) string {
	value, ok := cmd.Attributes[attr].(string)
	if !ok {
		return defaultValue
	}
	return value
}
