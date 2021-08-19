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

package exec

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

// ExecOpts are options used to run a command
type ExecOpts struct {
	Name             string
	Command          string
	WorkingDirectory string
	Stdout           io.Writer
	Stderr           io.Writer
	Cmdout           io.Writer
	Env              []string
	Image            string
	Debug            bool
}

// Executor is an interface for executing commands
type Executor interface {

	// Execute a command
	Execute(ctx context.Context, opts ExecOpts) error

	// UsesDocker indicates whether this executor runs commands in a container
	UsesDocker() bool

	// ExecutorPath returns the corresponding path within the executor
	// environment that corresponds to the given path on the host. For
	// simple executors this will be identical. For executors that run
	// inside Docker containers, that path will be translated to the
	// path within the container filesystem. The provided path must be
	// an absolute path on the host, and the returned path is also an
	// absolute path. An error is returned if a relative path is provided
	// or if the host path is not mapped within the executor.
	ExecutorPath(hostPath string) (string, error)
}

// NewBashExecutor returns an Executor that runs commands via bash
func NewBashExecutor() Executor {
	return &bashExecutor{}
}

type bashExecutor struct{}

// Execute runs a command in a subprocess
func (e *bashExecutor) Execute(ctx context.Context, opts ExecOpts) error {

	environment := append(os.Environ(), opts.Env...)

	workingDir := opts.WorkingDirectory
	if workingDir == "" {
		workingDir = "."
	}

	// Create list of bash options
	args := []string{"-e"}
	if opts.Debug {
		args = extendSlice(args, "-x")
	}

	bashCmd := exec.CommandContext(ctx, "bash", args...)
	bashCmd.Env = environment
	bashCmd.Dir = workingDir
	bashCmd.Stdout = getWriter(opts.Stdout, os.Stdout)
	bashCmd.Stderr = getWriter(opts.Stderr, os.Stderr)

	stdin, err := bashCmd.StdinPipe()
	if err != nil {
		return err
	}

	// Write command to the process' stdin.
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, opts.Command)
	}()

	// Show the command to be executed to the user
	cmdOut := getWriter(opts.Cmdout, os.Stdout)
	if opts.Debug {
		debugColor := color.New(color.FgYellow).SprintFunc()
		fmt.Fprintln(cmdOut, "dbg:", debugColor(strings.Join(bashCmd.Args, " ")))
	}
	cmdColor := color.New(color.FgMagenta).SprintFunc()
	fmt.Fprintln(cmdOut, "cmd:", cmdColor(opts.Command))

	return bashCmd.Run()
}

func (e *bashExecutor) UsesDocker() bool {
	return false
}

func (e *bashExecutor) ExecutorPath(hostPath string) (string, error) {
	if !filepath.IsAbs(hostPath) {
		return "", fmt.Errorf("A relative path was incorrectly passed: %s", hostPath)
	}
	return hostPath, nil
}

func getWriter(override, def io.Writer) io.Writer {
	if override != nil {
		return override
	}
	return def
}

// XDGCache returns the local cache directory
func XDGCache() string {
	value := os.Getenv("XDG_CACHE_HOME")
	if value != "" {
		return value
	}
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return path.Join(home, ".cache")
}
