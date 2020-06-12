package project

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
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
}

// NewBashExecutor returns an Executor that runs commands via bash
func NewBashExecutor() Executor {
	return &bashExecutor{}
}

type bashExecutor struct{}

// Execute runs a command in a subprocess
func (e *bashExecutor) Execute(ctx context.Context, opts ExecOpts) error {

	environment := append(os.Environ(), opts.Env...)

	// TODO

	// Replace newlines with semicolons if the command is multiline
	commands := strings.Split(strings.TrimSpace(opts.Command), "\n")
	commandText := strings.Join(commands, "; ")

	workingDir := opts.WorkingDirectory
	if workingDir == "" {
		workingDir = "."
	}

	bashCmd := exec.CommandContext(ctx, "bash", "-e", "-c", commandText)
	bashCmd.Env = environment
	bashCmd.Dir = workingDir
	bashCmd.Stdout = getWriter(opts.Stdout, os.Stdout)
	bashCmd.Stderr = getWriter(opts.Stderr, os.Stderr)

	// Show the command to be executed to the user
	cmdOut := getWriter(opts.Cmdout, os.Stdout)
	if opts.Debug {
		debugColor := color.New(color.FgYellow).SprintFunc()
		fmt.Fprintln(cmdOut, "dbg:", debugColor(bashCmd.Args))
	}
	cmdColor := color.New(color.FgMagenta).SprintFunc()
	fmt.Fprintln(cmdOut, "cmd:", cmdColor(commandText))

	return bashCmd.Run()
}

func (e *bashExecutor) UsesDocker() bool {
	return false
}

func getWriter(override, def io.Writer) io.Writer {
	if override != nil {
		return override
	}
	return def
}
