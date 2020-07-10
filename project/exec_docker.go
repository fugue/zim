package project

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
)

// NewDockerExecutor returns an Executor that runs commands within containers
func NewDockerExecutor(mountDirectory string) Executor {

	var userID, groupID string

	if me, err := user.Current(); err == nil {
		userID = me.Uid
		groupID = me.Gid
	}

	return &dockerExecutor{
		MountDirectory: mountDirectory,
		UserID:         userID,
		GroupID:        groupID,
	}
}

type dockerExecutor struct {
	MountDirectory string
	UserID         string
	GroupID        string
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

	// Replace newlines with semicolons if the command is multiline
	commands := strings.Split(strings.TrimSpace(opts.Command), "\n")
	commandText := strings.Join(commands, "; ")

	args := []string{
		"run",
		"--rm",
		"-t",
		"--volume",
		fmt.Sprintf("%s:/build", mountDir),
		"--workdir",
		path.Join("/build", workingRelDir),
		"-e",
		"HOME=/build",
		"-e",
		"GOPATH=/build/.go",
	}
	if opts.Name != "" {
		args = extendSlice(args, "--name", opts.Name)
	}
	if e.UserID != "" && e.GroupID != "" {
		args = extendSlice(args, "--user", fmt.Sprintf("%s:%s", e.UserID, e.GroupID))
	}
	if XDGCache() != "" {
		args = extendSlice(args, "-e", "XDG_CACHE_HOME=/build/.cache")
	}
	if os.Getenv("GOPROXY") != "" {
		args = extendSlice(args, "-e", fmt.Sprintf("GOPROXY=%s", os.Getenv("GOPROXY")))
	}
	if os.Getenv("ZIM_DOCKER_IN_DOCKER") == "1" {
		args = extendSlice(args, "--group-add", "root")
		args = extendSlice(args, "--volume", "/var/run/docker.sock:/var/run/docker.sock")
	}
	for _, envVar := range opts.Env {
		args = extendSlice(args, "-e", envVar)
	}
	args = extendSlice(args, opts.Image, "bash", "-e", "-c", commandText)

	dockerCmd := exec.CommandContext(ctx, "docker", args...)
	dockerCmd.Stdout = getWriter(opts.Stdout, os.Stdout)
	dockerCmd.Stderr = getWriter(opts.Stderr, os.Stderr)

	// Show the command to be executed to the user
	cmdOut := getWriter(opts.Cmdout, os.Stdout)
	if opts.Debug {
		debugColor := color.New(color.FgYellow).SprintFunc()
		fmt.Fprintln(cmdOut, "dbg:", debugColor(dockerCmd.Args))
	}
	cmdColor := color.New(color.FgCyan).SprintFunc()
	fmt.Fprintln(cmdOut, "cmd:", cmdColor(commandText))

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

func extendSlice(s []string, item ...string) []string {
	s = append(s, item...)
	return s
}

func killContainerWithName(name string) error {
	return exec.Command("docker", "rm", "-f", name).Run()
}
