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
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

// DefaultDockerExecutorDir is the path to the execution root within the
// Docker container
const DefaultDockerExecutorDir = "/build"

// NewDockerExecutor returns an Executor that runs commands within containers
func NewDockerExecutor(mountDirectory, platform string) Executor {

	var userID, groupID string

	if me, err := user.Current(); err == nil {
		userID = me.Uid
		groupID = me.Gid
	}

	return &dockerExecutor{
		MountDirectory: mountDirectory,
		UserID:         userID,
		GroupID:        groupID,
		ExecDirectory:  DefaultDockerExecutorDir,
		Platform:       platform,
	}
}

type dockerExecutor struct {
	MountDirectory string
	UserID         string
	GroupID        string
	ExecDirectory  string
	Platform       string
}

// Execute runs a command in a container
func (e *dockerExecutor) Execute(ctx context.Context, opts ExecOpts) error {

	if opts.Image == "" {
		return errors.New("Docker image is not specified")
	}
	mountDir, err := filepath.Abs(e.MountDirectory)
	if err != nil {
		return fmt.Errorf("Invalid mount dir %s: %s", e.MountDirectory, err)
	}
	workingDir := opts.WorkingDirectory
	if workingDir == "" {
		workingDir = "."
	}
	workingAbsDir, err := filepath.Abs(workingDir)
	if err != nil {
		return fmt.Errorf("Invalid working dir %s: %s", workingDir, err)
	}
	workingRelDir, err := filepath.Rel(mountDir, workingAbsDir)
	if err != nil {
		return fmt.Errorf("Failed to get relative dir: %s", err)
	}

	args := []string{
		"run",
		"-i",
		"--rm",
		"--volume",
		fmt.Sprintf("%s:%s", mountDir, e.ExecDirectory),
		"--workdir",
		path.Join(e.ExecDirectory, workingRelDir),
		"-e",
		fmt.Sprintf("HOME=%s", e.ExecDirectory),
		"-e",
		fmt.Sprintf("GOPATH=%s", path.Join(e.ExecDirectory, ".go")),
	}
	if opts.Name != "" {
		args = extendSlice(args, "--name", opts.Name)
	}
	if e.UserID != "" && e.GroupID != "" {
		args = extendSlice(args, "--user", fmt.Sprintf("%s:%s", e.UserID, e.GroupID))
	}
	if XDGCache() != "" {
		args = extendSlice(args, "-e", fmt.Sprintf("XDG_CACHE_HOME=%s", path.Join(e.ExecDirectory, ".cache")))
	}
	if os.Getenv("GOPROXY") != "" {
		args = extendSlice(args, "-e", fmt.Sprintf("GOPROXY=%s", os.Getenv("GOPROXY")))
	}
	if os.Getenv("ZIM_DOCKER_IN_DOCKER") == "1" {
		args = extendSlice(args, "--group-add", "root")
		args = extendSlice(args, "--volume", "/var/run/docker.sock:/var/run/docker.sock")
	}
	if e.Platform != "" {
		args = extendSlice(args, "--platform", e.Platform)
	}
	for _, envVar := range opts.Env {
		args = extendSlice(args, "-e", envVar)
	}
	args = extendSlice(args, opts.Image, "bash", "-e")
	if opts.Debug {
		args = extendSlice(args, "-x")
	}

	dockerCmd := exec.CommandContext(ctx, "docker", args...)
	dockerCmd.Stdout = getWriter(opts.Stdout, os.Stdout)
	dockerCmd.Stderr = getWriter(opts.Stderr, os.Stderr)

	stdin, err := dockerCmd.StdinPipe()
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
		fmt.Fprintln(cmdOut, "dbg:", debugColor(strings.Join(dockerCmd.Args, " ")))
	}
	cmdColor := color.New(color.FgCyan).SprintFunc()
	fmt.Fprintln(cmdOut, "cmd:", cmdColor(opts.Command))

	// Notice if the context was canceled and call "docker rm" on the
	// container since it continues to run otherwise. Better way possible?
	done := make(chan bool)
	defer close(done)
	go func() {
		select {
		// Run finished normally
		case <-done:
			return
		// Kill container since the context was canceled
		case <-ctx.Done():
			if opts.Name != "" {
				killContainerWithName(opts.Name)
			}
		}
	}()

	return dockerCmd.Run()
}

func (e *dockerExecutor) UsesDocker() bool {
	return true
}

func (e *dockerExecutor) ExecutorPath(hostPath string) (string, error) {
	if !filepath.IsAbs(hostPath) {
		return "", fmt.Errorf("A relative path was incorrectly passed: %s", hostPath)
	}
	relPath, err := filepath.Rel(e.MountDirectory, hostPath)
	if err != nil {
		return "", err
	}
	// Append the relative path to the base directory within the container.
	// For example {repo_path}/src/foo as the `hostPath` could be translated
	// to /build/src/foo within the container -- {e.ExecDirectory}/src/foo.
	return filepath.Join(e.ExecDirectory, relPath), nil
}

func extendSlice(s []string, item ...string) []string {
	s = append(s, item...)
	return s
}

func killContainerWithName(name string) error {
	return exec.Command("docker", "rm", "-f", name).Run()
}
