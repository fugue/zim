// +build dockertest

package project

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecDocker(t *testing.T) {
	fmt.Println("OK")

	tmpDir, err := ioutil.TempDir("", "zim-test-")
	require.Nil(t, err)

	stdout := &bytes.Buffer{}

	exec := NewDockerExecutor(tmpDir)
	err = exec.Execute(context.Background(), ExecOpts{
		Name:             "test",
		Command:          "pwd && ls",
		Image:            "fugue2/builder:0.0.3",
		Stdout:           stdout,
		Stderr:           stdout,
		WorkingDirectory: tmpDir,
	})
	require.Nil(t, err)

	fmt.Println("OUT:", stdout.String())
}

func TestExecDockerMultiline(t *testing.T) {
	fmt.Println("OK")

	tmpDir, err := ioutil.TempDir("", "zim-test-")
	require.Nil(t, err)

	stdout := &bytes.Buffer{}

	exec := NewDockerExecutor(tmpDir)
	err = exec.Execute(context.Background(), ExecOpts{
		Name: "test",
		Command: `echo foo
# This is a comment.
echo bar
`,
		Image:            "fugue2/builder:0.0.3",
		Stdout:           stdout,
		Stderr:           stdout,
		WorkingDirectory: tmpDir,
	})
	require.Nil(t, err)

	require.Equal(t, "foo\nbar\n", stdout.String())
}
